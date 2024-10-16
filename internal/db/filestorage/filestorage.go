package filestorage

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/jayjaytrn/URLShortener/config"
	"github.com/jayjaytrn/URLShortener/internal/types"
)

type StorageData struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type Manager struct {
	file        *os.File
	FileStorage *[]StorageData
}

func NewManager(cfg *config.Config) (*Manager, error) {
	file, err := os.OpenFile(cfg.FileStoragePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	var storage []StorageData

	fm := &Manager{
		file:        file,
		FileStorage: &storage,
	}

	err = fm.LoadURLStorageFromFile()
	if err != nil {
		return nil, fmt.Errorf("failed to load URL storage from file: %w", err)
	}

	return fm, nil
}

func (fm *Manager) GetShort(originalURL string) (string, error) {
	for _, urlData := range *fm.FileStorage {
		if urlData.OriginalURL == originalURL {
			return urlData.ShortURL, nil
		}
	}
	return "", fmt.Errorf("URL not found")
}

func (fm *Manager) GetOriginal(shortURL string) (string, error) {
	for _, urlData := range *fm.FileStorage {
		if urlData.ShortURL == shortURL {
			return urlData.OriginalURL, nil
		}
	}
	return "", fmt.Errorf("URL not found")
}

func (fm *Manager) Put(urlData types.URLData) (bool, error) {
	// длина стораджа будет на 1 больше чем его UUID значение
	storageLastIndex := len(*fm.FileStorage)
	data := StorageData{
		UUID:        strconv.Itoa(storageLastIndex),
		ShortURL:    urlData.ShortURL,
		OriginalURL: urlData.OriginalURL,
	}
	*fm.FileStorage = append(*fm.FileStorage, data)
	err := fm.WriteURL(urlData)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (fm *Manager) PutBatch(_ context.Context, batchData []types.URLData) error {
	for _, urlData := range batchData {
		_, err := fm.Put(urlData)
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

func (fm *Manager) LoadURLStorageFromFile() error {
	fi, err := fm.file.Stat()
	if err != nil {
		return err
	}

	if fi.Size() == 0 {
		*fm.FileStorage = []StorageData{}
		return nil
	}

	// Перемещаем указатель на начало файла
	_, err = fm.file.Seek(0, 0)
	if err != nil {
		return err
	}

	var scanner = bufio.NewScanner(fm.file)
	for scanner.Scan() {
		var data StorageData
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
