package config

import (
	"flag"
	"github.com/caarlos0/env/v6"
	"github.com/jayjaytrn/URLShortener/logging"
)

type Config struct {
	ServerAddress   string `env:"SERVER_ADDRESS,required"`
	BaseURL         string `env:"BASE_URL,required"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	DatabaseDSN     string `env:"DATABASE_DSN"`
	StorageType     string
}

func GetConfig() *Config {
	logger := logging.GetSugaredLogger()
	defer logger.Sync()

	config := &Config{}

	flag.Parse()

	err := env.Parse(config)
	if err != nil {
		logger.Debug("Failed to parse environment variables:", err)

		flag.StringVar(&config.ServerAddress, "a", "localhost:8080", "server listen address")
		flag.StringVar(&config.BaseURL, "b", "http://localhost:8080", "short URL base")
		flag.StringVar(&config.FileStoragePath, "f", "", "file storage path")
		flag.StringVar(&config.DatabaseDSN, "d", "", "database DSN")

		if config.FileStoragePath == "" {
			config.StorageType = "memory"
			return config
		}
		config.StorageType = "file"
	}

	config.StorageType = "db"
	return config
}
