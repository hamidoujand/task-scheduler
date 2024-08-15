// Package scheduler provides functionality to schedule tasks.
package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/business/broker/rabbitmq"
	redisRepo "github.com/hamidoujand/task-scheduler/business/domain/scheduler/store/redis"
	"github.com/hamidoujand/task-scheduler/business/domain/task"
	"github.com/hamidoujand/task-scheduler/foundation/docker"
	"github.com/rabbitmq/amqp091-go"
)

const (
	queueSuccess = "queue_success"
	queueFailed  = "queue_failed"
	queueTasks   = "queue_tasks"
	queueRetry   = "queue_retry"
)

// Scheduler represents set of APIs used for scheduling tasks using worker.
type Scheduler struct {
	rClient             *rabbitmq.Client
	redisRepo           *redisRepo.Repository
	logger              *slog.Logger
	taskService         *task.Service
	maxRetries          int
	maxTimeForUpdateOps time.Duration
	wg                  sync.WaitGroup
	mu                  sync.RWMutex
	sem                 chan struct{}
	shutdown            chan struct{}
	executers           map[string]context.CancelFunc
}

// Config represents all of required configuration to create a scheduler.
type Config struct {
	RabbitClient        *rabbitmq.Client
	Logger              *slog.Logger
	TaskService         *task.Service
	RedisRepo           *redisRepo.Repository
	MaxRunningTask      int
	MaxRetries          int
	MaxTimeForUpdateOps time.Duration
}

// New creates a scheduler.
func New(conf Config) (*Scheduler, error) {
	//register queues
	queues := [...]string{queueSuccess, queueFailed, queueRetry, queueTasks}
	for _, name := range queues {
		if err := conf.RabbitClient.DeclareQueue(name); err != nil {
			return nil, fmt.Errorf("declare queue: %w", err)
		}
	}

	if conf.MaxRunningTask <= 0 {
		return nil, fmt.Errorf("max running tasks must be greater than 0")
	}

	sem := make(chan struct{}, conf.MaxRunningTask)
	for range conf.MaxRunningTask {
		sem <- struct{}{}
	}

	if conf.MaxRetries < 0 {
		return nil, fmt.Errorf("max retries must be greater or equal to 0: %d", conf.MaxRetries)
	}

	return &Scheduler{
		rClient:             conf.RabbitClient,
		logger:              conf.Logger,
		taskService:         conf.TaskService,
		redisRepo:           conf.RedisRepo,
		maxRetries:          conf.MaxRetries,
		sem:                 sem,
		shutdown:            make(chan struct{}),
		executers:           make(map[string]context.CancelFunc),
		maxTimeForUpdateOps: conf.MaxTimeForUpdateOps,
	}, nil
}

// Shutdown is going to provide a graceful shutdown to all currently running executers.
func (s *Scheduler) Shutdown(ctx context.Context) error {
	close(s.shutdown) //send signal to all WAITING executers that we are shutting down

	//take care running ones
	func() {
		//allows better lock handling
		s.mu.RLock()
		defer s.mu.RUnlock()

		for _, cancel := range s.executers {
			cancel()
		}
	}()

	//wait for all of the running ones to finish
	ch := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(ch)
	}()

	//waiting ...
	select {
	case <-ch:
		//all graceful shutdown
		return nil
	case <-ctx.Done():
		//forced shutdown
		return ctx.Err()
	}
}

// ConsumeTasks will listen to the "tasks" queue for new tasks.
func (s *Scheduler) ConsumeTasks(ctx context.Context) error {
	msgs, err := s.rClient.Consumer(queueTasks)
	if err != nil {
		return fmt.Errorf("create consumer: %w", err)
	}

	//consumer
	go func() {
		for msg := range msgs {
			select {
			case <-s.shutdown:
				//do not proccess messages
				return
			default:
				if err := msg.Ack(false); err != nil {
					s.logger.Error("consumeTasks", "status", "failed to ack()", "msg", err)
					continue
				}

				tsk, err := s.parseTask(msg.Body)
				if err != nil {
					s.logger.Error("consumeTasks", "status", "failed to parse task from message body", "msg", err)
					continue
				}

				if err := s.submitTask(ctx, tsk); err != nil {
					s.logger.Error("consumeTasks", "status", "failed to submit task to executer", "msg", err)
					continue
				}
			}
		}
	}()

	return nil
}

