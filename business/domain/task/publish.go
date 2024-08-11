package task

import (
	"fmt"

	"github.com/hamidoujand/task-scheduler/business/broker/rabbitmq"
)

const queue = "tasks"

func publish(client *rabbitmq.Client, bs []byte) error {
	if err := client.Publish(queue, bs); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	return nil
}
