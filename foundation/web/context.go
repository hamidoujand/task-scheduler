package web

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type key int

const requestMetadataKey key = 1

// RequestMetadata represents some additional info related to each requests that
// travels in between middlewares till reaches handlers
type requestMetadata struct {
	StartedAt  time.Time
	StatusCode int
	RequestId  uuid.UUID
}

func injectRequestMetadata(ctx context.Context, rm *requestMetadata) context.Context {
	return context.WithValue(ctx, requestMetadataKey, rm)
}

func setStatusCode(ctx context.Context, status int) {
	rm, ok := ctx.Value(requestMetadataKey).(*requestMetadata)
	if !ok {
		return
	}
	rm.StatusCode = status
}

func GetStatusCode(ctx context.Context) int {
	rm, ok := ctx.Value(requestMetadataKey).(*requestMetadata)
	if !ok {
		return 0
	}
	return rm.StatusCode
}

func GetStartedAt(ctx context.Context) time.Time {
	rm, ok := ctx.Value(requestMetadataKey).(*requestMetadata)
	if !ok {
		return time.Time{}
	}
	return rm.StartedAt
}

func GetRequestId(ctx context.Context) uuid.UUID {
	rm, ok := ctx.Value(requestMetadataKey).(*requestMetadata)
	if !ok {
		return uuid.UUID{}
	}
	return rm.RequestId
}
