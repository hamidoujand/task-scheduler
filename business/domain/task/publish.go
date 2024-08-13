package task

import (
	"encoding/json"
	"fmt"

	"github.com/hamidoujand/task-scheduler/business/broker/rabbitmq"
)

const queue = "queue_tasks"

func publishTask(client *rabbitmq.Client, task Task) error {
	bs, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := client.Publish(queue, bs); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	return nil
}
