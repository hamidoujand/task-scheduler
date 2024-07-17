package handlers

import (
	"net/http"

	"github.com/hamidoujand/task-scheduler/app/services/scheduler/api/handlers/tasks"
	"github.com/hamidoujand/task-scheduler/foundation/web"
)

func RegisterRoutes(app *web.App) {
	taskHandler := tasks.Handler{}

	app.HandleFunc(http.MethodGet, "v1", "/api/tasks", taskHandler.CreateTask)
}