func (s *Scheduler) submitTask(ctx context.Context, tsk task.Task) error {
	//wait for a semaphore
	select {
	case <-s.shutdown:
		return errors.New("received shutdown signal")
	case <-ctx.Done(): //provided context is only used to get a semaphore not for running tasks.
		return ctx.Err()
	case <-s.sem:
		//move on
	}

	executerId := uuid.NewString()

	deadline, ok := ctx.Deadline()
	if !ok {
		//set a default deadline
		deadline = time.Now().Add(time.Minute)
	}

	//create a new ctx that only scheduler uses to control executers
	ctx, cancel := context.WithDeadline(context.Background(), deadline)

	//register executer
	func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.executers[executerId] = cancel
	}()

	//create the executer goroutine
	s.wg.Add(1)
	go func() {

		//cleanup
		defer func() {
			//call cancel from closure
			cancel()
			//remove it
			s.removeExecuter(executerId)
			s.wg.Done()
			//release semaphore
			s.sem <- struct{}{}
		}()

		//actual task running logic
		timeTillExecution := time.Until(tsk.ScheduledAt)
		if timeTillExecution > 0 {
			//sleep
			time.Sleep(timeTillExecution)
		}

		var builder strings.Builder
		for _, env := range strings.Split(tsk.Environment, " ") {
			builder.WriteString("-e ")
			builder.WriteString(env)
			builder.WriteByte(' ')
		}

		dockerArgs := []string{builder.String()}

		output, err := docker.RunCommand(ctx, tsk.Image, tsk.Command, dockerArgs, tsk.Args)

		if err != nil {
			//failed
			tsk.ErrMessage = err.Error()
			tsk.Status = task.StatusFailed
			// publish task for retry queue
			if err := s.publishTask(tsk, queueRetry); err != nil {
				//logging is our error handler right now
				s.logger.Error("submitTask", "status", fmt.Sprintf("failed to publish task %s to retry queue", tsk.Id), "msg", err)
				return
			}

		} else {
			//success
			tsk.Result = output
			tsk.Status = task.StatusCompleted
			//publish task for success queue
			if err := s.publishTask(tsk, queueSuccess); err != nil {
				s.logger.Error("submitTask", "status", fmt.Sprintf("failed to publish task %s to success queue", tsk.Id), "msg", err)
				return
			}
		}
	}()
	return nil
}

// OnTaskSuccess handles the saving task into task service.
func (s *Scheduler) OnTaskSuccess() error {
	msgs, err := s.rClient.Consumer(queueSuccess)
	if err != nil {
		return fmt.Errorf("creating consumer: %w", err)
	}

	//consumer
	go func() {
		for msg := range msgs {
			select {
			case <-s.shutdown:
				s.logger.Info("onTaskSuccess", "status", "received shutdown signal", "msg", "shutting down")
				return
			default:
				//handle message
				go s.handleSuccessMessage(msg)
			}
		}
	}()

	return nil
}

// OnTaskFailure handles the retries and eventually saving task into task service.
func (s *Scheduler) OnTaskFailure() error {
	panic("")
}

func (s *Scheduler) handleSuccessMessage(msg amqp091.Delivery) {
	if err := msg.Ack(false); err != nil {
		//handle error with logging them
		s.logger.Error("handleSuccessMessage", "status", "failed to ack()", "msg", err)
		return
	}

	//parse the task from body
	tsk, err := s.parseTask(msg.Body)
	if err != nil {
		s.logger.Error("handleSuccessMessage", "status", "failed to parse task from body", "msg", err)
		return
	}

	ut := task.UpdateTask{
		Status: &tsk.Status,
		Result: &tsk.Result,
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.maxTimeForUpdateOps)
	defer cancel()

	if _, err := s.taskService.UpdateTask(ctx, tsk, ut); err != nil {
		s.logger.Error("handleSuccessMessage", "status", "failed to update task inside of task service", "msg", err)
		return
	}
}

func (s *Scheduler) removeExecuter(exId string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.executers, exId)
}

func (s *Scheduler) publishTask(tsk task.Task, queue string) error {
	bs, err := json.Marshal(tsk)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := s.rClient.Publish(queue, bs); err != nil {
		return fmt.Errorf("publish task to queue %s: %w", queue, err)
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
