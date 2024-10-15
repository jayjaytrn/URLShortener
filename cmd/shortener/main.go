package main

import (
	"context"
	"flag"
	"github.com/go-chi/chi/v5"
	"github.com/jayjaytrn/URLShortener/config"
	"github.com/jayjaytrn/URLShortener/internal/db"
	"github.com/jayjaytrn/URLShortener/internal/db/filestorage"
	"github.com/jayjaytrn/URLShortener/internal/db/postgres"
	"github.com/jayjaytrn/URLShortener/internal/handlers"
	"github.com/jayjaytrn/URLShortener/internal/middleware"
	"github.com/jayjaytrn/URLShortener/logging"
	"go.uber.org/zap"
	"net/http"
)

func main() {
	flag.Parse()

	logger := logging.GetSugaredLogger()
	defer logger.Sync()

	ctx := context.Background()

	cfg := config.GetConfig()

	s := GetStorage(cfg, logger)
	defer s.Close(ctx)

	h := handlers.Handler{
		Config:  cfg,
		Storage: s,
	}

	r := chi.NewRouter()
	r.Post(`/`,
		func(w http.ResponseWriter, r *http.Request) {
			middleware.Conveyor(
				http.HandlerFunc(h.URLWaiter),
				logger,
				middleware.WithLogging,
				middleware.WriteWithCompression,
				middleware.ReadWithCompression,
			).ServeHTTP(w, r)
		},
	)
	r.Post(`/api/shorten`,
		func(w http.ResponseWriter, r *http.Request) {
			middleware.Conveyor(
				http.HandlerFunc(h.Shorten),
				logger,
				middleware.WithLogging,
				middleware.WriteWithCompression,
				middleware.ReadWithCompression,
			).ServeHTTP(w, r)
		},
	)

	r.Get(`/{id}`,
		func(w http.ResponseWriter, r *http.Request) {
			middleware.Conveyor(
				http.HandlerFunc(h.URLReturner),
				logger,
				middleware.WithLogging,
				middleware.WriteWithCompression,
			).ServeHTTP(w, r)
		},
	)

	err := http.ListenAndServe(cfg.ServerAddress, r)
	logger.Fatalw("failed to start server", "error", err)
}

func GetStorage(cfg *config.Config, logger *zap.SugaredLogger) db.ShortenerStorage {
	if cfg.DatabaseDSN == "" {
		logger.Debug("database DSN not provided, using file storage")
		s, err := filestorage.NewFileManager(cfg)
		if err != nil {
			logger.Fatalw("failed to initialize file storage", "error", err)
		}
		return s
	}
	s, err := postgres.NewManager(cfg)
	if err != nil {
		logger.Fatalw("failed to initialize postgres database", "error", err)
	}
	return s
}
