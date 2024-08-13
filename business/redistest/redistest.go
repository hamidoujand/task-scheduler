// Package brokertest provides stup and clean up for testing redis clients.
package redistest

import (
	"context"
	"testing"
	"time"

	"github.com/hamidoujand/task-scheduler/foundation/docker"
	"github.com/redis/go-redis/v9"
)

func NewRedisClient(t *testing.T, ctx context.Context, name string) *redis.Client {
	// setup
	image := "redis:latest"
	internalPort := "6379"
	c, err := docker.StartContainer(image, name, internalPort, nil, nil)
	if err != nil {
		t.Fatalf("expected to create a redis container: %s", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     c.HostPort,
		Password: "",
		DB:       0,
	})

	//retries
	//slow machine
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), time.Minute*2)
		defer cancel()
	}

	for attemp := 1; ; attemp++ {
		pingErr := client.Ping(ctx).Err()
		if pingErr == nil {
			break
		}
		time.Sleep(time.Millisecond * 100 * time.Duration(attemp))
		if ctx.Err() != nil {
			t.Fatalf("expected to ping redis: %s", pingErr)
		}
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
