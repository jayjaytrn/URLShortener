package main

import (
	"flag"
	"github.com/go-chi/chi/v5"
	"github.com/jayjaytrn/URLShortener/config"
	"github.com/jayjaytrn/URLShortener/internal/handlers"
	"github.com/jayjaytrn/URLShortener/logging"
	"net/http"
)

func main() {
	flag.Parse()

	sugar := logging.GetSugaredLogger()
	defer sugar.Sync()

	r := chi.NewRouter()
	r.Post(`/`, handlers.WithLogging(handlers.URLWaiter, sugar))
	r.Post(`/api/shorten`, handlers.WithLogging(handlers.URLWaiter, sugar))
	r.Get(`/{id}`, handlers.WithLogging(handlers.URLReturner, sugar))

	err := http.ListenAndServe(config.Config.ServerAddress, r)
	if err != nil {
		panic(err)
	}
}
