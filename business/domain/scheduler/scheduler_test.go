package scheduler_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"strings"
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
	"github.com/redis/go-redis/v9"
)

const maxRetries = 1

func TestOnSuccess(t *testing.T) {
	t.Parallel()
	setups := setupTest(t, "test_on_success_tasks")

	scheduler, err := scheduler.New(scheduler.Config{
		MaxRunningTask:      4,
		RabbitClient:        setups.rabbitC,
		Logger:              setups.logger,
		TaskService:         setups.taskService,
		RedisRepo:           setups.redisR,
		MaxRetries:          maxRetries,
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

				t.Logf("\ncommand: %s\nResult: %s\n", fetched1.Command, fetched1.Result)
				t.Logf("\ncommand: %s\nResult: %s\n", fetched2.Command, fetched2.Result)

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

func TestOnFailure(t *testing.T) {
	t.Parallel()
	setups := setupTest(t, "test_on_failure_tasks")

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

	if err := scheduler.OnTaskRetry(); err != nil {
		t.Fatalf("expected to start onTasksRetry: %s", err)
	}

	if err := scheduler.OnTaskFailure(); err != nil {
		t.Fatalf("expected to start onTaskFailure: %s", err)
	}

	now := time.Now()
	nt1 := task.NewTask{
		UserId:      uuid.New(),
		Command:     "invalid-command",
		Image:       "alpine:3.20",
		Environment: "APP_NAME=test",
		ScheduledAt: now,
	}

	nt2 := task.NewTask{
		UserId:      uuid.New(),
		Command:     "ls",
		Args:        []string{"nowhere"},
		Image:       "alpine:3.20",
		Environment: "APP_NAME=test",
		ScheduledAt: now,
	}

	nts := []task.NewTask{nt1, nt2}
	ids := make([]uuid.UUID, 0, len(nts))

	for _, nt := range nts {
		tsk, err := setups.taskService.CreateTask(context.Background(), nt)
		if err != nil {
			t.Fatalf("expected to create task: %s", err)
		}
		ids = append(ids, tsk.Id)
	}

	var redisMonitor sync.WaitGroup
	redisErrs := make(chan error, 1)

	redisMonitor.Add(1)
	go func() {
		defer redisMonitor.Done()
		//keep watching redis till the task hit 1 retry
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
		defer cancel()

		for attemp := 1; ; attemp++ {
			retries1, err := setups.redisR.Get(ctx, ids[0].String())
			if err != nil {
				//not-found is ok
				if errors.Is(err, redis.Nil) {
					time.Sleep(time.Second)
					continue
				} else {
					redisErrs <- fmt.Errorf("get: %w", err)
					return
				}
			}

			retries2, err := setups.redisR.Get(ctx, ids[1].String())
			if err != nil {
				//not-found is ok
				if errors.Is(err, redis.Nil) {
					time.Sleep(time.Second)
					continue
				} else {
					redisErrs <- fmt.Errorf("get: %w", err)
					return
				}
			}

			if (retries1 == maxRetries) && (retries2 == maxRetries) {
				//success
				redisErrs <- nil
				return
			}
			time.Sleep(time.Second)
		}
	}()

	redisMonitor.Wait()

	if err := <-redisErrs; err != nil {
		t.Fatalf("expected onRetry to handle retries: %s", err)
	}

	var taskServiceMonitor sync.WaitGroup
	taskErrs := make(chan error, 1)

	taskServiceMonitor.Add(1)
	go func() {
		//check db for failure
		defer taskServiceMonitor.Done()
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
		defer cancel()

		for attemp := 1; ; attemp++ {
			fetched1, err := setups.taskService.GetTaskById(ctx, ids[0])
			if err != nil {
				taskErrs <- fmt.Errorf("getTaskById: %w", err)
				return
			}

			fetched2, err := setups.taskService.GetTaskById(ctx, ids[1])
			if err != nil {
				taskErrs <- fmt.Errorf("getTaskById: %w", err)
				return
			}

			if (fetched1.Status != task.StatusPending) && (fetched2.Status != task.StatusPending) {
				if fetched1.Status != task.StatusFailed {
					taskErrs <- fmt.Errorf("status1=%s, got %s", task.StatusFailed, fetched1.Status)
					return
				}
				if fetched2.Status != task.StatusFailed {
					taskErrs <- fmt.Errorf("status2=%s, got %s", task.StatusFailed, fetched2.Status)
					return
				}

				if fetched1.ErrMessage == "" {
					taskErrs <- fmt.Errorf("errorMessage1=%s, got %s", "<empty>", fetched1.ErrMessage)
					return
				}

				if fetched2.ErrMessage == "" {
					taskErrs <- fmt.Errorf("errorMessage1=%s, got %s", "<empty>", fetched2.ErrMessage)
					return
				}

				t.Logf("\ncommand: %s\nErrorMessage: %s\n", fetched1.Command, fetched1.ErrMessage)
				t.Logf("\ncommand: %s\nErrorMessage: %s\n", fetched2.Command, fetched2.ErrMessage)

				taskErrs <- nil
				return
			}
			time.Sleep(time.Second)
		}
	}()

	taskServiceMonitor.Wait()
	if err := <-taskErrs; err != nil {
		t.Fatalf("expected the onFailure handle the failed task: %s", err)
	}

	if err := scheduler.Shutdown(context.Background()); err != nil {
		t.Fatalf("expected to get a clean shutdown: %s", err)
	}
}

func TestMonitorScheduledTasks(t *testing.T) {
	t.Parallel()
	setups := setupTest(t, "test_monitor_scheduled_tasks")

	scheduler, err := scheduler.New(scheduler.Config{
		MaxRunningTask:      4,
		RabbitClient:        setups.rabbitC,
		Logger:              setups.logger,
		TaskService:         setups.taskService,
		RedisRepo:           setups.redisR,
		MaxRetries:          maxRetries,
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := scheduler.MonitorScheduledTasks(ctx); err != nil {
		t.Fatalf("expected to start monitor service: %s", err)
	}

	//insert few tasks with "+1:30" deadline
	now := time.Now().Add(time.Second * 90)
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

				t.Logf("\ncommand: %s\nResult: %s\n", fetched1.Command, fetched1.Result)
				t.Logf("\ncommand: %s\nResult: %s\n", fetched2.Command, fetched2.Result)

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
	alpha := "abcdefghijklmnopqrstuvwxyz"
	var builder strings.Builder
	for range 8 {
		idx := rand.IntN(len(alpha))
		builder.WriteByte(alpha[idx])
	}

	salt := builder.String()

	rabbitC := brokertest.NewTestClient(t, context.Background(), container+"_rebbitmq_"+salt)
	redisC := redistest.NewRedisClient(t, context.Background(), container+"_redis_"+salt)
	postgresC := dbtest.NewDatabaseClient(t, container+"_postgres_"+salt)
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
