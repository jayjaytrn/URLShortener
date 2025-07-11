package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/jayjaytrn/URLShortener/logging"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/jayjaytrn/URLShortener/config"
	"github.com/jayjaytrn/URLShortener/internal/auth"
	"github.com/jayjaytrn/URLShortener/internal/db"
	"github.com/jayjaytrn/URLShortener/internal/db/postgres"
	"github.com/jayjaytrn/URLShortener/internal/middleware"
	"github.com/jayjaytrn/URLShortener/internal/types"
	"github.com/jayjaytrn/URLShortener/internal/urlshort"
)

// Handler represents the main HTTP handler for the URL shortening service.
type Handler struct {
	Storage     db.ShortenerStorage
	Config      *config.Config
	AuthManager *auth.Manager
}

// URLWaiter handles waiting for a URL input and processing it.
func (h *Handler) URLWaiter(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(res, "only POST method is allowed", http.StatusBadRequest)
		return
	}

	req.FormValue("url")
	res.Header().Set("Content-Type", "text/plain")

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, "error when read body", http.StatusBadRequest)
		return
	}
	url := string(body)
	valid := urlshort.ValidateURL(url)
	if !valid {
		http.Error(res, "wrong parameters", http.StatusBadRequest)
		return
	}

	su, err := urlshort.GenerateShortURL(h.Storage)
	if err != nil {
		http.Error(res, "failed to generate short URL: "+err.Error(), http.StatusInternalServerError)
		return
	}

	userID, ok := req.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(res, "internal server error", http.StatusBadRequest)
		return
	}

	urlData := types.URLData{
		OriginalURL: url,
		ShortURL:    su,
		UserID:      userID,
	}

	err = h.Storage.Put(urlData)
	if err != nil {
		var originalExistErr *postgres.OriginalExistError
		if errors.As(err, &originalExistErr) {
			r := h.Config.BaseURL + "/" + originalExistErr.ShortURL
			res.Header().Set("content-type", "text/plain")
			res.WriteHeader(http.StatusConflict)
			res.Write([]byte(r))
			return
		}
		http.Error(res, "error when trying to put data in storage", http.StatusInternalServerError)
		return
	}

	r := h.Config.BaseURL + "/" + su
	res.Header().Set("content-type", "text/plain")
	res.WriteHeader(http.StatusCreated)
	res.Write([]byte(r))
}

// URLReturner retrieves the original URL from the shortened URL.
func (h *Handler) URLReturner(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(res, "only GET method is allowed", http.StatusBadRequest)
		return
	}

	shortURL := req.URL.Path[len("/"):]

	originalURL, err := h.Storage.GetOriginal(shortURL)
	if err != nil {
		if strings.Contains(err.Error(), "URL has been deleted") {
			http.Error(res, "URL has been deleted", http.StatusGone) // 410 Gone
			return
		}
		http.Error(res, "URL not found", http.StatusNotFound)
		return
	}

	res.Header().Set("Location", originalURL)
	res.WriteHeader(http.StatusTemporaryRedirect)
}

// Shorten handles the request to shorten a given URL.
func (h *Handler) Shorten(res http.ResponseWriter, req *http.Request) {
	var buf bytes.Buffer
	_, err := buf.ReadFrom(req.Body)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	var shortenRequest types.ShortenRequest
	err = json.Unmarshal(buf.Bytes(), &shortenRequest)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	url := shortenRequest.URL
	valid := urlshort.ValidateURL(url)
	if !valid {
		http.Error(res, "wrong parameters", http.StatusBadRequest)
		return
	}

	su, err := urlshort.GenerateShortURL(h.Storage)
	if err != nil {
		http.Error(res, "failed to generate short URL: "+err.Error(), http.StatusInternalServerError)
		return
	}

	userID, ok := req.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(res, "internal server error", http.StatusBadRequest)
		return
	}

	urlData := types.URLData{
		OriginalURL: url,
		ShortURL:    su,
		UserID:      userID,
	}
	err = h.Storage.Put(urlData)
	if err != nil {
		var originalExistErr *postgres.OriginalExistError
		if errors.As(err, &originalExistErr) {
			r := h.Config.BaseURL + "/" + originalExistErr.ShortURL
			shortenResponse := types.ShortenResponse{
				Result: r,
			}

			br, err := json.Marshal(shortenResponse)
			if err != nil {
				http.Error(res, err.Error(), http.StatusBadRequest)
				return
			}

			res.Header().Set("Content-Type", "application/json")
			res.WriteHeader(http.StatusConflict)
			res.Write(br)
			return
		}
		http.Error(res, "error when trying to put data in storage", http.StatusInternalServerError)
		return
	}

	r := h.Config.BaseURL + "/" + su
	shortenResponse := types.ShortenResponse{
		Result: r,
	}
	br, err := json.Marshal(shortenResponse)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)
	res.Write(br)
}

