package middleware

import (
	"github.com/Roshick/go-autumn-slog/pkg/logging"
	"log/slog"
	"net/http"
	"time"

	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/go-chi/chi/v5/middleware"
)

var NanosFieldName = "event.duration"
var StatusFieldName = "http.response.status_code"

// RequestLogger logs each incoming request with a single line.
//
// Place it below RequestIdMiddleware and the log line will include the request id.
func RequestLogger(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		method := r.Method
		path := r.URL.EscapedPath()

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		start := time.Now()
		aulogging.Debugf(ctx, "received request %s %s", method, path)

		defer func() {
			elapsed := time.Since(start).Nanoseconds()
			status := ww.Status()

			logger := logging.FromContext(ctx)
			logger = logger.With(NanosFieldName, elapsed, StatusFieldName, status)
			newCtx := logging.ContextWithLogger(ctx, logger)
			aulogging.Infof(newCtx, "request %s %s -> %d (%d ms)", method, path, status, elapsed/1000000)
		}()

		next.ServeHTTP(ww, r)
	}

	return http.HandlerFunc(fn)
}

var RequestIdFieldName = "http.request.id"
var MethodFieldName = "http.request.method"
var PathFieldName = "url.path"

// AddRequestScopedLoggerToContext adds a request scoped logger to the context.
//
// Place it below RequestIdMiddleware and the logger will include the request id.
//
// This ensures all intermediate log messages also list the request id, request method,
// and path. If this middleware isn't present, then only the
func AddRequestScopedLoggerToContext(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		logger := slog.Default().With(
			MethodFieldName, r.Method,
			PathFieldName, r.URL.Path,
		)

		if aulogging.RequestIdRetriever != nil {
			requestId := aulogging.RequestIdRetriever(ctx)
			logger = logger.With(RequestIdFieldName, requestId)
		}

		newCtx := logging.ContextWithLogger(ctx, logger)

		next.ServeHTTP(w, r.WithContext(newCtx))
	}
	return http.HandlerFunc(fn)
}
