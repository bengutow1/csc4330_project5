// Author: Caden Lund
package logging

import (
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

// RequestLogger logs one structured line per request through log. It records the path and never
// the query string, which is where a token or a code would ride.
func RequestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)
			log.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", sw.status,
				"bytes", sw.bytes,
				"elapsed_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}

// Recoverer turns a panic in a handler into a logged error and a 500. net/http would survive the
// panic on its own, but it prints a bare stack outside slog and drops the connection, which a
// client reports as a network failure instead of a server one.
func Recoverer(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				rec := recover()
				if rec == nil {
					return
				}
				// ErrAbortHandler is how a handler abandons a response on purpose; net/http
				// knows to stay quiet about it.
				err, isErr := rec.(error)
				if isErr && errors.Is(err, http.ErrAbortHandler) {
					panic(rec)
				}
				log.Error("panic serving request",
					"method", r.Method,
					"path", r.URL.Path,
					"panic", rec,
					"stack", string(debug.Stack()),
				)
				// If the handler already wrote a status this is a no-op, and the log is the
				// only record.
				http.Error(w, "internal error", http.StatusInternalServerError)
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// statusWriter records the status and body size on their way through. A handler that never
// calls WriteHeader sent a 200.
type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

// Flush passes through, so a server stream still reaches the client message by message.
func (w *statusWriter) Flush() {
	f, ok := w.ResponseWriter.(http.Flusher)
	if ok {
		f.Flush()
	}
}
