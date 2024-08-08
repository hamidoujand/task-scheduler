// Package redis contains CRUD functionlaity around worker
package redis

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/business/domain/worker"
	"github.com/redis/go-redis/v9"
)

const entity = "worker"

// Repository represents set of APIs related to worker's CRUD
type Repository struct {
	redisClient *redis.Client
}

// NewRepository init a redis repo with the given client.
func NewRepository(c *redis.Client) *Repository {
	return &Repository{
		redisClient: c,
	}
}

// Create inserts a worker into redis or return possible error.
func (r *Repository) Create(ctx context.Context, w worker.Worker) error {
	key := entity + ":" + w.ID.String()

	redisWorker := fromWorkerService(w)
	if _, err := r.redisClient.Pipelined(ctx, func(p redis.Pipeliner) error {
		r.redisClient.HSet(ctx, key, "id", redisWorker.ID)
		r.redisClient.HSet(ctx, key, "load", redisWorker.Load)
		r.redisClient.HSet(ctx, key, "status", redisWorker.Status)
		return nil
	}); err != nil {
		return fmt.Errorf("pipelined: %w", err)
	}

	return nil
}

// GetByID fetches the worker from redis by id or returns possible error.
func (r *Repository) GetByID(ctx context.Context, workerId uuid.UUID) (worker.Worker, error) {
	key := entity + ":" + workerId.String()

	//check the existence
	exists, err := r.redisClient.Exists(ctx, key).Result()
	if err != nil {
		return worker.Worker{}, fmt.Errorf("key existence check: %w", err)
	}

	//not found
	if exists == 0 {
		return worker.Worker{}, redis.Nil
	}

	var rdWorker redisWorker
	if err := r.redisClient.HGetAll(ctx, key).Scan(&rdWorker); err != nil {
		return worker.Worker{}, fmt.Errorf("scan: %w", err)
	}

	return rdWorker.toServiceWorker(), nil
}

// DeleteById removes a worker from repo with id or returns possible error.
func (r *Repository) DeleteById(ctx context.Context, workerId uuid.UUID) error {
	key := entity + ":" + workerId.String()

	counts, err := r.redisClient.Del(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("del: %w", err)
	}

	//not found
	if counts == 0 {
		return redis.Nil
	}

	return nil
}

// UpdateWorker updates the worker inside repo or returns possible error.
func (r *Repository) Update(ctx context.Context, w worker.Worker) error {
	key := entity + ":" + w.ID.String()

	redisWorker := fromWorkerService(w)

	if _, err := r.redisClient.Pipelined(ctx, func(p redis.Pipeliner) error {
		r.redisClient.HSet(ctx, key, "load", redisWorker.Load)
		r.redisClient.HSet(ctx, key, "status", redisWorker.Status)
		return nil
	}); err != nil {
		return fmt.Errorf("pipelined: %w", err)
	}

	return nil
}
