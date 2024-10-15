package handlers

import (
	"bytes"
	"encoding/json"
	"github.com/jayjaytrn/URLShortener/config"
	"github.com/jayjaytrn/URLShortener/internal/db"
	"github.com/jayjaytrn/URLShortener/internal/types"
	"github.com/jayjaytrn/URLShortener/internal/urlshort"
	"io"
	"net/http"
)

type (
	ShortenRequest struct {
		URL string `json:"url"`
	}

	ShortenResponse struct {
		Result string `json:"result"`
	}
)

type Handler struct {
	Storage db.ShortenerStorage
	Config  *config.Config
}

func (h *Handler) URLWaiter(res http.ResponseWriter, req *http.Request) {
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

	storageLastIndex, err := h.Storage.GetNextUUID()
	if err != nil {
		http.Error(res, "could not generate UUID", http.StatusInternalServerError)
		return
	}
	urlData := types.URLData{
		UUID:        storageLastIndex,
		OriginalURL: url,
		ShortURL:    su,
	}
	//storage.WriteManager.Put(us)
	err = h.Storage.Put(urlData)
	if err != nil {
		http.Error(res, "error when trying to put data in storage", http.StatusInternalServerError)
	}

	r := h.Config.BaseURL + "/" + su
	res.Header().Set("content-type", "text/plain")
	res.WriteHeader(http.StatusCreated)
	res.Write([]byte(r))
}

func (h *Handler) URLReturner(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(res, "only GET method is allowed", http.StatusBadRequest)
		return
	}

	shortURL := req.URL.Path[len("/"):] // Получаем короткий URL из пути

	// Получаем оригинальный URL
	originalURL, err := h.Storage.GetOriginal(shortURL)
	if err != nil {
		http.Error(res, "URL not found", http.StatusNotFound)
		return
	}

	// Перенаправляем на оригинальный URL
	res.Header().Set("Location", originalURL)
	res.WriteHeader(http.StatusTemporaryRedirect)
}

func (h *Handler) Shorten(res http.ResponseWriter, req *http.Request) {
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

	storageLastIndex, err := h.Storage.GetNextUUID()
	if err != nil {
		http.Error(res, "could not generate UUID", http.StatusInternalServerError)
		return
	}
	urlData := types.URLData{
		UUID:        storageLastIndex,
		OriginalURL: url,
		ShortURL:    su,
	}
	err = h.Storage.Put(urlData)
	if err != nil {
		http.Error(res, "error when trying to put data in storage", http.StatusInternalServerError)
	}

	r := h.Config.BaseURL + "/" + su
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
