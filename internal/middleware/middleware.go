package middleware

import (
	"compress/gzip"
	"go.uber.org/zap"
	"io"
	"net/http"
	"time"
)

type (
	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}

	responseData struct {
		status int
		size   int
	}

	gzipWriter struct {
		http.ResponseWriter
		Writer io.Writer
	}

	Middleware func(http.Handler, *zap.SugaredLogger) http.Handler
)

func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func Conveyor(h http.Handler, sugar *zap.SugaredLogger, middlewares ...Middleware) http.Handler {
	for _, middleware := range middlewares {
		h = middleware(h, sugar)
	}
	return h
}

func WriteWithCompression(h http.Handler, sugar *zap.SugaredLogger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptEncodingValues := r.Header.Values("Accept-Encoding")
		target := "gzip"
		found := false

		for _, encoding := range acceptEncodingValues {
			if encoding == target {
				found = true
				break
			}
		}

		if !found {
			sugar.Info("Accept-Encoding is not allowed")
			h.ServeHTTP(w, r)
			return
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" && contentType != "text/html" {
			sugar.Info("Content-Type is not supported for compression. Content-Type: " + contentType)
			h.ServeHTTP(w, r)
			return
		}

		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			sugar.Error("Failed to create gzip writer", zap.Error(err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer gz.Close()

		w.Header().Set("Content-Encoding", "gzip")
		h.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: gz}, r)
	})
}

func ReadWithCompression(h http.Handler, sugar *zap.SugaredLogger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentEncodingValues := r.Header.Values("Content-Encoding")
		target := "gzip"
		found := false

		for _, encoding := range contentEncodingValues {
			if encoding == target {
				found = true
				break
			}
		}

		if !found {
			sugar.Info("Content-Encoding is not allowed")
			h.ServeHTTP(w, r)
			return
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" && contentType != "text/html" {
			sugar.Info("Content-Type is not supported for decompression. Content-Type: " + contentType)
			h.ServeHTTP(w, r)
			return
		}

		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			sugar.Error("Failed to create gzip writer", zap.Error(err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer gz.Close()

		newReq := r.Clone(r.Context())
		newReq.Body = gz
		newReq.ContentLength = -1

		h.ServeHTTP(w, newReq)
	})
}

func WithLogging(h http.Handler, sugar *zap.SugaredLogger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rd := &responseData{
			status: 0,
			size:   0,
		}

		lw := loggingResponseWriter{
			ResponseWriter: w,
			responseData:   rd,
		}
		h.ServeHTTP(&lw, r)

		duration := time.Since(start)

		sugar.Infoln(
			"uri", r.RequestURI,
			"method", r.Method,
			"status", rd.status,
			"duration", duration,
			"size", rd.size,
		)
	})
}

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}
