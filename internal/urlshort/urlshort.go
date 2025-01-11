package urlshort

import (
	"fmt"
	"github.com/jayjaytrn/URLShortener/config"
	"github.com/jayjaytrn/URLShortener/internal/types"
	"math/rand"
	"regexp"
	"time"

	"github.com/jayjaytrn/URLShortener/internal/db"
)

var urlRegex = regexp.MustCompile(`^https?://([a-zA-Z0-9-]+\.)*[a-zA-Z0-9-]+\.[a-zA-Z]{2,}(/.*)?$`)

func GenerateShortURL(storage db.ShortenerStorage) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const keyLength = 8

	rand.New(rand.NewSource(time.Now().UnixNano()))

	shortURL := make([]byte, keyLength)
	for {
		for i := range shortURL {
			shortURL[i] = charset[rand.Intn(len(charset))]
		}

		exists, err := storage.Exists(string(shortURL))
		if err != nil {
			return "", fmt.Errorf("failed to check if URL exists: %w", err)
		}

		if !exists {
			break
		}
	}
	return string(shortURL), nil
}

func GenerateShortBatch(cfg *config.Config, storage db.ShortenerStorage, batch []types.ShortenBatchRequest, userID string) ([]types.ShortenBatchResponse, []types.URLData, error) {
	var batchResponse []types.ShortenBatchResponse
	var urlData []types.URLData
	newShorts := make(map[string]interface{})
	for n := 0; n < len(batch); {
		// Генерируем короткий URL
		// проверяем есть ли такой в БД
		shortURL, err := GenerateShortURL(storage)
		if err != nil {
			return nil, nil, err
		}

		// Проверяем, существует ли уже такой короткий URL среди сгенерированных новых
		if _, ok := newShorts[shortURL]; ok {
			// Если такой URL уже есть, продолжаем цикл с того же индекса
			continue
		}

		// Добавляем новый короткий URL
		newShorts[shortURL] = batch[n]

		// Формируем батч для ответа клиенту
		batchResponse = append(batchResponse, types.ShortenBatchResponse{
			CorrelationID: batch[n].CorrelationID,
			ShortURL:      cfg.BaseURL + "/" + shortURL,
		})

		// Формируем данные дял записи в БД
		urlData = append(urlData, types.URLData{
			ShortURL:    shortURL,
			OriginalURL: batch[n].OriginalURL,
			UserID:      userID,
		})

		// Переходим к следующей итерации только если URL уникальный
		n++
	}
	return batchResponse, urlData, nil
}

func ValidateURL(url string) bool {
	return urlRegex.MatchString(url)
}

func ValidateBatchRequestURLs(batch []types.ShortenBatchRequest) bool {
	for _, b := range batch {
		if !ValidateURL(b.OriginalURL) {
			return false
		}
	}
	return true
}
