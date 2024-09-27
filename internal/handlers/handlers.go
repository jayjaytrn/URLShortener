package handlers

import (
	"bytes"
	"encoding/json"
	"github.com/jayjaytrn/URLShortener/config"
	"github.com/jayjaytrn/URLShortener/internal/storage"
	"github.com/jayjaytrn/URLShortener/internal/urlshort"
	"io"
	"net/http"
	"strconv"
)

type (
	ShortenRequest struct {
		URL string `json:"url"`
	}

	ShortenResponse struct {
		Result string `json:"result"`
	}
)

func URLWaiter(res http.ResponseWriter, req *http.Request) {
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

	storageLastIndex := len(storage.URLStorage)
	us := storage.URLData{
		UUID:        strconv.Itoa(storageLastIndex),
		OriginalURL: url,
		ShortURL:    su,
	}
	storage.WriteManager.AddURL(us)

	r := config.Config.BaseURL + "/" + su
	res.Header().Set("content-type", "text/plain")
	res.WriteHeader(http.StatusCreated)
	res.Write([]byte(r))
}

func URLReturner(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(res, "only GET method is allowed", http.StatusBadRequest)
		return
	}
	shortURL := req.URL.Path[len("/"):]

	originalURL := ""
	for _, urlData := range storage.URLStorage {
		if urlData.ShortURL == shortURL {
			originalURL = urlData.OriginalURL
			res.Header().Set("Location", originalURL)
			res.WriteHeader(http.StatusTemporaryRedirect)
			return
		}
	}

	http.Error(res, "not found", http.StatusBadRequest)
}

func Shorten(res http.ResponseWriter, req *http.Request) {
	var buf bytes.Buffer
	_, err := buf.ReadFrom(req.Body)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	var shortenRequest ShortenRequest
	err = json.Unmarshal(buf.Bytes(), &shortenRequest)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	url := shortenRequest.URL
	valid := urlshort.ValidateURL(url)
	if !valid {
		http.Error(res, "wrong parameters", http.StatusBadRequest)
		return
	}

	su := urlshort.GenerateShortURL()

	storageLastIndex := len(storage.URLStorage)
	us := storage.URLData{
		UUID:        strconv.Itoa(storageLastIndex),
		OriginalURL: url,
		ShortURL:    su,
	}
	storage.WriteManager.AddURL(us)

	r := config.Config.BaseURL + "/" + su
	shortenResponse := ShortenResponse{
		Result: r,
	}
	br, err := json.Marshal(shortenResponse)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)
	res.Write(br)

}
