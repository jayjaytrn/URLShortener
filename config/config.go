package config

import (
	"flag"
	"fmt"
	"github.com/caarlos0/env/v6"
)

var Config struct {
	ServerAddress   string `env:"SERVER_ADDRESS,required"`
	BaseURL         string `env:"BASE_URL,required"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
}

func SetArgs() {
	flag.StringVar(&Config.ServerAddress, "a", "localhost:8080", "server listen address")
	flag.StringVar(&Config.BaseURL, "b", "http://localhost:8080", "short URL base")
	flag.StringVar(&Config.FileStoragePath, "f", "storage.json", "file storage path")

	err := env.Parse(&Config)
	if err != nil {
		fmt.Println("environments are not defined")
	}
}

func init() {
	SetArgs()
}
