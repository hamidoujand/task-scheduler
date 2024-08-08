package redis

import (
	"github.com/google/uuid"
	"github.com/hamidoujand/task-scheduler/business/domain/worker"
)

type redisWorker struct {
	ID     string `redis:"id"`
	Load   int    `redis:"load"`
	Status string `redis:"status"`
}

func fromWorkerService(worker worker.Worker) redisWorker {
	return redisWorker{
		ID:     worker.ID.String(),
		Load:   worker.Load,
		Status: worker.Status.String(),
	}
}

func (rw redisWorker) toServiceWorker() worker.Worker {
	id, _ := uuid.Parse(rw.ID)
	return worker.Worker{
		ID:     id,
		Status: worker.ParseStatus(rw.Status),
		Load:   rw.Load,
	}
}
