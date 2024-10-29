package filestorage

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/jayjaytrn/URLShortener/config"
	"github.com/jayjaytrn/URLShortener/internal/types"
	"os"
)

type Manager struct {
	file        *os.File
	FileStorage *[]types.URLData
	cfg         *config.Config
}

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

func (fm *Manager) GetOriginal(shortURL string) (string, error) {
	for _, urlData := range *fm.FileStorage {
		if urlData.ShortURL == shortURL {
			return urlData.OriginalURL, nil
		}
	}
	return "", fmt.Errorf("URL not found")
}

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

func (fm *Manager) PutBatch(_ context.Context, batchData []types.URLData) error {
	for _, urlData := range batchData {
		err := fm.Put(urlData)
		if err != nil {
			return err
		}
	}
	return nil
}

func (fm *Manager) Exists(shortURL string) (bool, error) {
	for _, urlData := range *fm.FileStorage {
		if urlData.ShortURL == shortURL {
			return true, nil
		}
	}
	return false, nil
}

func (fm *Manager) GenerateNewUserID() string {
	return uuid.New().String()
}

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

	// Если нет URL для данного пользователя, возвращаем пустой список
	if len(userURLs) == 0 {
		return nil, fmt.Errorf("no URLs found for userID: %s", userID)
	}

	return userURLs, nil
}

func (fm *Manager) Ping(ctx context.Context) error {
	return fmt.Errorf("ping is not supported for file storage")
}

func (fm *Manager) Close(_ context.Context) error {
	return fm.file.Close()
}

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

func (fm *Manager) LoadURLStorageFromFile() error {
	fi, err := fm.file.Stat()
	if err != nil {
		return err
	}

	if fi.Size() == 0 {
		*fm.FileStorage = []types.URLData{}
		return nil
	}

	// Перемещаем указатель на начало файла
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
