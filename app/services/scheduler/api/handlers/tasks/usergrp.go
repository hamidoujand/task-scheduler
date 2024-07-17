package tasks

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hamidoujand/task-scheduler/foundation/web"
)

type Handler struct {
}

func (h *Handler) CreateTask(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	fmt.Println("successfully hit")

	msg := map[string]string{
		"msg": "hello world",
	}
	return web.Respond(ctx, w, http.StatusOK, msg)
}
