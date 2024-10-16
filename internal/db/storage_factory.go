package db

import (
	"github.com/jayjaytrn/URLShortener/config"
	"github.com/jayjaytrn/URLShortener/internal/db/filestorage"
	"github.com/jayjaytrn/URLShortener/internal/db/memorystorage"
	"github.com/jayjaytrn/URLShortener/internal/db/postgres"
	"go.uber.org/zap"
)

func GetStorage(cfg *config.Config, logger *zap.SugaredLogger) ShortenerStorage {
	if cfg.StorageType == "file" {
		logger.Debug("using file storage")
		s, err := filestorage.NewManager(cfg)
		if err != nil {
			logger.Fatalw("failed to initialize file storage", "error", err)
		}
		return s
	}

	if cfg.StorageType == "postgres" {
		logger.Debug("using postgres storage")
		s, err := postgres.NewManager(cfg)
		if err != nil {
			logger.Fatalw("failed to initialize file storage", "error", err)
		}
		return s
	}

	if cfg.StorageType == "memory" {
		logger.Debug("using memory storage")
		s, err := memorystorage.NewManager(cfg)
		if err != nil {
			logger.Fatalw("failed to initialize file storage", "error", err)
		}
		return s
	}

	logger.Fatalw("unknown storage type", "type", cfg.StorageType)
	return nil
}
