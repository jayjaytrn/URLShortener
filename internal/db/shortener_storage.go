package db

import (
	"context"

	"github.com/jayjaytrn/URLShortener/internal/types"
)

// ShortenerStorage defines the interface for URL shortening storage operations.
type ShortenerStorage interface {
	// GetOriginal retrieves the original URL corresponding to the given short URL.
	GetOriginal(shortURL string) (string, error)

	// Put adds a new URL record to the storage. Returns an error if the insertion fails.
	Put(urlData types.URLData) error

	// Exists checks if a given short URL exists in the storage.
	Exists(url string) (bool, error)

	// PutBatch inserts a batch of URL records atomically. If one record fails, the entire batch is not inserted.
	PutBatch(ctx context.Context, batchData []types.URLData) error

	// Close closes the connection to the storage.
	Close(ctx context.Context) error

	// Ping checks the availability of the storage.
	Ping(ctx context.Context) error

	// GenerateNewUserID generates and returns a new unique user ID.
	GenerateNewUserID() string

	// GetURLsByUserID retrieves all URLs associated with a given user ID.
	GetURLsByUserID(userID string) ([]types.URLData, error)

	// BatchDelete marks a batch of URLs as deleted for a given user.
	BatchDelete(urlChannel chan string, userID string)

	// GetStats возвращает количество сокращенных URL и количество пользователей
	GetStats() (types.Stats, error)
}
