package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/jayjaytrn/URLShortener/internal/db"
	"github.com/jayjaytrn/URLShortener/internal/middleware"
	"github.com/jayjaytrn/URLShortener/logging"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/jayjaytrn/URLShortener/config"
	"github.com/jayjaytrn/URLShortener/internal/auth"
	"github.com/jayjaytrn/URLShortener/internal/handlers"
	"github.com/jayjaytrn/URLShortener/internal/types"
)

// ExampleShorten demonstrates the functionality of the /api/shorten handler.
func ExampleShorten() {
	cfg := config.GetConfig()
	storage := db.GetStorage(cfg, logging.GetSugaredLogger())
	authManager := auth.NewManager()
	h := handlers.Handler{Storage: storage, Config: cfg, AuthManager: authManager}

	// Create a request
	requestBody, _ := json.Marshal(types.ShortenRequest{URL: "https://example.com"})
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "test-user")

	w := httptest.NewRecorder()
	h.Shorten(w, req.WithContext(ctx))
}

// ExampleURLReturner demonstrates the functionality of the /{id} handler.
func ExampleURLReturner() {
	cfg := &config.Config{
		ServerAddress:   "localhost:8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "test_storage.json",
		DatabaseDSN:     "",
		StorageType:     "memory",
	}
	storage := db.GetStorage(cfg, logging.GetSugaredLogger())
	authManager := auth.NewManager()
	h := handlers.Handler{Storage: storage, Config: cfg, AuthManager: authManager}

	// Add test data
	storage.Put(types.URLData{
		ShortURL:    "abcd1234",
		OriginalURL: "https://example.com",
		UserID:      "test-user",
	})

	// Create a request
	req := httptest.NewRequest(http.MethodGet, "/abcd1234", nil)
	w := httptest.NewRecorder()
	h.URLReturner(w, req)

	// Get the response
	resp := w.Result()
	defer resp.Body.Close()

	fmt.Println(resp.StatusCode)             // Expected output: 307 (redirect)
	fmt.Println(resp.Header.Get("Location")) // Expected output: https://example.com

	// Output:
	// 307
	// https://example.com
}

// ExampleUrls demonstrates the functionality of the /api/user/urls handler.
func ExampleUrls() {
	cfg := &config.Config{
		ServerAddress:   "localhost:8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "test_storage.json",
		DatabaseDSN:     "",
		StorageType:     "memory", // ✅ Explicitly setting storage type
	}

	storage := db.GetStorage(cfg, logging.GetSugaredLogger())
	h := handlers.Handler{Storage: storage, Config: cfg}

	// Add test data
	storage.Put(types.URLData{
		ShortURL:    "abcd1234",
		OriginalURL: "https://example.com",
		UserID:      "test-user",
	})

	// Create a request
	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	w := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "test-user")

	h.Urls(w, req.WithContext(ctx))

	// Get the response
	resp := w.Result()
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	fmt.Println(resp.StatusCode) // Expected output: 200
	fmt.Println(string(body))    // JSON array of URLs

	// Output:
	// 200
	// [{"short_url":"http://localhost:8080/abcd1234","original_url":"https://example.com","is_deleted":false}]
}
