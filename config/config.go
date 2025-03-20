package config

import (
	"encoding/json"
	"flag"
	"os"

	"github.com/caarlos0/env/v6"
	"github.com/jayjaytrn/URLShortener/logging"
)

// Config stores configuration settings for the URL shortener service.
type Config struct {
	ServerAddress   string `env:"SERVER_ADDRESS,required" json:"server_address"` // Server address to listen on
	BaseURL         string `env:"BASE_URL,required" json:"base_url"`             // Base URL for shortened links
	FileStoragePath string `env:"FILE_STORAGE_PATH" json:"file_storage_path"`    // Path to file storage (if used)
	DatabaseDSN     string `env:"DATABASE_DSN" json:"database_dsn"`              // Database connection string (if used)
	StorageType     string // Storage type: memory, file, or postgres (не загружается из JSON)
	EnableHTTPS     bool   `env:"ENABLE_HTTPS" json:"enable_https"` // Enable HTTPS
}

// GetConfig initializes and returns the application configuration.
// It parses command-line flags and environment variables to determine
// the storage type and other configuration settings.
func GetConfig() *Config {
	logger := logging.GetSugaredLogger()
	defer logger.Sync()

	config := &Config{}

	configFilePath := flag.String("c", os.Getenv("CONFIG"), "path to config file")

	// Parsing command-line flags

	flag.StringVar(&config.ServerAddress, "a", "localhost:8080", "server listen address")
	flag.StringVar(&config.BaseURL, "b", "http://localhost:8080", "short URL base")
	flag.StringVar(&config.FileStoragePath, "f", "storage.json", "file storage path")
	flag.StringVar(&config.DatabaseDSN, "d", "", "database DSN")
	flag.BoolVar(&config.EnableHTTPS, "s", false, "enable https")
	flag.Parse()

	// Parsing environment variables
	err := env.Parse(config)

	if *configFilePath != "" {
		jsonConfig, err := loadFromJSON(*configFilePath)
		if err != nil {
			logger.Debug("failed to load config from JSON:", err)
		} else {
			if config.ServerAddress == "" {
				config.ServerAddress = jsonConfig.ServerAddress
			}
			if config.BaseURL == "" {
				config.BaseURL = jsonConfig.BaseURL
			}
			if config.FileStoragePath == "" {
				config.FileStoragePath = jsonConfig.FileStoragePath
			}
			if config.DatabaseDSN == "" {
				config.DatabaseDSN = jsonConfig.DatabaseDSN
			}
			if !config.EnableHTTPS { // false по умолчанию, значит если false, то заменяем
				config.EnableHTTPS = jsonConfig.EnableHTTPS
			}
		}
	}
	if err != nil {
		logger.Debug("failed to parse environment variables:", err)
	}

	// Determine storage type based on available configuration
	if config.DatabaseDSN != "" {
		config.StorageType = "postgres"
		return config
	}

	if config.FileStoragePath != "" {
		config.StorageType = "file"
		return config
	}

	config.StorageType = "memory" // Default to in-memory storage
	return config
}

func loadFromJSON(filePath string) (*Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg Config
	decoder := json.NewDecoder(file)
	if err = decoder.Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
