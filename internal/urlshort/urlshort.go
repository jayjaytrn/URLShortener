package urlshort

import (
	"github.com/jayjaytrn/URLShortener/internal/storage"
	"math/rand"
	"regexp"
	"time"
)

func GenerateShortURL() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const keyLength = 8

	rand.New(rand.NewSource(time.Now().UnixNano()))

	shortURL := make([]byte, keyLength)
	for {
		for i := range shortURL {
			shortURL[i] = charset[rand.Intn(len(charset))]
		}

		oldURLs := make(map[string]interface{})
		for _, urlData := range storage.UrlStorage {
			oldURLs[urlData.ShortUrl] = struct{}{}
		}

		if _, exists := oldURLs[string(shortURL)]; !exists {
			break
		}
	}
	return string(shortURL)
}

func ValidateURL(url string) bool {
	regex := `^https?://([a-zA-Z0-9-]+\.)*[a-zA-Z0-9-]+\.[a-zA-Z]{2,}(/.*)?$`

	m, _ := regexp.MatchString(regex, url)
	return m
}
