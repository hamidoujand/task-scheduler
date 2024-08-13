package scheduler_test

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/business/broker/rabbitmq"
	"github.com/hamidoujand/task-scheduler/business/brokertest"
	"github.com/hamidoujand/task-scheduler/business/dbtest"
	"github.com/hamidoujand/task-scheduler/business/domain/scheduler"
	redisRepo "github.com/hamidoujand/task-scheduler/business/domain/scheduler/store/redis"
	"github.com/hamidoujand/task-scheduler/business/domain/task"
	taskRepo "github.com/hamidoujand/task-scheduler/business/domain/task/store/postgres"
	"github.com/hamidoujand/task-scheduler/business/redistest"
	"github.com/hamidoujand/task-scheduler/foundation/logger"
)

func TestConsumeTasks(t *testing.T) {
	setups := setupTest(t, "test_consume_tasks")

	scheduler, err := scheduler.New(scheduler.Config{
		MaxRunningTask: 4,
		RabbitClient:   setups.rabbitC,
		Logger:         setups.logger,
		TaskService:    setups.taskService,
		RedisRepo:      setups.redisR,
	})

	if err != nil {
		t.Fatalf("expected to create a scheduler: %s", err)
	}

	now := time.Now()
	nu := task.NewTask{
		UserId:      uuid.New(),
		Command:     "date",
		Image:       "alpine:3.20",
		Environment: "APP_NAME=test",
		ScheduledAt: now,
	}
	//save it into db, since "scheduledAt" is "now" which is less than 1 min, the task service itself
	//enqueue this into "queue_tasks"
	tsk, err := setups.taskService.CreateTask(context.Background(), nu)
	if err != nil {
		t.Fatalf("expected to create the task: %s", err)
	}

	//creates a goroutines inside of it that is waiting for tasks
	if err := scheduler.ConsumeTasks(context.Background()); err != nil {
		t.Fatalf("expected to consume tasks: %s", err)
	}

	//creates a goroutine inside of it that waiting for success tasks
	if err := scheduler.OnTaskSuccess(context.Background()); err != nil {
		t.Fatalf("expected to run onTaskSuccess: %s", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	errChan := make(chan error, 1)
	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
		defer cancel()

		//this will watch db the results
		for attemp := 1; ; attemp++ {
			fetched, err := setups.taskService.GetTaskById(ctx, tsk.Id)
			if err != nil {
				errChan <- fmt.Errorf("get task by id: %w", err)
				return
			}
			if fetched.Status != task.StatusPending {
				if fetched.ErrMessage != "" {
					errChan <- fmt.Errorf("errorMessage=%s , got %s", "<empty>", fetched.ErrMessage)
					return
				}
				errChan <- nil
				return //success
			}
			time.Sleep(time.Second * time.Duration(attemp))
		}
	}()

	wg.Wait()
	//check errors
	if err := <-errChan; err != nil {
		t.Fatalf("expected the task to run successfully and get the results: %s", err)
	}
}

type setup struct {
	logger      *slog.Logger
	taskService *task.Service
	rabbitC     *rabbitmq.Client
	redisR      *redisRepo.Repository
}

func setupTest(t *testing.T, container string) setup {
	rabbitC := brokertest.NewTestClient(t, context.Background(), container+"_rebbitmq")
	redisC := redistest.NewRedisClient(t, context.Background(), container+"_redis")
	postgresC := dbtest.NewDatabaseClient(t, container+"_postgres")
	logger := logger.NewCustomLogger(slog.LevelInfo, false, slog.String("Env", "Test"))
	store := taskRepo.NewRepository(postgresC)
	redisRepo := redisRepo.NewRepository(redisC)
	tasksService, err := task.NewService(store, rabbitC)
	if err != nil {
		t.Fatalf("expected to create task service: %s", err)
	}

	return setup{
		logger:      logger,
		taskService: tasksService,
		rabbitC:     rabbitC,
		redisR:      redisRepo,
	}
}
