package main

import (
	"flag"
	"github.com/go-chi/chi/v5"
	"github.com/jayjaytrn/URLShortener/config"
	"github.com/jayjaytrn/URLShortener/internal/handlers"
	"github.com/jayjaytrn/URLShortener/internal/middleware"
	"github.com/jayjaytrn/URLShortener/internal/storage"
	"github.com/jayjaytrn/URLShortener/logging"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	flag.Parse()

	sugar := logging.GetSugaredLogger()
	defer sugar.Sync()

	err := storage.LoadURLStorageFromFile()
	if err != nil {
		panic(err)
	}

	mgr, err := storage.NewManager()
	if err != nil {
		panic(err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	r := chi.NewRouter()
	r.Post(`/`,
		func(w http.ResponseWriter, r *http.Request) {
			middleware.Conveyor(
				http.HandlerFunc(handlers.URLWaiter),
				sugar,
				middleware.WithLogging,
				middleware.WriteWithCompression,
				middleware.ReadWithCompression,
			).ServeHTTP(w, r)
		},
	)
	r.Post(`/api/shorten`,
		func(w http.ResponseWriter, r *http.Request) {
			middleware.Conveyor(
				http.HandlerFunc(handlers.Shorten),
				sugar,
				middleware.WithLogging,
				middleware.WriteWithCompression,
				middleware.ReadWithCompression,
			).ServeHTTP(w, r)
		},
	)

	r.Get(`/{id}`,
		func(w http.ResponseWriter, r *http.Request) {
			middleware.Conveyor(
				http.HandlerFunc(handlers.URLReturner),
				sugar,
				middleware.WithLogging,
				middleware.WriteWithCompression,
			).ServeHTTP(w, r)
		},
	)

	err = http.ListenAndServe(config.Config.ServerAddress, r)
	if err != nil {
		panic(err)
	}

	go func() {
		err = http.ListenAndServe(config.Config.ServerAddress, r)
		if err != nil {
			panic(err)
		}
	}()

	<-quit
	mgr.WriteURLs()
}
