package middleware

import (
	"compress/gzip"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strings"
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
		GzipWriter io.Writer
	}

	gzipReader struct {
		r          io.ReadCloser
		GzipReader *gzip.Reader
	}

	Middleware func(http.Handler, *zap.SugaredLogger) http.Handler
)

func Conveyor(h http.Handler, sugar *zap.SugaredLogger, middlewares ...Middleware) http.Handler {
	for _, middleware := range middlewares {
		h = middleware(h, sugar)
	}
	return h
}

func WriteWithCompression(h http.Handler, sugar *zap.SugaredLogger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" && contentType != "text/html" {
			sugar.Info("Content-Type is not supported for compression. Content-Type: " + contentType)
			h.ServeHTTP(w, r)
			return
		}

		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
		if !supportsGzip {
			sugar.Info("Accept-Encoding is not allowed")
			h.ServeHTTP(w, r)
			return
		}

		gz := newGzipWriter(w)

		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length")
		h.ServeHTTP(gzipWriter{ResponseWriter: w, GzipWriter: gz}, r)
	})
}

func (r *gzipReader) Read(p []byte) (n int, err error) {
	return r.GzipReader.Read(p)
}

func (r *gzipReader) Close() error {
	if err := r.r.Close(); err != nil {
		return err
	}
	return r.GzipReader.Close()
}

func (w gzipWriter) Write(b []byte) (int, error) {
	return w.GzipWriter.Write(b)
}

func newGzipWriter(w http.ResponseWriter) *gzipWriter {
	return &gzipWriter{
		GzipWriter: gzip.NewWriter(w),
	}
}

func ReadWithCompression(h http.Handler, sugar *zap.SugaredLogger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" && contentType != "text/html" {
			sugar.Info("Content-Type is not supported for decompression. Content-Type: " + contentType)
			h.ServeHTTP(w, r)
			return
		}

		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if !sendsGzip {
			sugar.Info("Content-Encoding is not allowed")
			h.ServeHTTP(w, r)
			return
		}

		gr, err := newGzipReader(r.Body)
		if err != nil {
			sugar.Error("Failed to create gzip reader", zap.Error(err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		r.Body = gr.r

		defer gr.Close()

		h.ServeHTTP(w, r)
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

func newGzipReader(r io.ReadCloser) (*gzipReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &gzipReader{
		r:          r,
		GzipReader: zr,
	}, nil
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
