package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/jayjaytrn/URLShortener/config"
	"io"
	"math/rand"
	"net/http"
	"regexp"
	"time"
)

var relatesURLs = make(map[string]string)

func urlWaiter(res http.ResponseWriter, req *http.Request) {
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
	valid := validateURL(url)
	if !valid {
		http.Error(res, "wrong parameters", http.StatusBadRequest)
		return
	}

	su := generateShortURL()
	relatesURLs[su] = url
	r := config.Config.BaseURL + "/" + su
	res.Header().Set("content-type", "text/plain")
	res.WriteHeader(http.StatusCreated)
	res.Write([]byte(r))
}

func urlReturner(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(res, "only GET method is allowed", http.StatusBadRequest)
		return
	}
	shortURL := req.URL.Path[len("/"):]
	originalURL, exists := relatesURLs[shortURL]
	if !exists {
		http.Error(res, "not found", http.StatusBadRequest)
		return
	}

	res.Header().Set("Location", originalURL)
	res.WriteHeader(http.StatusTemporaryRedirect)
}

func generateShortURL() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const keyLength = 8

	rand.New(rand.NewSource(time.Now().UnixNano()))

	shortURL := make([]byte, keyLength)
	for {
		for i := range shortURL {
			shortURL[i] = charset[rand.Intn(len(charset))]
		}

		if _, exists := relatesURLs[string(shortURL)]; !exists {
			break
		}
	}
	return string(shortURL)
}

func validateURL(url string) bool {
	regex := `^https?://([a-zA-Z0-9-]+\.)*[a-zA-Z0-9-]+\.[a-zA-Z]{2,}(/.*)?$`

	m, _ := regexp.MatchString(regex, url)
	return m
}

func main() {
	r := chi.NewRouter()
	r.Post(`/`, urlWaiter)
	r.Get(`/{id}`, urlReturner)

	err := http.ListenAndServe(config.Config.ServerAddress, r)
	if err != nil {
		panic(err)
	}
}

func init() {
	config.SetArgs()
}
