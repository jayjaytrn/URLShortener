package storage

import (
	"bufio"
	"encoding/json"
	"github.com/jayjaytrn/URLShortener/config"
	"os"
)

var URLStorage []URLData
var WriteManager *Manager

type (
	URLData struct {
		UUID        string `json:"uuid"`
		ShortURL    string `json:"short_url"`
		OriginalURL string `json:"original_url"`
	}

	Manager struct {
		file *os.File
	}
)

func StartNewManager() error {
	file, err := os.OpenFile(config.Config.FileStoragePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	WriteManager = &Manager{file: file}
	return nil
}

func (p *Manager) Close() error {
	return p.file.Close()
}

func (p *Manager) WriteURL(url URLData) error {
	data, err := json.Marshal(&url)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	_, err = p.file.Write(data)
	if err != nil {
		return err
	}

	return err
}

func LoadURLStorageFromFile() error {
	file, err := os.Open(config.Config.FileStoragePath)
	if err != nil {
		if os.IsNotExist(err) {
			URLStorage = []URLData{}
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
		URLStorage = []URLData{}
		return nil
	}

	var scanner = bufio.NewScanner(file)
	for scanner.Scan() {
		var data URLData
		line := scanner.Bytes()
		if err = json.Unmarshal(line, &data); err != nil {
			return err
		}
		URLStorage = append(URLStorage, data)
	}

	if err = scanner.Err(); err != nil {
		return err
	}

	return nil
}

func (p *Manager) AddURL(url URLData) {
	URLStorage = append(URLStorage, url)
	p.WriteURL(url)
}
