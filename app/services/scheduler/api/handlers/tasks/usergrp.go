package tasks

import (
	"context"
	"fmt"
	"net/http"
)

type Handler struct {
}

func (h *Handler) CreateTask(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	fmt.Println("successfully hit")
	return nil
}
