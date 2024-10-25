package memorystorage

import (
	"context"
	"fmt"
	"github.com/jayjaytrn/URLShortener/config"
	"github.com/jayjaytrn/URLShortener/internal/types"
)

type Manager struct {
	RelatesURLs map[string]string
}

func NewManager(_ *config.Config) (*Manager, error) {
	manager := &Manager{}
	manager.RelatesURLs = make(map[string]string)
	return manager, nil
}

func (m *Manager) GetOriginal(shortURL string) (string, error) {
	return m.RelatesURLs[shortURL], nil
}

func (m *Manager) Put(urlData types.URLData) error {
	m.RelatesURLs[urlData.ShortURL] = urlData.OriginalURL
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

// Exists возвращает true если запись найдена
func (m *Manager) Exists(url string) (bool, error) {
	_, ok := m.RelatesURLs[url]
	if ok {
		return true, nil
	}
	return false, nil
}

// Close закрывает соединение с базой
func (m *Manager) Close(_ context.Context) error {
	return nil
}

// Ping проверяет доступность хранилища
func (m *Manager) Ping(_ context.Context) error {
	return fmt.Errorf("ping is not supported for memory storage")
}
