package redis_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	redisRepo "github.com/hamidoujand/task-scheduler/business/domain/scheduler/store/redis"
	"github.com/hamidoujand/task-scheduler/business/redistest"
	"github.com/redis/go-redis/v9"
)

func TestCreate(t *testing.T) {
	t.Parallel()
	client := redistest.NewRedisClient(t, context.Background(), "test_redis_create")
	repo := redisRepo.NewRepository(client)

	taskId := uuid.NewString()

	if err := repo.Create(context.Background(), taskId); err != nil {
		t.Fatalf("expected to insert a new record: %s", err)
	}
}

func TestGet(t *testing.T) {
	t.Parallel()
	client := redistest.NewRedisClient(t, context.Background(), "test_redis_get")
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
	t.Parallel()
	client := redistest.NewRedisClient(t, context.Background(), "test_redis_update")
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
	t.Parallel()
	client := redistest.NewRedisClient(t, context.Background(), "test_redis_delete")
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
