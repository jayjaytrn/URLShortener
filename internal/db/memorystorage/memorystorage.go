package memorystorage

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jayjaytrn/URLShortener/config"
	"github.com/jayjaytrn/URLShortener/internal/types"
)

// Manager handles in-memory storage for shortened URLs.
type Manager struct {
	RelatesURLs []types.URLData
	Config      *config.Config
}

// NewManager initializes a new memory storage manager.
func NewManager(cfg *config.Config) (*Manager, error) {
	return &Manager{
		RelatesURLs: []types.URLData{},
		Config:      cfg,
	}, nil
}

// GetOriginal retrieves the original URL associated with the given short URL.
func (m *Manager) GetOriginal(shortURL string) (string, error) {
	for _, urlData := range m.RelatesURLs {
		if urlData.ShortURL == shortURL {
			return urlData.OriginalURL, nil
		}
	}
	return "", fmt.Errorf("URL not found")
}

// Put stores a new URL mapping in memory.
func (m *Manager) Put(urlData types.URLData) error {
	m.RelatesURLs = append(m.RelatesURLs, urlData)
	return nil
}

// PutBatch stores multiple URL mappings in memory.
func (m *Manager) PutBatch(_ context.Context, batchData []types.URLData) error {
	for _, urlData := range batchData {
		err := m.Put(urlData)
		if err != nil {
			return err
		}
	}
	return nil
}

// GenerateNewUserID generates a new unique user ID.
func (m *Manager) GenerateNewUserID() string {
	return uuid.New().String()
}

// Exists checks if a given short URL exists in the storage.
func (m *Manager) Exists(shortURL string) (bool, error) {
	for _, urlData := range m.RelatesURLs {
		if urlData.ShortURL == shortURL {
			return true, nil
		}
	}
	return false, nil
}

// GetURLsByUserID retrieves all URLs shortened by a specific user.
func (m *Manager) GetURLsByUserID(userID string) ([]types.URLData, error) {
	var userURLs []types.URLData

	for _, urlData := range m.RelatesURLs {
		if urlData.UserID == userID {
			userURLs = append(userURLs, types.URLData{
				ShortURL:    m.Config.BaseURL + "/" + urlData.ShortURL,
				OriginalURL: urlData.OriginalURL,
			})
		}
	}

	if len(userURLs) == 0 {
		return nil, fmt.Errorf("no URLs found for userID: %s", userID)
	}

	return userURLs, nil
}

// BatchDelete is a placeholder for batch deletion, not implemented for memory storage.
func (m *Manager) BatchDelete(_ chan string, _ string) {
}

// Close releases any allocated resources (not required for memory storage).
func (m *Manager) Close(_ context.Context) error {
	return nil
}

// Ping checks the availability of the storage, not supported for memory storage.
func (m *Manager) Ping(_ context.Context) error {
	return fmt.Errorf("ping is not supported for memory storage")
}

// GetStats возвращает количество сокращенных URL и количество пользователей.
func (m *Manager) GetStats() (types.Stats, error) {
	urlCount := len(m.RelatesURLs)
	userSet := make(map[string]struct{})

	for _, urlData := range m.RelatesURLs {
		if urlData.UserID != "" {
			userSet[urlData.UserID] = struct{}{}
		}
	}

	return types.Stats{
		Urls:  urlCount,
		Users: len(userSet),
	}, nil
}
