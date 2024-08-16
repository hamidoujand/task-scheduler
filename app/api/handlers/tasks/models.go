package tasks

import (
	"strings"
	"time"

	"github.com/hamidoujand/task-scheduler/business/domain/task"
)

// Task represents a task that goes to client
type Task struct {
	Id          string            `json:"id"`
	UserId      string            `json:"user_id"`
	Command     string            `json:"command"`
	Args        []string          `json:"args"`
	Image       string            `json:"image"`
	Environment map[string]string `json:"environment"`
	Status      string            `json:"status"`
	Result      string            `json:"result"`
	ErrMessage  string            `json:"errorMsg"`
	ScheduledAt time.Time         `json:"scheduledAt"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

func fromDomainTask(t task.Task) Task {
	envs := strings.Fields(t.Environment)
	envMap := make(map[string]string, len(envs))

	for _, env := range envs {
		parts := strings.Split(env, "=")
		envMap[parts[0]] = parts[1]
	}

	return Task{
		Id:          t.Id.String(),
		UserId:      t.UserId.String(),
		Command:     t.Command,
		Args:        t.Args,
		Image:       t.Image,
		Environment: envMap,
		Status:      t.Status.String(),
		Result:      t.Result,
		ErrMessage:  t.ErrMessage,
		ScheduledAt: t.ScheduledAt,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

// NewTask represents data required for task creation.
type NewTask struct {
	Command     string            `json:"command" validate:"required,ascii,commonCommands"`
	Args        []string          `json:"args" validate:"commonArgs"`
	Image       string            `json:"image" validate:"required"`
	Environment map[string]string `json:"environment"`
	ScheduledAt time.Time         `json:"scheduledAt" validate:"required,validScheduledAt"`
}
