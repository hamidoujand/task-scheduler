package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

func Respond(ctx context.Context, w http.ResponseWriter, statusCode int, data any) error {
	//if ctx is cancelled, that means client is disconnect
	if err := ctx.Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			return errors.New("client is disconnected")
		}
	}

	//otherwise
	setStatusCode(ctx, statusCode)

	if statusCode == http.StatusNoContent {
		w.WriteHeader(statusCode)
		return nil
	}

	w.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		return fmt.Errorf("encoding data: %w", err)
	}
	return nil
}
