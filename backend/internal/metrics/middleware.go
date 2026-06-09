package metrics

import (
	"net/http"
	"strconv"
	"time"
)

func HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		path := normalizePath(r.URL.Path)

		wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}
		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()
		HTTPDuration.WithLabelValues(r.Method, path).Observe(duration)
		HTTPRequests.WithLabelValues(r.Method, path, strconv.Itoa(wrapped.statusCode)).Inc()
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func normalizePath(path string) string {
	if len(path) > 20 {
		return path[:20]
	}
	return path
}
