// Package scheduler provides functionality to schedule tasks.
package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/hamidoujand/task-scheduler/business/broker/rabbitmq"
	redisRepo "github.com/hamidoujand/task-scheduler/business/domain/scheduler/store/redis"
	"github.com/hamidoujand/task-scheduler/business/domain/task"
	"github.com/hamidoujand/task-scheduler/business/worker"
	"github.com/hamidoujand/task-scheduler/foundation/docker"
	"github.com/redis/go-redis/v9"
)

const (
	queueSuccess  = "queue_success"
	queueFailed   = "queue_failed"
	queueTasks    = "queue_tasks"
	maxNumRetries = 3
)

// Scheduler represents set of APIs used for scheduling tasks using worker.
type Scheduler struct {
	rClient     *rabbitmq.Client
	redisRepo   *redisRepo.Repository
	worker      *worker.Worker
	logger      *slog.Logger
	taskService *task.Service
}

// Config represents all of required configuration to create a scheduler.
type Config struct {
	RabbitClient   *rabbitmq.Client
	Logger         *slog.Logger
	TaskService    *task.Service
	RedisRepo      *redisRepo.Repository
	MaxRunningTask int
}

// New creates a scheduler.
func New(conf Config) (*Scheduler, error) {
	//register queues
	queues := [...]string{queueSuccess, queueFailed, queueTasks}
	for _, name := range queues {
		if err := conf.RabbitClient.DeclareQueue(name); err != nil {
			return nil, fmt.Errorf("declare queue: %w", err)
		}
	}

	worker, err := worker.New(conf.MaxRunningTask)
	if err != nil {
		return nil, fmt.Errorf("new worker: %w", err)
	}

	return &Scheduler{
		rClient:     conf.RabbitClient,
		worker:      worker,
		logger:      conf.Logger,
		taskService: conf.TaskService,
		redisRepo:   conf.RedisRepo,
	}, nil
}

// ConsumeTasks will listen to "tasks" queue for new tasks.
func (s *Scheduler) ConsumeTasks(ctx context.Context) error {
	msgs, err := s.rClient.Consumer(queueTasks)
	if err != nil {
		return fmt.Errorf("consumer: %w", err)
	}

	go func() {
		for msg := range msgs {
			if err := msg.Ack(false); err != nil {
				//handle error
				s.logger.Error("ack", "status", "failed", "msg", err.Error())
				continue
			}

			tsk, err := s.parseTask(msg.Body)
			if err != nil {
				s.logger.Error("parse task", "status", "failed", "msg", err.Error())
				continue
			}

			if err := s.submitTask(ctx, tsk); err != nil {
				//handle error
				s.logger.Error("submit task", "status", "failed", "msg", err.Error())
				continue
			}

		}
	}()

	return nil
}

// OnTaskSuccess handles the saving task into task service.
func (s *Scheduler) OnTaskSuccess(ctx context.Context) error {
	msgs, err := s.rClient.Consumer(queueSuccess)
	if err != nil {
		return fmt.Errorf("consumer: %w", err)
	}

	//consumer
	go func() {
		for msg := range msgs {
			//ack the message rightway
			if err := msg.Ack(false); err != nil {
				s.logger.Error("ack success message", "status", "failed to ack the message", "msg", err.Error())
				continue
			}

			tsk, err := s.parseTask(msg.Body)
			if err != nil {
				s.logger.Error("onTaskSuccess", "status", "failed to parse task", "msg", err.Error())
				continue
			}

			//save it into db
			ut := task.UpdateTask{
				Status: &tsk.Status,
				Result: &tsk.Result,
			}

			if _, err := s.taskService.UpdateTask(ctx, tsk, ut); err != nil {
				s.logger.Error("update succeeded task", "status", "failed to update task into task service", "msg", err.Error())
				continue
			}
		}
	}()
	return nil
}

