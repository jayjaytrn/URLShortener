package main

import (
	"context"
	"fmt"
	"golang.org/x/crypto/acme/autocert"
	"net/http"

	"github.com/go-chi/chi/v5"
	pprof "github.com/go-chi/chi/v5/middleware"
	"github.com/jayjaytrn/URLShortener/config"
	"github.com/jayjaytrn/URLShortener/internal/auth"
	"github.com/jayjaytrn/URLShortener/internal/db"
	"github.com/jayjaytrn/URLShortener/internal/handlers"
	"github.com/jayjaytrn/URLShortener/internal/middleware"
	"github.com/jayjaytrn/URLShortener/logging"
	"go.uber.org/zap"
	// _ "net/http/pprof"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	if buildVersion == "" {
		buildVersion = "N/A"
	}
	if buildDate == "" {
		buildDate = "N/A"
	}
	if buildCommit == "" {
		buildCommit = "N/A"
	}

	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)

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

	if cfg.EnableHttps {
		manager := &autocert.Manager{
			// директория для хранения сертификатов
			Cache: autocert.DirCache("cache-dir"),
			// функция, принимающая Terms of Service издателя сертификатов
			Prompt: autocert.AcceptTOS,
			// перечень доменов, для которых будут поддерживаться сертификаты
			HostPolicy: autocert.HostWhitelist("mysite.ru", "www.mysite.ru"),
		}

		server := &http.Server{
			Addr:      ":443",
			Handler:   r,
			TLSConfig: manager.TLSConfig(),
		}
		err := server.ListenAndServeTLS("", "")
		logger.Fatalw("failed to start server", "error", err)
		return
	}
	err := http.ListenAndServe(cfg.ServerAddress, r)
	logger.Fatalw("failed to start server", "error", err)
}

func initRouter(h handlers.Handler, authManager *auth.Manager, storage db.ShortenerStorage, logger *zap.SugaredLogger) *chi.Mux {
	r := chi.NewRouter()
	r.Mount("/debug", pprof.Profiler())
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
