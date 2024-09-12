package handlers

import (
	"github.com/jayjaytrn/URLShortener/config"
	"github.com/jayjaytrn/URLShortener/internal/db"
	"github.com/jayjaytrn/URLShortener/internal/urlshort"
	"io"
	"net/http"
)

func UrlWaiter(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(res, "only POST method is allowed", http.StatusBadRequest)
		return
	}

	req.FormValue("url")
	res.Header().Set("Content-Type", "text/plain")

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, "error when read body", http.StatusBadRequest)
		return
	}
	url := string(body)
	valid := urlshort.ValidateURL(url)
	if !valid {
		http.Error(res, "wrong parameters", http.StatusBadRequest)
		return
	}

	su := urlshort.GenerateShortURL()
	db.RelatesURLs[su] = url
	r := config.Config.BaseURL + "/" + su
	res.Header().Set("content-type", "text/plain")
	res.WriteHeader(http.StatusCreated)
	res.Write([]byte(r))
}

func UrlReturner(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(res, "only GET method is allowed", http.StatusBadRequest)
		return
	}
	shortURL := req.URL.Path[len("/"):]
	originalURL, exists := db.RelatesURLs[shortURL]
	if !exists {
		http.Error(res, "not found", http.StatusBadRequest)
		return
	}

	res.Header().Set("Location", originalURL)
	res.WriteHeader(http.StatusTemporaryRedirect)
}
