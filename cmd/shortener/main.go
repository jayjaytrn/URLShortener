package main

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/crypto/acme/autocert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	pprof "github.com/go-chi/chi/v5/middleware"
	"github.com/jayjaytrn/URLShortener/config"
	"github.com/jayjaytrn/URLShortener/internal/auth"
	"github.com/jayjaytrn/URLShortener/internal/db"
	"github.com/jayjaytrn/URLShortener/internal/handlers"
	"github.com/jayjaytrn/URLShortener/internal/middleware"
	"github.com/jayjaytrn/URLShortener/logging"
	pb "github.com/jayjaytrn/URLShortener/proto"
	"go.uber.org/zap"
	//
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

	server := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: r,
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// gRPC сервер
	grpcServer := grpc.NewServer()
	pb.RegisterURLShortenerServer(grpcServer, handlers.NewURLShortener(s, authManager, cfg))

	// Register reflection service on gRPC server.
	reflection.Register(grpcServer)

	go func() {
		logger.Infow("starting server", "address", cfg.ServerAddress)
		var err error
		if cfg.EnableHTTPS {
			manager := &autocert.Manager{
				Cache:      autocert.DirCache("cache-dir"),
				Prompt:     autocert.AcceptTOS,
				HostPolicy: autocert.HostWhitelist("mysite.ru", "www.mysite.ru"),
			}
			server.TLSConfig = manager.TLSConfig()
			err = server.ListenAndServeTLS("", "")
		} else {
			err = server.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatalw("server error", "error", err)
		}
	}()

	// Запуск gRPC сервера
	go func() {
		lis, err := net.Listen("tcp", ":50051")
		if err != nil {
			logger.Fatalw("failed to listen for gRPC", "error", err)
		}

		logger.Infow("starting gRPC server", "address", ":50051")
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatalw("failed to serve gRPC", "error", err)
		}
	}()

	sig := <-sigChan
	logger.Infow("received shutdown signal", "signal", sig)

	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 5*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Errorw("server shutdown error", "error", err)
	}

	logger.Infow("server gracefully stopped")
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

	r.Get(`/api/internal/stats`,
		func(w http.ResponseWriter, r *http.Request) {
			middleware.Conveyor(
				http.HandlerFunc(h.Stats),
				logger,
				middleware.WithLogging,
				func(next http.Handler, _ *zap.SugaredLogger) http.Handler {
					return middleware.WithAuth(next, authManager, storage, logger)
				},
			).ServeHTTP(w, r)
		},
	)

	return r
}
