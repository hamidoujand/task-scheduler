package web

import (
	"context"
	"fmt"
	"net/http"
	"os"
)

type Handler func(context.Context, http.ResponseWriter, *http.Request) error

type App struct {
	mux      *http.ServeMux
	shutdown chan<- os.Signal
}

func NewApp(shutdown chan<- os.Signal) *App {
	return &App{
		mux:      http.NewServeMux(),
		shutdown: shutdown,
	}
}

func (a *App) HandleFunc(method string, version string, path string, handler Handler) {
	h := func(w http.ResponseWriter, r *http.Request) {
		//call our custom handler
		if err := handler(r.Context(), w, r); err != nil {
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
