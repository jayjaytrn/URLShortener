package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/jayjaytrn/URLShortener/config"
	"github.com/jayjaytrn/URLShortener/internal/auth"
	"github.com/jayjaytrn/URLShortener/internal/db"
	"github.com/jayjaytrn/URLShortener/internal/handlers"
	"github.com/jayjaytrn/URLShortener/internal/middleware"
	"github.com/jayjaytrn/URLShortener/internal/types"
	"github.com/jayjaytrn/URLShortener/logging"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupHandler() *handlers.Handler {
	cfg := &config.Config{
		ServerAddress:   "localhost:8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "test_storage.json",
		DatabaseDSN:     "",
		StorageType:     "memory",
	}

	storage := db.GetStorage(cfg, logging.GetSugaredLogger())
	authManager := auth.NewManager()

	return &handlers.Handler{
		Storage:     storage,
		Config:      cfg,
		AuthManager: authManager,
	}
}

func BenchmarkShorten(b *testing.B) {
	h := setupHandler()

	requestBody, _ := json.Marshal(types.ShortenRequest{URL: "https://example.com"})
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "test-user")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		h.Shorten(w, req.WithContext(ctx))
	}
}

func BenchmarkURLReturner(b *testing.B) {
	h := setupHandler()

	// Добавляем тестовый URL
	h.Storage.Put(types.URLData{
		ShortURL:    "abcd1234",
		OriginalURL: "https://example.com",
		UserID:      "test-user",
	})

	req := httptest.NewRequest(http.MethodGet, "/abcd1234", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		h.URLReturner(w, req)
	}
}

func BenchmarkPing(b *testing.B) {
	h := setupHandler()

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		h.Ping(w, req)
	}
}

func BenchmarkUrls(b *testing.B) {
	h := setupHandler()

	h.Storage.Put(types.URLData{
		ShortURL:    "abcd1234",
		OriginalURL: "https://example.com",
		UserID:      "test-user",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "test-user")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		h.Urls(w, req.WithContext(ctx))
	}
}

func BenchmarkShortenBatch(b *testing.B) {
	h := setupHandler()

	batchRequest := []types.ShortenBatchRequest{
		{CorrelationID: "1", OriginalURL: "https://example.com/1"},
		{CorrelationID: "2", OriginalURL: "https://example.com/2"},
	}
	requestBody, _ := json.Marshal(batchRequest)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "test-user")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		h.ShortenBatch(w, req.WithContext(ctx))
	}
}
