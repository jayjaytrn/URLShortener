package urlshort

import (
	"fmt"
	"math/rand"
	"regexp"
	"time"

	"github.com/jayjaytrn/URLShortener/config"
	"github.com/jayjaytrn/URLShortener/internal/types"

	"github.com/jayjaytrn/URLShortener/internal/db"
)

var urlRegex = regexp.MustCompile(`^https?://([a-zA-Z0-9-]+\.)*[a-zA-Z0-9-]+\.[a-zA-Z]{2,}(/.*)?$`)

// GenerateShortURL generates a random short URL that does not already exist in the storage.
// It uses a random selection from a defined character set and ensures the generated short URL is unique.
func GenerateShortURL(storage db.ShortenerStorage) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const keyLength = 8

	rand.New(rand.NewSource(time.Now().UnixNano()))

	shortURL := make([]byte, keyLength)
	for {
		for i := range shortURL {
			shortURL[i] = charset[rand.Intn(len(charset))]
		}

		// Check if the generated short URL already exists
		exists, err := storage.Exists(string(shortURL))
		if err != nil {
			return "", fmt.Errorf("failed to check if URL exists: %w", err)
		}

		// If the short URL does not exist, break the loop
		if !exists {
			break
		}
	}
	return string(shortURL), nil
}

// GenerateShortBatch generates a batch of short URLs for a list of original URLs.
// It checks for uniqueness among the newly generated short URLs and ensures no conflicts exist in the storage.
func GenerateShortBatch(cfg *config.Config, storage db.ShortenerStorage, batch []types.ShortenBatchRequest, userID string) ([]types.ShortenBatchResponse, []types.URLData, error) {
	var batchResponse []types.ShortenBatchResponse
	var urlData []types.URLData
	newShorts := make(map[string]interface{})

	for n := 0; n < len(batch); {
		// Generate a short URL and check if it exists
		shortURL, err := GenerateShortURL(storage)
		if err != nil {
			return nil, nil, err
		}

		// If the short URL already exists in the newly generated list, skip it
		if _, ok := newShorts[shortURL]; ok {
			continue
		}

		// Add the new short URL to the list of generated URLs
		newShorts[shortURL] = batch[n]

		// Append the short URL response for the batch
		batchResponse = append(batchResponse, types.ShortenBatchResponse{
			CorrelationID: batch[n].CorrelationID,
			ShortURL:      cfg.BaseURL + "/" + shortURL,
		})

		// Prepare the data for database insertion
		urlData = append(urlData, types.URLData{
			ShortURL:    shortURL,
			OriginalURL: batch[n].OriginalURL,
			UserID:      userID,
		})

		// Move to the next item in the batch only if the short URL is unique
		n++
	}
	return batchResponse, urlData, nil
}

// ValidateURL validates if a URL matches the expected structure (HTTP/HTTPS).
func ValidateURL(url string) bool {
	return urlRegex.MatchString(url)
}

// ValidateBatchRequestURLs validates that all URLs in a batch are correctly formatted.
func ValidateBatchRequestURLs(batch []types.ShortenBatchRequest) bool {
	for _, b := range batch {
		if !ValidateURL(b.OriginalURL) {
			return false
		}
	}
	return true
}
