package web

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

// Handler represents our custom handler that takes context and returns errors to be handle in middleware.
type Handler func(context.Context, http.ResponseWriter, *http.Request) error

// App represents the extended version of the ServeMux that has ability for middleware support.
type App struct {
	mux               *http.ServeMux
	shutdown          chan<- os.Signal
	globalMiddlewares []Middleware
}

// NewApp factory function that setup and return a *App value
func NewApp(shutdown chan<- os.Signal, mids ...Middleware) *App {
	return &App{
		mux:               http.NewServeMux(),
		shutdown:          shutdown,
		globalMiddlewares: mids,
	}
}

// HandleFunc is the custom version of *ServeMux.HandleFunc with extended features
func (a *App) HandleFunc(method string, version string, path string, handler Handler, mids ...Middleware) {
	//apply the middlewares before calling handler

	//route specific mids
	handler = applyMiddlewares(handler, mids...)
	//global mids
	handler = applyMiddlewares(handler, a.globalMiddlewares...)

	h := func(w http.ResponseWriter, r *http.Request) {
		//inject metadata
		ctx := r.Context()
		reqMeta := requestMetadata{
			StartedAt: time.Now(),
			RequestId: uuid.New(),
		}
		ctx = injectRequestMetadata(ctx, &reqMeta)

		//call our custom handler
		if err := handler(ctx, w, r); err != nil {
			//TODO handle this
		}
	}
	finalPath := path

	if version != "" {
		finalPath = "/" + version + path
	}

	finalPath = fmt.Sprintf("%s %s", method, finalPath)
	//delegate it to default HandleFunc

	a.mux.HandleFunc(finalPath, h)
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mux.ServeHTTP(w, r)
}
