package middleware

import (
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jayjaytrn/URLShortener/internal/auth"
	"github.com/jayjaytrn/URLShortener/internal/db"

	"go.uber.org/zap"
)

// ContextKey defines a custom type for context keys to avoid key collisions.
type ContextKey string

// UserIDKey is the key used to store the user ID in the request context.
const UserIDKey ContextKey = "userID"

// CookieExistedKey is the key to determine if the cookie existed before the request.
const CookieExistedKey ContextKey = "cookieExisted"

type (
	// loggingResponseWriter is a wrapper around http.ResponseWriter that captures response details.
	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}

	// responseData stores response metadata such as status code and response size.
	responseData struct {
		status int
		size   int
	}

	// gzipWriter wraps the ResponseWriter and allows writing compressed data.
	gzipWriter struct {
		http.ResponseWriter
		GzipWriter io.Writer
	}

	// gzipReader wraps a gzip reader to handle decompression.
	gzipReader struct {
		r          io.ReadCloser
		GzipReader *gzip.Reader
	}

	// Middleware defines the signature of a middleware function.
	Middleware func(http.Handler, *zap.SugaredLogger) http.Handler
)

// Conveyor applies a chain of middlewares to a given handler.
func Conveyor(h http.Handler, sugar *zap.SugaredLogger, middlewares ...Middleware) http.Handler {
	for _, middleware := range middlewares {
		h = middleware(h, sugar)
	}
	return h
}

// WriteWithCompression is a middleware that enables GZIP compression for responses.
func WriteWithCompression(h http.Handler, sugar *zap.SugaredLogger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" && contentType != "text/html" {
			sugar.Info("Content-Type is not supported for compression. Content-Type: " + contentType)
			h.ServeHTTP(w, r)
			return
		}

		acceptEncoding := r.Header.Get("Accept-Encoding")
		if !strings.Contains(acceptEncoding, "gzip") {
			sugar.Info("Accept-Encoding does not allow compression")
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

// Write compresses and writes data to the response.
func (w gzipWriter) Write(b []byte) (int, error) {
	return w.GzipWriter.Write(b)
}

// ReadWithCompression is a middleware that enables GZIP decompression for incoming requests.
func ReadWithCompression(h http.Handler, sugar *zap.SugaredLogger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentEncoding := r.Header.Get("Content-Encoding")
		if !strings.Contains(contentEncoding, "gzip") {
			sugar.Info("Content-Encoding does not allow decompression")
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

// Read reads compressed data and decompresses it.
func (r *gzipReader) Read(p []byte) (n int, err error) {
	return r.GzipReader.Read(p)
}

// Close closes the underlying reader.
func (r *gzipReader) Close() error {
	if err := r.r.Close(); err != nil {
		return err
	}
	return r.GzipReader.Close()
}

// newGzipReader creates a new GZIP reader.
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

// WithLogging is a middleware that logs requests and responses.
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

// Write captures response size.
func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

// WriteHeader captures the response status code.
func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

// WithAuth is a middleware that manages JWT authentication and assigns user IDs.
func WithAuth(next http.Handler, authManager *auth.Manager, storage db.ShortenerStorage, logger *zap.SugaredLogger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var newJWT string
		newUserID := storage.GenerateNewUserID()
		cookie, err := r.Cookie("Authorization")
		if err != nil {
			if errors.Is(err, http.ErrNoCookie) {
				// Create a new JWT since the cookie is missing
				logger.Debug("Cookie missing, generating new JWT")
				newJWT, err = authManager.BuildJWTStringWithNewID(newUserID)
				if err != nil {
					http.Error(w, "authorization error", http.StatusInternalServerError)
					return
				}
				ctx := context.WithValue(r.Context(), UserIDKey, newUserID)
				ctx = context.WithValue(ctx, CookieExistedKey, false)
				r = r.WithContext(ctx)

				http.SetCookie(w, &http.Cookie{
					Name:     "Authorization",
					Value:    newJWT,
					Path:     "/",
					HttpOnly: true,
				})
			} else {
				logger.Debug("Error retrieving cookie: " + err.Error())
				http.Error(w, "Error retrieving cookie", http.StatusUnauthorized)
				return
			}
		} else {
			// Validate JWT from existing cookie
			logger.Debug("Cookie found, validating JWT")
			userID, err := authManager.GetUserIDFromJWTString(cookie.Value)
			if err != nil {
				logger.Debug("JWT validation failed: " + err.Error())
				// Generate a new JWT if the old one is invalid
				newJWT, err = authManager.BuildJWTStringWithNewID(newUserID)
				if err != nil {
					http.Error(w, "authorization error", http.StatusInternalServerError)
					return
				}
				ctx := context.WithValue(r.Context(), UserIDKey, newUserID)
				ctx = context.WithValue(ctx, CookieExistedKey, true)
				r = r.WithContext(ctx)

				http.SetCookie(w, &http.Cookie{
					Name:     "Authorization",
					Value:    newJWT,
					Path:     "/",
					HttpOnly: true,
				})
			} else {
				// Valid JWT, setting the user ID in context
				ctx := context.WithValue(r.Context(), UserIDKey, userID)
				ctx = context.WithValue(ctx, CookieExistedKey, true)
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
}
