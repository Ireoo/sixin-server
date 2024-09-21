package middleware

import (
	"net/http"
	"time"

	"github.com/Ireoo/sixin-server/logger"
)

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func LoggerMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sw := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		start := time.Now()
		path := r.URL.Path
		raw := r.URL.RawQuery

		next.ServeHTTP(sw, r)

		latency := time.Since(start)
		clientIP := r.RemoteAddr
		method := r.Method
		statusCode := sw.statusCode

		logger.Info("| %3d | %13v | %15s | %s  %s\n%s",
			statusCode,
			latency,
			clientIP,
			method,
			path,
			raw,
		)
	}
}
