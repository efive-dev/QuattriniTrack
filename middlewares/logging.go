package middleware

import (
	"log"
	"net/http"
	"time"
)

// RequestLogger logs all incoming HTTP requests
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log the incoming request
		log.Printf("Started %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		// Create a response writer that captures the status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call the next handler
		next.ServeHTTP(rw, r)

		// Log the completion
		duration := time.Since(start)
		log.Printf("Completed %s %s - %d %s in %v",
			r.Method, r.URL.Path, rw.statusCode,
			http.StatusText(rw.statusCode), duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
