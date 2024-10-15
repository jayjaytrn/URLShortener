package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/jayjaytrn/URLShortener/config"
	"github.com/jayjaytrn/URLShortener/internal/db/filestorage"
	"github.com/jayjaytrn/URLShortener/internal/handlers"
)

func Test_urlWaiter(t *testing.T) {
	cfg := config.GetConfig()

	storage, err := filestorage.NewManager(cfg)
	if err != nil {
		t.Fatalf("failed to initialize file storage: %v", err)
	}
	defer storage.Close(context.Background())

	handler := handlers.Handler{
		Storage: storage,
		Config:  cfg,
	}

	tests := []struct {
		name         string
		method       string
		body         string
		expectedCode int
		expectedBody string
	}{
		{
			name:         "Valid POST",
			method:       http.MethodPost,
			body:         "https://practicum.yandex.ru/",
			expectedCode: http.StatusCreated,
			expectedBody: "shortURL",
		},
		{
			name:         "Invalid Method",
			method:       http.MethodGet,
			body:         "",
			expectedCode: http.StatusBadRequest,
			expectedBody: "only POST method is allowed\n",
		},
		{
			name:         "Empty URL",
			method:       http.MethodPost,
			body:         "",
			expectedCode: http.StatusBadRequest,
			expectedBody: "wrong parameters\n",
		},
		{
			name:         "Wrong parameters",
			method:       http.MethodPost,
			body:         string([]byte{0xFF}), // некорректный байт
			expectedCode: http.StatusBadRequest,
			expectedBody: "wrong parameters\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			req := httptest.NewRequest(tt.method, "http://localhost:8080/", io.NopCloser(strings.NewReader(tt.body)))
			w := httptest.NewRecorder()

			handler.URLWaiter(w, req)

			res := w.Result()
			defer res.Body.Close()

			if res.StatusCode != tt.expectedCode {
				t.Errorf("expected status %v, got %v", tt.expectedCode, res.StatusCode)
			}

			body, _ := io.ReadAll(res.Body)
			if tt.expectedBody == "shortURL" {
				re := regexp.MustCompile(`^http://localhost:8080/[a-zA-Z0-9]{8}$`)
				if !re.MatchString(string(body)) {
					t.Errorf("expected body to match regex, got %v", string(body))
				}
			} else {
				if string(body) != tt.expectedBody {
					t.Errorf("expected body %v, got %v", tt.expectedBody, string(body))
				}
			}
		})
	}
}

func Test_urlReturner(t *testing.T) {
	cfg := config.GetConfig()

	storage, err := filestorage.NewManager(cfg)
	if err != nil {
		t.Fatalf("failed to initialize file storage: %v", err)
	}
	defer storage.Close(context.Background())

	handler := handlers.Handler{
		Storage: storage,
		Config:  cfg,
	}

	tests := []struct {
		name           string
		method         string
		path           string
		expectedCode   int
		expectedHeader string
		expectedBody   string
	}{
		{
			name:           "Valid GET Request",
			method:         http.MethodGet,
			path:           "/shortURL",
			expectedCode:   http.StatusTemporaryRedirect,
			expectedHeader: "https://practicum.yandex.ru/",
			expectedBody:   "",
		},
		{
			name:           "Invalid Method",
			method:         http.MethodPost,
			path:           "/test",
			expectedCode:   http.StatusBadRequest,
			expectedHeader: "",
			expectedBody:   "only GET method is allowed\n",
		},
		{
			name:           "Non-existent Short URL",
			method:         http.MethodGet,
			path:           "/test",
			expectedCode:   http.StatusNotFound,
			expectedHeader: "",
			expectedBody:   "URL not found\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var postBody []byte
			var req *http.Request

			req = httptest.NewRequest(tt.method, tt.path, nil)
			if tt.path == "/shortURL" {
				postRequest := httptest.NewRequest("POST", "http://localhost:8080/", io.NopCloser(strings.NewReader("https://practicum.yandex.ru/")))
				postResponse := httptest.NewRecorder()

				handler.URLWaiter(postResponse, postRequest)

				postResult := postResponse.Result()
				defer postResult.Body.Close()
				postBody, _ = io.ReadAll(postResult.Body)
				req = httptest.NewRequest(tt.method, string(postBody), nil)
			}

			getResponse := httptest.NewRecorder()

			handler.URLReturner(getResponse, req)

			getResult := getResponse.Result()
			defer getResult.Body.Close()

			if getResult.StatusCode != tt.expectedCode {
				t.Errorf("expected status %v, got %v", tt.expectedCode, getResult.StatusCode)
			}

			if locHeader := getResult.Header.Get("Location"); locHeader != tt.expectedHeader {
				t.Errorf("expected Location header %v, got %v", tt.expectedHeader, locHeader)
			}

			body, _ := io.ReadAll(getResult.Body)
			if string(body) != tt.expectedBody {
				t.Errorf("expected body %v, got %v", tt.expectedBody, string(body))
			}
		})
	}
}
