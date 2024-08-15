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

func TestOnSuccess(t *testing.T) {
	setups := setupTest(t, "test_consume_tasks")

	scheduler, err := scheduler.New(scheduler.Config{
		MaxRunningTask:      4,
		RabbitClient:        setups.rabbitC,
		Logger:              setups.logger,
		TaskService:         setups.taskService,
		RedisRepo:           setups.redisR,
		MaxRetries:          1,
		MaxTimeForUpdateOps: time.Minute,
	})

	if err != nil {
		t.Fatalf("expected to create a scheduler: %s", err)
	}

	//creates a goroutines inside of it that is waiting for tasks
	if err := scheduler.ConsumeTasks(context.Background()); err != nil {
		t.Fatalf("expected to consume tasks: %s", err)
	}

	//creates a goroutine inside of it that waiting for success tasks
	if err := scheduler.OnTaskSuccess(); err != nil {
		t.Fatalf("expected to run onTaskSuccess: %s", err)
	}

	now := time.Now()
	nt1 := task.NewTask{
		UserId:      uuid.New(),
		Command:     "date",
		Image:       "alpine:3.20",
		Environment: "APP_NAME=test",
		ScheduledAt: now,
	}

	nt2 := task.NewTask{
		UserId:      uuid.New(),
		Command:     "pwd",
		Image:       "alpine:3.20",
		Environment: "APP_NAME=test",
		ScheduledAt: now,
	}

	nts := []task.NewTask{nt1, nt2}
	ids := make([]uuid.UUID, 0, len(nts))
	for _, nt := range nts {
		//save it into db, since "scheduledAt" is "now" which is less than 1 min, the task service itself
		//enqueue this into "queue_tasks"
		tsk, err := setups.taskService.CreateTask(context.Background(), nt)
		if err != nil {
			t.Fatalf("expected to create the task: %s", err)
		}
		ids = append(ids, tsk.Id)
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
			fetched1, err := setups.taskService.GetTaskById(ctx, ids[0])
			if err != nil {
				errChan <- fmt.Errorf("get task by id %q: %w", ids[0], err)
				return
			}

			fetched2, err := setups.taskService.GetTaskById(ctx, ids[1])
			if err != nil {
				errChan <- fmt.Errorf("get task by id %q: %w", ids[1], err)
				return
			}
			if (fetched1.Status != task.StatusPending) && (fetched2.Status != task.StatusPending) {
				t.Logf("command: %s Result: %s", fetched1.Command, fetched1.Result)
				t.Logf("command: %s Result: %s", fetched2.Command, fetched2.Result)
				if fetched1.ErrMessage != "" {
					errChan <- fmt.Errorf("errorMessage1=%s , got %s", "<empty>", fetched1.ErrMessage)
					return
				}

				if fetched2.ErrMessage != "" {
					errChan <- fmt.Errorf("errorMessage2=%s , got %s", "<empty>", fetched2.ErrMessage)
				}
				errChan <- nil
				return //success
			}
			time.Sleep(time.Second)
		}
	}()

	wg.Wait()
	//check errors
	if err := <-errChan; err != nil {
		t.Fatalf("expected the task to run successfully and get the results: %s", err)
	}

	if err := scheduler.Shutdown(context.Background()); err != nil {
		t.Fatalf("expected a clean shutdown: %s", err)
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
