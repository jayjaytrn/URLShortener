package memorystorage

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jayjaytrn/URLShortener/config"
	"github.com/jayjaytrn/URLShortener/internal/types"
)

type Manager struct {
	RelatesURLs []types.URLData
	Config      *config.Config
}

func NewManager(cfg *config.Config) (*Manager, error) {
	return &Manager{
		RelatesURLs: []types.URLData{},
		Config:      cfg,
	}, nil
}

func (m *Manager) GetOriginal(shortURL string) (string, error) {
	for _, urlData := range m.RelatesURLs {
		if urlData.ShortURL == shortURL {
			return urlData.OriginalURL, nil
		}
	}
	return "", fmt.Errorf("URL not found")
}

func (m *Manager) Put(urlData types.URLData) error {
	m.RelatesURLs = append(m.RelatesURLs, urlData)
	return nil
}

func (m *Manager) PutBatch(_ context.Context, batchData []types.URLData) error {
	for _, urlData := range batchData {
		err := m.Put(urlData)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) GenerateNewUserID() string {
	return uuid.New().String()
}

// Exists возвращает true если запись найдена
func (m *Manager) Exists(shortURL string) (bool, error) {
	for _, urlData := range m.RelatesURLs {
		if urlData.ShortURL == shortURL {
			return true, nil
		}
	}
	return false, nil
}

// GetURLsByUserID возвращает все URL, сокращённые пользователем
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

func (m *Manager) BatchDelete(_ chan string, _ string) {
}

// Close закрывает соединение с базой
func (m *Manager) Close(_ context.Context) error {
	return nil
}

// Ping проверяет доступность хранилища
func (m *Manager) Ping(_ context.Context) error {
	return fmt.Errorf("ping is not supported for memory storage")
}
