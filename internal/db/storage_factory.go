package db

import (
	"github.com/jayjaytrn/URLShortener/config"
	"github.com/jayjaytrn/URLShortener/internal/db/filestorage"
	"github.com/jayjaytrn/URLShortener/internal/db/memorystorage"
	"github.com/jayjaytrn/URLShortener/internal/db/postgres"
	"go.uber.org/zap"
)

// GetStorage initializes and returns a storage manager based on the configured storage type.
func GetStorage(cfg *config.Config, logger *zap.SugaredLogger) ShortenerStorage {
	// Initialize file-based storage
	if cfg.StorageType == "file" {
		logger.Debug("using file storage")
		s, err := filestorage.NewManager(cfg)
		if err != nil {
			logger.Fatalw("failed to initialize file storage", "error", err)
		}
		return s
	}

	// Initialize PostgreSQL-based storage
	if cfg.StorageType == "postgres" {
		logger.Debug("using postgres storage")
		s, err := postgres.NewManager(cfg)
		if err != nil {
			logger.Fatalw("failed to initialize postgres storage", "error", err)
		}
		return s
	}

	// Initialize in-memory storage
	if cfg.StorageType == "memory" {
		logger.Debug("using memory storage")
		s, err := memorystorage.NewManager(cfg)
		if err != nil {
			logger.Fatalw("failed to initialize memory storage", "error", err)
		}
		return s
	}

	// Handle unknown storage types
	logger.Fatalw("unknown storage type", "type", cfg.StorageType)
	return nil
}
