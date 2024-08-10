package redis_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	redisRepo "github.com/hamidoujand/task-scheduler/business/domain/scheduler/store/redis"
	"github.com/hamidoujand/task-scheduler/foundation/docker"
	"github.com/redis/go-redis/v9"
)

func TestCreate(t *testing.T) {
	client := setup(t, "test_create")
	repo := redisRepo.NewRepository(client)

	taskId := uuid.NewString()

	if err := repo.Create(context.Background(), taskId); err != nil {
		t.Fatalf("expected to insert a new record: %s", err)
	}
}

func TestGet(t *testing.T) {
	client := setup(t, "test_get")
	repo := redisRepo.NewRepository(client)

	taskId := uuid.NewString()

	if err := repo.Create(context.Background(), taskId); err != nil {
		t.Fatalf("expected to insert a new record: %s", err)
	}

	retries, err := repo.Get(context.Background(), taskId)
	if err != nil {
		t.Fatalf("expected to fetch retries for id %s: %s", taskId, err)
	}

	if retries != 0 {
		t.Fatalf("retries= %d, got %d", 0, retries)
	}

	//random id
	taskId = uuid.NewString()
	_, err = repo.Get(context.Background(), taskId)
	if err == nil {
		t.Fatal("expected to get error when passing random id")
	}

	if !errors.Is(err, redis.Nil) {
		t.Fatalf("expected error to be %v, got %v", redis.Nil, err)
	}
}

func TestUpdate(t *testing.T) {
	client := setup(t, "test_update")
	repo := redisRepo.NewRepository(client)

	taskId := uuid.NewString()

	if err := repo.Create(context.Background(), taskId); err != nil {
		t.Fatalf("expected to insert a new record: %s", err)
	}

	retries := 5
	if err := repo.Update(context.Background(), taskId, retries); err != nil {
		t.Fatalf("expected to update retries: %s", err)
	}

	fetched, err := repo.Get(context.Background(), taskId)
	if err != nil {
		t.Fatalf("expected to fetch retries for id %s: %s", taskId, err)
	}

	if retries != fetched {
		t.Fatalf("retries= %d, got %d", retries, fetched)
	}
}

func TestDelete(t *testing.T) {
	client := setup(t, "test_delete")
	repo := redisRepo.NewRepository(client)

	taskId := uuid.NewString()

	if err := repo.Create(context.Background(), taskId); err != nil {
		t.Fatalf("expected to insert a new record: %s", err)
	}

	if err := repo.Delete(context.Background(), taskId); err != nil {
		t.Fatalf("expected to delete task with id %s: %s", taskId, err)
	}

	_, err := repo.Get(context.Background(), taskId)
	if err == nil {
		t.Fatal("expeted to not get back a deleted record")
	}

	if !errors.Is(err, redis.Nil) {
		t.Fatalf("error = %v, got %v", redis.Nil, err)
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
		t.Fatalf("expected to ping redis engine: %s", err)
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
