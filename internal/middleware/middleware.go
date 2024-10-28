package middleware

import (
	"compress/gzip"
	"context"
	"errors"
	"github.com/jayjaytrn/URLShortener/internal/auth"
	"github.com/jayjaytrn/URLShortener/internal/db"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
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

		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			sugar.Error("Failed to create gzip writer", zap.Error(err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer gz.Close()

		w.Header().Set("Content-Encoding", "gzip")
		h.ServeHTTP(gzipWriter{ResponseWriter: w, GzipWriter: gz}, r)
	})
}

func (w gzipWriter) Write(b []byte) (int, error) {
	return w.GzipWriter.Write(b)
}

func ReadWithCompression(h http.Handler, sugar *zap.SugaredLogger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if !sendsGzip {
			sugar.Info("Content-Encoding is not allowed")
			h.ServeHTTP(w, r)
			return
		}

		gz, err := newGzipReader(r.Body)
		if err != nil {
			sugar.Error("Failed to create gzip reader", zap.Error(err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		r.Body = gz
		defer gz.Close()
		defer r.Body.Close()

		h.ServeHTTP(w, r)
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

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

func WithAuth(next http.Handler, authManager *auth.Manager, storage db.ShortenerStorage, logger *zap.SugaredLogger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var newJWT string
		newUserID := storage.GenerateNewUserID()
		cookie, err := r.Cookie("Authorization")
		if err != nil {
			// Если кука отсутствует, создаём новый JWT
			logger.Debug("Кука отсутствует")
			if errors.Is(err, http.ErrNoCookie) {
				logger.Debug("ErrNoCookie, создаем новый JWT")
				newJWT, err = authManager.BuildJWTStringWithNewID(newUserID)
				if err != nil {
					http.Error(w, "authorization error", http.StatusInternalServerError)
					return
				}
				ctx := context.WithValue(r.Context(), "userID", newUserID)
				r = r.WithContext(ctx)
				http.SetCookie(w, &http.Cookie{
					Name:     "Authorization",
					Value:    newJWT,
					Path:     "/",
					HttpOnly: true,
				})
			} else {
				logger.Debug("Другая ошибка: " + err.Error())
				http.Error(w, "authorization error", http.StatusInternalServerError)
				return
			}
		} else {
			// Если кука существует, проверяем JWT
			logger.Debug("Кука существует, проверяем JWT")
			userID, err := authManager.GetUserIdFromJWTString(cookie.Value)
			if err != nil {
				logger.Debug("Проверить не удалось: " + err.Error())
				// Если JWT не валиден, создаём новый JWT
				logger.Debug("Ошибка при получения ID из куки token is not valid: " + err.Error())
				newJWT, err = authManager.BuildJWTStringWithNewID(newUserID)
				if err != nil {
					http.Error(w, "authorization error", http.StatusInternalServerError)
					return
				}
				ctx := context.WithValue(r.Context(), "userID", userID)
				r = r.WithContext(ctx)

				http.SetCookie(w, &http.Cookie{
					Name:     "Authorization",
					Value:    newJWT,
					Path:     "/",
					HttpOnly: true,
				})
			}

			ctx := context.WithValue(r.Context(), "userID", userID)
			r = r.WithContext(ctx)

			http.SetCookie(w, &http.Cookie{
				Name:     "Authorization",
				Value:    newJWT,
				Path:     "/",
				HttpOnly: true,
			})
		}

		next.ServeHTTP(w, r)
	})
}