// ShortenBatch processes batch URL shortening requests.
func (h *Handler) ShortenBatch(res http.ResponseWriter, req *http.Request) {
	var buf bytes.Buffer
	_, err := buf.ReadFrom(req.Body)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	var batchRequest []types.ShortenBatchRequest
	err = json.Unmarshal(buf.Bytes(), &batchRequest)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	valid := urlshort.ValidateBatchRequestURLs(batchRequest)
	if !valid {
		http.Error(res, "wrong parameters", http.StatusBadRequest)
		return
	}

	userID, ok := req.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(res, "internal server error", http.StatusBadRequest)
		return
	}

	batchResponse, batchData, err := urlshort.GenerateShortBatch(h.Config, h.Storage, batchRequest, userID)
	if err != nil {
		http.Error(res, "failed to generate short URL: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = h.Storage.PutBatch(req.Context(), batchData)
	if err != nil {
		http.Error(res, "error when trying to put data in storage", http.StatusInternalServerError)
	}

	br, err := json.Marshal(batchResponse)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)
	res.Write(br)
}

// Ping checks the database connection status.
func (h *Handler) Ping(res http.ResponseWriter, req *http.Request) {
	if err := h.Storage.Ping(req.Context()); err != nil {
		http.Error(res, "database connection error", http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
}

// Urls retrieves all shortened URLs associated with a specific user.
func (h *Handler) Urls(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")
	userID, ok := req.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(res, "internal server error", http.StatusBadRequest)
		return
	}

	if req.Context().Value(middleware.CookieExistedKey) == false {
		http.Error(res, "Unauthorized - cookie was created by request", http.StatusNoContent)
		return
	}

	urls, err := h.Storage.GetURLsByUserID(userID)
	if err != nil {
		if strings.Contains(err.Error(), "no URLs found for userID") {
			res.WriteHeader(http.StatusNoContent)
			return
		}
	}

	urlsResponse, err := json.Marshal(urls)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	res.WriteHeader(http.StatusOK)
	res.Write(urlsResponse)
}

// DeleteUrlsAsync asynchronously deletes a list of shortened URLs.
func (h *Handler) DeleteUrlsAsync(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")
	userID, ok := req.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(res, "internal server error", http.StatusBadRequest)
		return
	}

	// Проверка авторизации
	if req.Context().Value(middleware.CookieExistedKey) == false {
		http.Error(res, "Unauthorized - cookie was created by request", http.StatusUnauthorized)
		return
	}

	// Декодирование списка URL из запроса
	var shortURLs []string
	if err := json.NewDecoder(req.Body).Decode(&shortURLs); err != nil {
		http.Error(res, "Invalid request payload", http.StatusBadRequest)
		return
	}

	res.WriteHeader(http.StatusAccepted)

	// Создаём канал для передачи URL на удаление
	urlChannel := make(chan string)

	// Запуск горутины для отправки URL в канал
	go func() {
		for _, shortURL := range shortURLs {
			urlChannel <- shortURL
		}
		close(urlChannel) // Закрываем канал после передачи всех URL
	}()

	// Запускаем BatchDelete с каналом urlChannel
	go h.Storage.BatchDelete(urlChannel, userID)
}

// Stats return stats.
func (h *Handler) Stats(res http.ResponseWriter, req *http.Request) {
	logger := logging.GetSugaredLogger()
	defer logger.Sync()

	trustedSubnet := h.Config.TrustedSubnet
	if trustedSubnet == "" {
		http.Error(res, "access denied: trusted_subnet is not set", http.StatusForbidden)
		return
	}

	clientIP := req.Header.Get("X-Real-IP")
	if clientIP == "" {
		http.Error(res, "access denied: missing X-Real-IP", http.StatusForbidden)
		return
	}

	if !isIPInTrustedSubnet(clientIP, trustedSubnet) {
		http.Error(res, "access denied: IP not in trusted subnet", http.StatusForbidden)
		return
	}

	// Получаем количество уникальных пользователей и сокращённых URL
	stats, err := h.Storage.GetStats()
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Формируем JSON-ответ
	response := types.Stats{
		Urls:  stats.Urls,
		Users: stats.Users,
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	json.NewEncoder(res).Encode(response)
}

// isIPInTrustedSubnet проверяет, входит ли IP в доверенную подсеть
func isIPInTrustedSubnet(ip, subnet string) bool {
	clientIP := net.ParseIP(ip)
	if clientIP == nil {
		return false
	}

	_, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		return false
	}

	return ipNet.Contains(clientIP)
}