// OnTaskFailure handles the retries and eventually saving task into task service.
func (s *Scheduler) OnTaskFailure(ctx context.Context) error {
	msgs, err := s.rClient.Consumer(queueFailed)
	if err != nil {
		return fmt.Errorf("consumer: %w", err)
	}

	//consumer
	go func() {
		for msg := range msgs {
			//ack the message when you got it
			if err := msg.Ack(false); err != nil {
				//handle it with logging
				s.logger.Error("ack", "status", "failed to ack the message", "msg", err.Error())
				continue
			}

			//parse it
			tsk, err := s.parseTask(msg.Body)
			if err != nil {
				//handle it with logging
				s.logger.Error("parse failed task", "status", "failed parsing the failed task", "msg", err.Error())
				continue
			}

			retries, err := s.redisRepo.Get(ctx, tsk.Id.String())
			if err != nil {
				if errors.Is(err, redis.Nil) {
					//first time failure
					retries = 0
				} else {
					//handle error by logging
					s.logger.Error("redis get", "status", "failed to fetch task from repo", "msg", err.Error())
					continue
				}
			}

			//inc
			retries++

			if retries > maxNumRetries {
				//save it as failed task into db
				ut := task.UpdateTask{
					Status:     &tsk.Status,
					ErrMessage: &tsk.ErrMessage,
				}
				if _, err := s.taskService.UpdateTask(ctx, tsk, ut); err != nil {
					s.logger.Error("update task", "status", "failed to update the failed task into db", "msg", err.Error())
					continue
				}
			} else {
				//send it for retry
				if err := s.publishTask(tsk, queueTasks); err != nil {
					s.logger.Error("task retry", "status", "failed to publish the task for retries into queue_tasks", "msg", err.Error())
					continue
				}

				//update redis info
				if err := s.redisRepo.Update(ctx, tsk.Id.String(), retries); err != nil {
					s.logger.Error("redis retry", "status", "failed to save retries into redis", "msg", err.Error())
					continue
				}
			}
		}
	}()

	return nil
}

// MonitorScheduledTasks fetches all of the tasks that have less than or equal
// to one minute to their scheduledAt deadline every one minute.
func (s *Scheduler) MonitorScheduledTasks(ctx context.Context) error {
	panic("")
}

// submitTask uses the worker to execute the task inside of it.
func (s *Scheduler) submitTask(ctx context.Context, tsk task.Task) error {
	//going to execute inside of goroutine.

	timeTillExecution := time.Until(tsk.ScheduledAt)
	if timeTillExecution > 0 {
		//need to wait
		time.Sleep(timeTillExecution)
	}

	executer := func(ctx context.Context) {
		var builder strings.Builder
		for _, env := range strings.Split(tsk.Environment, " ") {
			builder.WriteString("-e ")
			builder.WriteString(env)
			builder.WriteByte(' ')
		}

		dockerArgs := []string{builder.String()}

		output, err := docker.RunCommand(tsk.Image, tsk.Command, dockerArgs, tsk.Args)
		if err != nil {
			//failed, enqueue into queue_failed
			tsk.ErrMessage = output
			tsk.Status = task.StatusFailed

			if err := s.publishTask(tsk, queueFailed); err != nil {
				//handle error in goroutine
				s.logger.Error("publish failed task", "status", "failed to publish into queue_failed", "msg", err.Error())
				return
			}

		} else {
			//success, enqueue into queue_success
			tsk.Result = output
			tsk.Status = task.StatusCompleted

			if err := s.publishTask(tsk, queueSuccess); err != nil {
				//handle error in goroutine
				s.logger.Error("publish success task", "status", "failed to publish into queue_success", "msg", err.Error())
				return
			}
		}
	}
	if _, err := s.worker.Start(ctx, executer); err != nil {
		return fmt.Errorf("start worker: %w", err)
	}

	return nil
}

func (s *Scheduler) publishTask(tsk task.Task, queue string) error {
	bs, err := json.Marshal(tsk)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := s.rClient.Publish(queue, bs); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	return nil
}

func (s *Scheduler) parseTask(bs []byte) (task.Task, error) {
	var tsk task.Task
	if err := json.Unmarshal(bs, &tsk); err != nil {
		//handle error
		return task.Task{}, fmt.Errorf("unmarshal task: %s", err)
	}
	return tsk, nil
}
