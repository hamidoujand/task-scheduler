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
	"github.com/redis/go-redis/v9"
)

const (
	queueSuccess = "queue_success"
	queueFailed  = "queue_failed"
	queueTasks   = "queue_tasks"
	queueRetry   = "queue_retry"
)

// Scheduler represents set of APIs used for scheduling tasks using worker.
type Scheduler struct {
	rClient                 *rabbitmq.Client
	redisRepo               *redisRepo.Repository
	logger                  *slog.Logger
	taskService             *task.Service
	maxRetries              int
	maxTimeForUpdateOps     time.Duration
	maxTimeForTaskExecution time.Duration
	wg                      sync.WaitGroup
	mu                      sync.RWMutex
	sem                     chan struct{}
	shutdown                chan struct{}
	executers               map[string]context.CancelFunc
}

// Config represents all of required configuration to create a scheduler.
type Config struct {
	RabbitClient            *rabbitmq.Client
	Logger                  *slog.Logger
	TaskService             *task.Service
	RedisRepo               *redisRepo.Repository
	MaxRunningTask          int
	MaxRetries              int
	MaxTimeForUpdateOps     time.Duration
	MaxTimeForTaskExecution time.Duration
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

	if conf.MaxTimeForTaskExecution <= 0 {
		return nil, fmt.Errorf("max time for task execution must be greater than 0")
	}

	return &Scheduler{
		rClient:                 conf.RabbitClient,
		logger:                  conf.Logger,
		taskService:             conf.TaskService,
		redisRepo:               conf.RedisRepo,
		maxRetries:              conf.MaxRetries,
		sem:                     sem,
		shutdown:                make(chan struct{}),
		executers:               make(map[string]context.CancelFunc),
		maxTimeForUpdateOps:     conf.MaxTimeForUpdateOps,
		maxTimeForTaskExecution: conf.MaxTimeForTaskExecution,
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
func (s *Scheduler) ConsumeTasks() error {
	msgs, err := s.rClient.Consumer(queueTasks)
	if err != nil {
		return fmt.Errorf("create consumer: %w", err)
	}

	//consumer
	go func() {
		for msg := range msgs {
			select {
			case <-s.shutdown:
				//do not proccess messages any more.
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

				if err := s.submitTask(tsk); err != nil {
					s.logger.Error("consumeTasks", "status", "failed to submit task to executer", "msg", err)
					continue
				}
			}
		}
	}()

	return nil
}

func (s *Scheduler) submitTask(tsk task.Task) error {
	//wait for a semaphore
	select {
	case <-s.shutdown:
		return errors.New("received shutdown signal")
	case <-s.sem:
		//move on
	}

	executerId := uuid.NewString()

	//default we gave a 30 sec
	if s.maxTimeForTaskExecution < time.Second*30 {
		s.maxTimeForTaskExecution = time.Second * 30
	}

	//set a deadline
	deadline := time.Now().Add(s.maxTimeForTaskExecution)

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

		s.logger.Info("executer", "status", fmt.Sprintf("executing task with id %s", tsk.Id))

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
			//handle message.
			//we need to ignore shutdown in here in case an executer manages to be in middle of a publish.
			go s.handleSuccessMessage(msg)
		}
	}()

	return nil
}

// OnTaskFailure handles the failed tasks by updating them into task service.
func (s *Scheduler) OnTaskFailure() error {
	msgs, err := s.rClient.Consumer(queueFailed)
	if err != nil {
		return fmt.Errorf("create on failure consumer: %w", err)
	}

	//consumer
	go func() {
		for msg := range msgs {
			//we need this to ignore shutdown in case an executer manages to be in middle of a publish
			//so we able to update that task.
			go s.handleFailedMessage(msg)
		}
	}()

	return nil
}

// OnTaskRetry handles the retry of failed tasks or sending them for total failure.
func (s *Scheduler) OnTaskRetry() error {
	msgs, err := s.rClient.Consumer(queueRetry)
	if err != nil {
		return fmt.Errorf("creating on retry consumer: %w", err)
	}

	//consumer
	go func() {
		for msg := range msgs {
			select {
			case <-s.shutdown:
				//do not send for retry since there is not consumer listening anymore on "tasks queue".
				s.logger.Info("onTaskRetry", "status", "received shutdown signal", "msg", "shutting down")
				//update the task as failure since we can not retry it any more
				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					defer wg.Done()
					s.handleFailedMessage(msg)
				}()
				wg.Wait()

				return
			default:
				go s.handleRetryMessage(msg)
			}
		}
	}()

	return nil
}

