package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

type (
	responseData struct {
		status int
		size   int
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		start := time.Now()

		responseData := &responseData{}

		loggingResponse := loggingResponseWriter{
			ResponseWriter: resp,
			responseData:   responseData,
		}

		next.ServeHTTP(&loggingResponse, req)

		duration := time.Since(start)

		zap.L().Sugar().Infow(
			"HTTP request",
			"uri", req.RequestURI,
			"method", req.Method,
			"duration", duration,
		)

		zap.L().Sugar().Infow(
			"HTTP response",
			"status", responseData.status,
			"size", responseData.size,
		)
	})
}
