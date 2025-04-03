package filestorage

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/jayjaytrn/URLShortener/config"
	"github.com/jayjaytrn/URLShortener/internal/types"
)

// Manager handles file-based URL storage operations.
type Manager struct {
	file        *os.File
	FileStorage *[]types.URLData
	cfg         *config.Config
}

// NewManager creates a new instance of the file storage manager.
//
// It opens the specified storage file and loads existing URLs into memory.
func NewManager(cfg *config.Config) (*Manager, error) {
	file, err := os.OpenFile(cfg.FileStoragePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	var storage []types.URLData

	fm := &Manager{
		file:        file,
		FileStorage: &storage,
		cfg:         cfg,
	}

	err = fm.LoadURLStorageFromFile()
	if err != nil {
		return nil, fmt.Errorf("failed to load URL storage from file: %w", err)
	}

	return fm, nil
}

// GetOriginal retrieves the original URL corresponding to a given short URL.
func (fm *Manager) GetOriginal(shortURL string) (string, error) {
	for _, urlData := range *fm.FileStorage {
		if urlData.ShortURL == shortURL {
			return urlData.OriginalURL, nil
		}
	}
	return "", fmt.Errorf("URL not found")
}

// Put stores a new URL mapping in the file storage.
func (fm *Manager) Put(urlData types.URLData) error {
	data := types.URLData{
		ShortURL:    urlData.ShortURL,
		OriginalURL: urlData.OriginalURL,
		UserID:      urlData.UserID,
	}
	*fm.FileStorage = append(*fm.FileStorage, data)
	err := fm.WriteURL(urlData)
	if err != nil {
		return err
	}
	return nil
}

// PutBatch stores multiple URL mappings in the file storage.
func (fm *Manager) PutBatch(_ context.Context, batchData []types.URLData) error {
	for _, urlData := range batchData {
		err := fm.Put(urlData)
		if err != nil {
			return err
		}
	}
	return nil
}

// Exists checks if a given short URL exists in the storage.
func (fm *Manager) Exists(shortURL string) (bool, error) {
	for _, urlData := range *fm.FileStorage {
		if urlData.ShortURL == shortURL {
			return true, nil
		}
	}
	return false, nil
}

// GenerateNewUserID generates a new unique user ID.
func (fm *Manager) GenerateNewUserID() string {
	return uuid.New().String()
}

// GetURLsByUserID retrieves all stored URLs associated with a given user ID.
func (fm *Manager) GetURLsByUserID(userID string) ([]types.URLData, error) {
	var userURLs []types.URLData

	for _, urlData := range *fm.FileStorage {
		if urlData.UserID == userID {
			userURLs = append(userURLs, types.URLData{
				ShortURL:    fm.cfg.BaseURL + "/" + urlData.ShortURL,
				OriginalURL: urlData.OriginalURL,
			})
		}
	}

	// If no URLs are found for the user, return an error.
	if len(userURLs) == 0 {
		return nil, fmt.Errorf("no URLs found for userID: %s", userID)
	}

	return userURLs, nil
}

// Ping checks the availability of the storage, not supported for file storage.
func (fm *Manager) Ping(ctx context.Context) error {
	return fmt.Errorf("ping is not supported for file storage")
}

// Close closes the storage file.
func (fm *Manager) Close(_ context.Context) error {
	return fm.file.Close()
}

// WriteURL appends a new URL entry to the storage file.
func (fm *Manager) WriteURL(urlData types.URLData) error {
	data, err := json.Marshal(&urlData)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	_, err = fm.file.Write(data)
	if err != nil {
		return err
	}

	return err
}

// GetNewUserID generates a new unique user ID, ensuring it does not exist in storage.
func (fm *Manager) GetNewUserID() (string, error) {
	for {
		newUUID := uuid.New().String()

		collision := false
		for _, data := range *fm.FileStorage {
			if data.UserID == newUUID {
				collision = true
				break
			}
		}

		if !collision {
			return newUUID, nil
		}
	}
}

// BatchDelete is a placeholder for batch deletion, not implemented for file storage.
func (fm *Manager) BatchDelete(_ chan string, _ string) {
}

// LoadURLStorageFromFile reads stored URLs from the file and loads them into memory.
func (fm *Manager) LoadURLStorageFromFile() error {
	fi, err := fm.file.Stat()
	if err != nil {
		return err
	}

	if fi.Size() == 0 {
		*fm.FileStorage = []types.URLData{}
		return nil
	}

	// Move file pointer to the beginning of the file.
	_, err = fm.file.Seek(0, 0)
	if err != nil {
		return err
	}

	var scanner = bufio.NewScanner(fm.file)
	for scanner.Scan() {
		var data types.URLData
		line := scanner.Bytes()
		if err = json.Unmarshal(line, &data); err != nil {
			return err
		}
		*fm.FileStorage = append(*fm.FileStorage, data)
	}

	if err = scanner.Err(); err != nil {
		return err
	}

	return nil
}

// GetStats возвращает количество сокращенных URL и количество пользователей.
func (fm *Manager) GetStats() (types.Stats, error) {
	urlCount := len(*fm.FileStorage)
	userSet := make(map[string]struct{})

	for _, urlData := range *fm.FileStorage {
		if urlData.UserID != "" {
			userSet[urlData.UserID] = struct{}{}
		}
	}

	return types.Stats{
		Urls:  urlCount,
		Users: len(userSet),
	}, nil
}
