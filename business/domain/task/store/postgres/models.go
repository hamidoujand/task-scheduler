package postgres

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/business/domain/task"
)

// Task represents a task object inside of database.
type Task struct {
	Id           uuid.UUID
	UserId       uuid.UUID
	Command      string
	Args         []string
	Status       string
	Result       sql.Null[string]
	ErrorMessage sql.Null[string]
	ScheduledAt  time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func toDBTask(t task.Task) Task {
	return Task{
		Id:           t.Id,
		UserId:       t.UserId,
		Command:      t.Command,
		Args:         t.Args,
		Status:       t.Status.String(),
		Result:       sql.Null[string]{V: t.Result, Valid: t.Result != ""},
		ErrorMessage: sql.Null[string]{V: t.ErrMessage, Valid: t.ErrMessage != ""},
		ScheduledAt:  t.ScheduledAt.UTC(),
		CreatedAt:    t.CreatedAt.UTC(),
		UpdatedAt:    t.ScheduledAt.UTC(),
	}
}

func (t Task) toDomainTask() task.Task {
	result := ""

	if t.Result.Valid {
		result = t.Result.V
	}
	errMsgs := ""
	if t.ErrorMessage.Valid {
		errMsgs = t.ErrorMessage.V
	}

	status, _ := task.ParseStatus(t.Status)

	return task.Task{
		//must parse since we taking it out of db.
		Id:          t.Id,
		UserId:      t.UserId,
		Command:     t.Command,
		Args:        t.Args,
		Status:      status,
		Result:      result,
		ErrMessage:  errMsgs,
		ScheduledAt: t.ScheduledAt.In(time.Local),
		CreatedAt:   t.CreatedAt.In(time.Local),
		UpdatedAt:   t.UpdatedAt.In(time.Local),
	}
}
