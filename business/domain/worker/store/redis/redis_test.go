package redis_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/business/domain/worker"
	redisRepo "github.com/hamidoujand/task-scheduler/business/domain/worker/store/redis"
	"github.com/hamidoujand/task-scheduler/foundation/docker"
	"github.com/redis/go-redis/v9"
)

func TestCreate(t *testing.T) {
	client := setup(t, "test_create")
	repo := redisRepo.NewRepository(client)

	w := worker.Worker{
		ID:     uuid.New(),
		Status: worker.WorkerIdle,
		Load:   0,
	}

	if err := repo.Create(context.Background(), w); err != nil {
		t.Fatalf("expected the worker to be created: %s", err)
	}
}

func TestGetByID(t *testing.T) {
	client := setup(t, "test_get_by_id")
	repo := redisRepo.NewRepository(client)

	w := worker.Worker{
		ID:     uuid.New(),
		Status: worker.WorkerIdle,
		Load:   0,
	}

	if err := repo.Create(context.Background(), w); err != nil {
		t.Fatalf("expected the worker to be created: %s", err)
	}

	worker, err := repo.GetByID(context.Background(), w.ID)
	if err != nil {
		t.Fatalf("expected to fetch worker by id %q: %s", w.ID, err)
	}

	if worker.ID != w.ID {
		t.Errorf("id= %s, got %s", w.ID, worker.ID)
	}

	if worker.Load != w.Load {
		t.Errorf("load= %d, got %d", w.Load, worker.Load)
	}

	if worker.Status != w.Status {
		t.Errorf("status= %s, got %s", w.Status, worker.Status)
	}

	//random Id
	worker, err = repo.GetByID(context.Background(), uuid.New())

	if err == nil {
		t.Fatal("expected to get an error while asking for a random worker")
	}

	if !errors.Is(err, redis.Nil) {
		t.Errorf("error=%v, got %v", redis.Nil, err)
	}
}

func TestDeleteById(t *testing.T) {
	client := setup(t, "test_delete_by_id")
	repo := redisRepo.NewRepository(client)

	w := worker.Worker{
		ID:     uuid.New(),
		Status: worker.WorkerIdle,
		Load:   0,
	}

	if err := repo.Create(context.Background(), w); err != nil {
		t.Fatalf("expected the worker to be created: %s", err)
	}

	if err := repo.DeleteById(context.Background(), w.ID); err != nil {
		t.Fatalf("expected to delete worker with id %s: %s", w.ID, err)
	}

	//try to fetch it
	_, err := repo.GetByID(context.Background(), w.ID)
	if err == nil {
		t.Fatalf("expected the worker with id %s to be deleted", w.ID)
	}

	//random id
	err = repo.DeleteById(context.Background(), uuid.New())
	if err == nil {
		t.Fatalf("expected to get an error while deleting random worker")
	}

	if !errors.Is(err, redis.Nil) {
		t.Fatalf("error= %v, got %v", redis.Nil, err)
	}
}

func TestUpdate(t *testing.T) {
	client := setup(t, "test_update")
	repo := redisRepo.NewRepository(client)

	w := worker.Worker{
		ID:     uuid.New(),
		Status: worker.WorkerIdle,
		Load:   0,
	}

	if err := repo.Create(context.Background(), w); err != nil {
		t.Fatalf("expected the worker to be created: %s", err)
	}

	fetched, err := repo.GetByID(context.Background(), w.ID)
	if err != nil {
		t.Fatalf("expected to fetch worker by id %q: %s", w.ID, err)
	}

	fetched.Load = 10
	fetched.Status = worker.WorkerBusy
	if err := repo.Update(context.Background(), fetched); err != nil {
		t.Fatalf("expected to update worker: %s", err)
	}

	//fetch it again
	updated, err := repo.GetByID(context.Background(), w.ID)
	if err != nil {
		t.Fatalf("expected to fetch worker by id %q: %s", w.ID, err)
	}
	if updated.Load != fetched.Load {
		t.Errorf("load= %d, got %d", fetched.Load, updated.Load)
	}

	if updated.Status != fetched.Status {
		t.Errorf("status= %d, got %d", fetched.Status, updated.Status)
	}
}

func setup(t *testing.T, name string) *redis.Client {
	// setup
	image := "redis:latest"
	internalPort := "6379"
	c, err := docker.StartContainer(image, name, internalPort, nil, nil)
	if err != nil {
		t.Fatalf("expected to create a redis container: %s", err)
	}

	//slow machine
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()

	client := redis.NewClient(&redis.Options{
		Addr:     c.HostPort,
		Password: "",
		DB:       0,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		t.Errorf("expected to ping redis engine: %s", err)
	}
	//teardown
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("failed to close redis client: %s", err)
		}

		if err := c.Stop(); err != nil {
			t.Errorf("failed to stop container %s: %s", c.Id, err)
		}
	})

	return client
}
