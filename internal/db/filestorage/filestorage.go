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

type FileManager struct {
	file       *os.File
	URLStorage *[]types.URLData
}

func NewFileManager(cfg *config.Config) (*FileManager, error) {
	file, err := os.OpenFile(cfg.FileStoragePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	var storage []types.URLData

	fm := &FileManager{
		file:       file,
		URLStorage: &storage,
	}

	err = fm.LoadURLStorageFromFile()
	if err != nil {
		return nil, fmt.Errorf("failed to load URL storage from file: %w", err)
	}

	return fm, nil
}

func (fm *FileManager) GetOriginal(shortURL string) (string, error) {
	for _, urlData := range *fm.URLStorage {
		if urlData.ShortURL == shortURL {
			return urlData.OriginalURL, nil
		}
	}
	return "", fmt.Errorf("URL not found")
}

func (fm *FileManager) Put(urlData types.URLData) error {
	*fm.URLStorage = append(*fm.URLStorage, urlData)
	err := fm.WriteURL(urlData)
	if err != nil {
		return err
	}
	return nil
}

func (fm *FileManager) Exists(shortURL string) (bool, error) {
	for _, urlData := range *fm.URLStorage {
		if urlData.ShortURL == shortURL {
			return true, nil
		}
	}
	return false, nil
}

func (fm *FileManager) GetNextUUID() (string, error) {
	// UUID — это индекс следующего элемента в слайсе
	nextUUID := strconv.Itoa(len(*fm.URLStorage))
	return nextUUID, nil
}

func (fm *FileManager) Ping(ctx context.Context) error {
	return fmt.Errorf("ping is not supported for file storage")
}

func (fm *FileManager) Close(_ context.Context) error {
	return fm.file.Close()
}

func (fm *FileManager) WriteURL(urlData types.URLData) error {
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

func (fm *FileManager) LoadURLStorageFromFile() error {
	fi, err := fm.file.Stat()
	if err != nil {
		return err
	}

	if fi.Size() == 0 {
		*fm.URLStorage = []types.URLData{}
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
		*fm.URLStorage = append(*fm.URLStorage, data)
	}

	if err = scanner.Err(); err != nil {
		return err
	}

	return nil
}
