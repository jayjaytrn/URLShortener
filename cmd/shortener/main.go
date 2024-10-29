package main

import (
	"context"
	"github.com/jayjaytrn/URLShortener/internal/auth"
	"github.com/jayjaytrn/URLShortener/internal/db"
	"go.uber.org/zap"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jayjaytrn/URLShortener/config"
	"github.com/jayjaytrn/URLShortener/internal/handlers"
	"github.com/jayjaytrn/URLShortener/internal/middleware"
	"github.com/jayjaytrn/URLShortener/logging"
)

func main() {
	logger := logging.GetSugaredLogger()
	defer logger.Sync()

	ctx := context.Background()

	authManager := auth.NewManager()

	cfg := config.GetConfig()

	s := db.GetStorage(cfg, logger)
	defer s.Close(ctx)

	h := handlers.Handler{
		Config:      cfg,
		Storage:     s,
		AuthManager: authManager,
	}

	r := initRouter(h, authManager, s, logger)

	err := http.ListenAndServe(cfg.ServerAddress, r)
	logger.Fatalw("failed to start server", "error", err)
}

func initRouter(h handlers.Handler, authManager *auth.Manager, storage db.ShortenerStorage, logger *zap.SugaredLogger) *chi.Mux {
	r := chi.NewRouter()
	r.Post(`/`,
		func(w http.ResponseWriter, r *http.Request) {
			middleware.Conveyor(
				http.HandlerFunc(h.URLWaiter),
				logger,
				middleware.WithLogging,
				middleware.WriteWithCompression,
				middleware.ReadWithCompression,
				func(next http.Handler, logger *zap.SugaredLogger) http.Handler {
					return middleware.WithAuth(next, authManager, storage, logger)
				},
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
				func(next http.Handler, _ *zap.SugaredLogger) http.Handler {
					return middleware.WithAuth(next, authManager, storage, logger)
				},
			).ServeHTTP(w, r)
		},
	)

	r.Post(`/api/shorten/batch`,
		func(w http.ResponseWriter, r *http.Request) {
			middleware.Conveyor(
				http.HandlerFunc(h.ShortenBatch),
				logger,
				middleware.WithLogging,
				middleware.WriteWithCompression,
				middleware.ReadWithCompression,
				func(next http.Handler, _ *zap.SugaredLogger) http.Handler {
					return middleware.WithAuth(next, authManager, storage, logger)
				},
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
				func(next http.Handler, _ *zap.SugaredLogger) http.Handler {
					return middleware.WithAuth(next, authManager, storage, logger)
				},
			).ServeHTTP(w, r)
		},
	)

	r.Get(`/ping`,
		func(w http.ResponseWriter, r *http.Request) {
			middleware.Conveyor(
				http.HandlerFunc(h.Ping),
				logger,
				middleware.WithLogging,
				middleware.WriteWithCompression,
			).ServeHTTP(w, r)
		},
	)

	r.Get(`/api/user/urls`,
		func(w http.ResponseWriter, r *http.Request) {
			middleware.Conveyor(
				http.HandlerFunc(h.Urls),
				logger,
				middleware.WithLogging,
				middleware.WriteWithCompression,
				func(next http.Handler, _ *zap.SugaredLogger) http.Handler {
					return middleware.WithAuth(next, authManager, storage, logger)
				},
			).ServeHTTP(w, r)
		},
	)

	r.Delete(`/api/user/urls`,
		func(w http.ResponseWriter, r *http.Request) {
			middleware.Conveyor(
				http.HandlerFunc(h.DeleteUrlsAsync),
				logger,
				middleware.WithLogging,
				middleware.WriteWithCompression,
				func(next http.Handler, _ *zap.SugaredLogger) http.Handler {
					return middleware.WithAuth(next, authManager, storage, logger)
				},
			).ServeHTTP(w, r)
		},
	)

	return r
}
