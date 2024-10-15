package urlshort

import (
	"fmt"
	"math/rand"
	"regexp"
	"time"

	"github.com/jayjaytrn/URLShortener/internal/db"
)

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

func ValidateURL(url string) bool {
	regex := `^https?://([a-zA-Z0-9-]+\.)*[a-zA-Z0-9-]+\.[a-zA-Z]{2,}(/.*)?$`

	m, _ := regexp.MatchString(regex, url)
	return m
}
