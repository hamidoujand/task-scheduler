// Package brokertest provides stup and clean up for testing rabbitmq.
package brokertest

import (
	"context"
	"testing"
	"time"

	"github.com/hamidoujand/task-scheduler/business/broker/rabbitmq"
	"github.com/hamidoujand/task-scheduler/foundation/docker"
)

func NewTestClient(t *testing.T, ctx context.Context, containerName string) *rabbitmq.Client {
	image := "rabbitmq:3.13.6"

	c, err := docker.StartContainer(image, containerName, "5672", nil, nil)
	if err != nil {
		t.Fatalf("expected to start rabbitmq container: %s", err)
	}

	//slow machine
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), time.Minute*2)
		defer cancel()
	}

	//check conn
	client, err := rabbitmq.NewClient(ctx, rabbitmq.Configs{
		Host:     c.HostPort,
		User:     "guest",
		Password: "guest",
	})
	if err != nil {
		t.Fatalf("expected to create a rabbitmq client: %s", err)
	}

	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("expected to gracefully close rabbitmq: %s", err)
		}

		//close container
		if err := c.Stop(); err != nil {
			t.Errorf("expected to stop the container %s: %s", c.Id, err)
		}
	})
	return client
}
