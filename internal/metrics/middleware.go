package metrics

import (
	"net/http"
	"time" // Update with your actual module name
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.statusCode = http.StatusOK
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}

// MetricsMiddleware records HTTP metrics for all requests
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Increment active connections
		IncActiveConnections()
		defer DecActiveConnections()

		// Wrap response writer to capture status code
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     200, // default
			written:        false,
		}

		// Serve the request
		next.ServeHTTP(rw, r)

		// Record metrics
		duration := time.Since(start)
		RecordHTTPRequest(r.Method, r.URL.Path, rw.statusCode, duration)
	})
}
