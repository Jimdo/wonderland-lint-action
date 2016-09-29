package api

import (
	"net/http"
	"time"

	"context"
)

const (
	DefaultHandlerTimeout = 10 * time.Second
)

type ContextHandler func(context.Context, http.ResponseWriter, *http.Request)

func HandlerWithDefaultTimeout(handler ContextHandler) func(http.ResponseWriter, *http.Request) {
	return HandlerWithTimeout(DefaultHandlerTimeout, handler)
}

func HandlerWithTimeout(timeout time.Duration, handler ContextHandler) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		handler(ctx, w, req)
	}
}
