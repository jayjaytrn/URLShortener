package main

import (
	"flag"
	"github.com/go-chi/chi/v5"
	"github.com/jayjaytrn/URLShortener/config"
	"github.com/jayjaytrn/URLShortener/internal/handlers"
	"net/http"
)

func main() {
	config.SetArgs()
	flag.Parse()
	r := chi.NewRouter()
	r.Post(`/`, handlers.UrlWaiter)
	r.Get(`/{id}`, handlers.UrlReturner)

	err := http.ListenAndServe(config.Config.ServerAddress, r)
	if err != nil {
		panic(err)
	}
}