// MonitorScheduledTasks fetches all of the tasks that have less than or equal
// to one minute to their scheduledAt deadline every one minute.
func (s *Scheduler) MonitorScheduledTasks() error {
	//monitor
	go func() {
		//this is a long-lived ctx used inside of the loop for any db operation, and
		//will be canceled as soon as we receive shutdown signal.
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			select {
			case <-s.shutdown:
				s.logger.Info("monitorScheduledTasks", "status", "received shutdown signal", "msg", "shutting down")
				return

			default:
				dueTasks, err := s.taskService.GetAllDueTasks(ctx)
				if err != nil {
					s.logger.Error("monitorScheduledTasks", "status", "failed to fetch due tasks", "msg", err)
					return
				}

				for _, tsk := range dueTasks {
					if err := s.publishTask(tsk, queueTasks); err != nil {
						s.logger.Error("monitorScheduledTasks", "status", "failed to publish task into tasks queue", "msg", err)
						continue
					}
				}
			}
		}

	}()

	return nil
}

func (s *Scheduler) handleRetryMessage(msg amqp091.Delivery) {
	if err := msg.Ack(false); err != nil {
		//handle error with logging them
		s.logger.Error("handleRetryMessage", "status", "failed to ack()", "msg", err)
		return
	}

	tsk, err := s.parseTask(msg.Body)
	if err != nil {
		s.logger.Error("handleRetryTask", "status", "failed to parse task from body", "msg", err)
		return
	}

	//default ctx for redis
	ctx, cancel := context.WithTimeout(context.Background(), s.maxTimeForUpdateOps)
	defer cancel()

	retries, err := s.redisRepo.Get(ctx, tsk.Id.String())
	if err != nil {
		//not found
		if errors.Is(err, redis.Nil) {
			retries = 0
		} else {
			//something went wrong
			s.logger.Error("handleRetryMessage", "status", "failed to fetch retries", "msg", err)
			return
		}
	}

	retries++
	if retries > s.maxRetries {
		//publish into failed queue
		if err := s.publishTask(tsk, queueFailed); err != nil {
			s.logger.Error("handleRetryMessage", "status", "failed to publish into queue_failed", "msg", err)
		}
		return
	}

	//update redis
	if err := s.redisRepo.Update(ctx, tsk.Id.String(), retries); err != nil {
		s.logger.Error("handleRetryMessage", "status", "failed to update retries", "msg", err)
		return
	}

	s.logger.Info("handleRetryMessage", "status", fmt.Sprintf("%d/%d: retrying to execute task %s", retries, s.maxRetries, tsk.Id))
	if err := s.publishTask(tsk, queueTasks); err != nil {
		s.logger.Error("handleRetryMessage", "status", "failed to send task for a retry", "msg", err)
		return
	}

}

func (s *Scheduler) handleFailedMessage(msg amqp091.Delivery) {
	if err := msg.Ack(false); err != nil {
		//handle error with logging them
		s.logger.Error("handleFailedMessage", "status", "failed to ack()", "msg", err)
		return
	}
	// parse the task from body
	tsk, err := s.parseTask(msg.Body)
	if err != nil {
		s.logger.Error("handleFailedMessage", "status", "failed to parse task from body", "msg", err)
		return
	}

	ut := task.UpdateTask{
		Status:     &tsk.Status,
		ErrMessage: &tsk.ErrMessage,
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.maxTimeForUpdateOps)
	defer cancel()

	if _, err := s.taskService.UpdateTask(ctx, tsk, ut); err != nil {
		s.logger.Error("handleFailedMessage", "status", "failed to update task inside of task service", "msg", err)
		return
	}
	s.logger.Info("handleFailedMessage", "status", fmt.Sprintf("task with id %s failed", tsk.Id))
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
	//log message
	s.logger.Info("handleSuccessMessage", "status", fmt.Sprintf("task with id %s completed", tsk.Id))
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
