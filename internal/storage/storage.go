package storage

import (
	"bufio"
	"encoding/json"
	"errors"
	"github.com/jayjaytrn/URLShortener/config"
	"os"
	"syscall"
)

var UrlStorage []URLData
var NewURLs []URLData

type (
	URLData struct {
		UUID        string `json:"uuid"`
		ShortUrl    string `json:"short_url"`
		OriginalUrl string `json:"original_url"`
	}

	Manager struct {
		file      *os.File
		lastIndex int
	}
)

func NewManager() (*Manager, error) {
	file, err := os.OpenFile(config.Config.FileStoragePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	return &Manager{file: file}, nil
}

func (p *Manager) Close() error {
	return p.file.Close()
}

func (p *Manager) WriteURLs() error {
	var data []byte
	var err error

	for _, u := range NewURLs {
		d, err := json.Marshal(&u)
		if err != nil {
			return err
		}
		data = append(data, d...)
		data = append(data, '\n')
	}

	_, err = p.file.Write(data)
	if err != nil {
		return err
	}

	err = p.Close()
	return err

}

func LoadURLStorageFromFile() error {
	file, err := os.Open(config.Config.FileStoragePath)
	if err != nil {
		if errors.Is(err, syscall.ERROR_FILE_NOT_FOUND) {
			UrlStorage = []URLData{}
			return nil
		}
		return err
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return err
	}

	if fi.Size() == 0 {
		UrlStorage = []URLData{}
		return nil
	}

	var scanner = bufio.NewScanner(file)
	for scanner.Scan() {
		var data URLData
		line := scanner.Bytes()
		if err = json.Unmarshal(line, &data); err != nil {
			return err
		}
		UrlStorage = append(UrlStorage, data)
	}

	if err = scanner.Err(); err != nil {
		return err
	}

	return nil
}

func AddURL(url URLData) {
	UrlStorage = append(UrlStorage, url)
	NewURLs = append(NewURLs, url)
}
