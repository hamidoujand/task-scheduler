package rabbitmq_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/hamidoujand/task-scheduler/business/broker/rabbitmq"
	"github.com/hamidoujand/task-scheduler/foundation/docker"
)

const queueTest = "queue_test"

func TestRabbitMQClient(t *testing.T) {
	image := "rabbitmq:latest"

	c, err := docker.StartContainer(image, "test_rabbitmqClient", "5672", nil, nil)
	if err != nil {
		t.Fatalf("expected to start rabbitmq container: %s", err)
	}

	//slow machine
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	//check conn
	client, err := rabbitmq.NewClient(ctx, rabbitmq.Configs{
		Host:     c.HostPort,
		User:     "guest",
		Password: "guest",
	})
	if err != nil {
		t.Fatalf("expected to create a rabbitmq client: %s", err)
	}

	if err := client.DeclareQueue(queueTest); err != nil {
		t.Fatalf("expected to declare queue %s: %s", queueTest, err)
	}

	//publish
	msg := map[string]string{
		"message": "Hello World!",
	}
	bs, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshalling msg: %s", err)
	}

	if err := client.Publish(queueTest, bs); err != nil {
		t.Fatalf("expected to publish into %s: %s", queueTest, err)
	}

	msgs, err := client.Consumer(queueTest)
	if err != nil {
		t.Fatalf("expected to get delivery channel: %s", err)
	}

	delivery := <-msgs
	if delivery.ContentType != "application/json" {
		t.Errorf("contentType= %s, got %s", "application/json", delivery.ContentType)
	}

	var parsedMsg map[string]string
	if err := json.Unmarshal(delivery.Body, &parsedMsg); err != nil {
		t.Fatalf("expected msg to be parsed into map: %s", err)
	}

	got := parsedMsg["message"]
	want := msg["message"]
	if got != want {
		t.Errorf("message= %s, got %s", want, got)
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
}
