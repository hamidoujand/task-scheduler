// Package redis provides CRUD functionality for scheduler.
package redis

import (
	"context"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

const entity = "tasks"

// Repository represents all of the APIs used for CRUD against redis.
type Repository struct {
	client *redis.Client
}

// NewRepository creates a new redis repository.
func NewRepository(c *redis.Client) *Repository {
	return &Repository{
		client: c,
	}
}

// Create inserts a new record into redis.
func (r *Repository) Create(ctx context.Context, taskId string) error {
	key := entity + ":" + taskId
	retries := 0

	err := r.client.HSet(ctx, key, "retries", retries).Err()
	if err != nil {
		return fmt.Errorf("hset: %w", err)
	}
	return nil
}

// Get returns the number of retries for this given task id
func (r *Repository) Get(ctx context.Context, taskId string) (int, error) {
	key := entity + ":" + taskId
	result, err := r.client.HGet(ctx, key, "retries").Result()
	if err != nil {
		return 0, fmt.Errorf("hget: %w", err)
	}

	retries, err := strconv.Atoi(result)
	if err != nil {
		return 0, fmt.Errorf("atoi: %w", err)
	}

	return retries, nil
}

// Update updates the retries of the given task id.
func (r *Repository) Update(ctx context.Context, taskId string, retries int) error {
	key := entity + ":" + taskId
	if err := r.client.HSet(ctx, key, "retries", retries).Err(); err != nil {
		return fmt.Errorf("hset: %w", err)
	}
	return nil
}

// Delete delete a record with the given taskId.
func (r *Repository) Delete(ctx context.Context, taskId string) error {
	key := entity + ":" + taskId

	results, err := r.client.Del(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("del: %w", err)
	}

	if results == 0 {
		return redis.Nil
	}

	return nil
}
