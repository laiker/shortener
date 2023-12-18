package main

import (
	"bufio"
	"context"
	"encoding/base64"
	json2 "encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/laiker/shortener/cmd/config"
	logger "github.com/laiker/shortener/internal"
	compresser "github.com/laiker/shortener/internal/gzip"
	"github.com/laiker/shortener/internal/json"
	"github.com/laiker/shortener/internal/store"
	"github.com/mailru/easyjson"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// app инкапсулирует в себя все зависимости и логику приложения
type app struct {
	store store.Store
}

// newApp принимает на вход внешние зависимости приложения и возвращает новый объект app
func newApp(s store.Store) *app {
	return &app{
		store: s,
	}
}

func (a *app) pingHandler(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := a.store.PingContext(ctx); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("pong"))
}

func (a *app) gzipMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ow := w

		acceptContent := r.Header.Get("Content-Type")

		typesToCheck := []string{"application/json", "text/html", "text/plain", "application/x-gzip"}

		supportContent := false
		for _, contentType := range typesToCheck {
			if strings.Contains(acceptContent, contentType) {
				supportContent = true
				break
			}
		}

		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")

		if supportsGzip && supportContent {
			cw := compresser.NewCompressWriter(w)
			ow = cw
			defer cw.Close()
		}

		// проверяем, что клиент отправил серверу сжатые данные в формате gzip
		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")

		if sendsGzip && supportContent {
			cr, err := compresser.NewCompressReader(r.Body)

			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			r.Body = cr
			defer cr.Close()
		}

		h.ServeHTTP(ow, r)
	})
}

func (a *app) shortenHandler(w http.ResponseWriter, r *http.Request) {
	logger.Log.Info("shortenHandler")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
	}

	urlType := &json.URL{}
	easyjson.Unmarshal(body, urlType)

	uri, err := url.ParseRequestURI(urlType.URL)

	if err != nil {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	bodyURL := uri.String()

	encodedURL := a.encodeURL(bodyURL)

	err = a.SaveURL(string(encodedURL), bodyURL)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result := &json.Result{}
	result.Result = fmt.Sprintf("%s/%s", config.FlagOutputURL, encodedURL)

	response, err := easyjson.Marshal(result)

	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	w.Write(response)

}

func (a *app) shortenBatchHandler(w http.ResponseWriter, r *http.Request) {
	logger.Log.Info("shortenBatchHandler")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
	}

	var batchSlice json.BatchURLSlice

	err = easyjson.Unmarshal(body, &batchSlice)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	fmt.Println(batchSlice)
	result := make(json.BatchURLSlice, 0)
	saveBatch := make(json.BatchURLSlice, 0)

	for i := 0; i < len(batchSlice); i++ {
		currentItem := batchSlice[i]
		uri, err := url.ParseRequestURI(currentItem.OriginalURL)

		if err != nil {
			http.Error(w, "Invalid URL", http.StatusBadRequest)
			return
		}

		bodyURL := uri.String()

		encodedURL := a.encodeURL(bodyURL)

		err = a.SaveURL(string(encodedURL), bodyURL)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		batchSaveUrl := json.DBRow{
			ShortURL:    currentItem.ShortURL,
			OriginalURL: currentItem.OriginalURL,
		}

		batchOutputUrl := json.DBRow{
			CorrelationID: currentItem.CorrelationID,
			ShortURL:      fmt.Sprintf("%s/%s", config.FlagOutputURL, encodedURL),
		}

		saveBatch = append(saveBatch, batchSaveUrl)
		result = append(result, batchOutputUrl)
	}

	err = a.store.SaveBatchURL(context.Background(), saveBatch)

	if err != nil {
		return
	}

	response, err := easyjson.Marshal(result)

	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	w.Write(response)

}

func (a *app) encodeHandler(w http.ResponseWriter, r *http.Request) {
	logger.Log.Info("encodeHandler")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	reqURL, err := io.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	bodyURL := string(reqURL)

	_, err = url.ParseRequestURI(bodyURL)

	if err != nil {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	response := a.encodeURL(bodyURL)

	logger.Log.Info(string(response) + " " + bodyURL)

	err = a.SaveURL(string(response), bodyURL)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	shortURL := fmt.Sprintf("%s/%s", config.FlagOutputURL, response)
	logger.Log.Info(config.FlagOutputURL)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))

}

func (a *app) decodeHandler(w http.ResponseWriter, r *http.Request) {

	logger.Log.Info("decodeHandler")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	id := chi.URLParam(r, "id")

	result, err := a.decodeURL(id)

	if err != nil {
		http.Error(w, "Error: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Location", result)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (a *app) decodeURL(code string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(code)

	if err != nil {
		return "", fmt.Errorf("wrong decode")
	}

	return string(data), nil
}

func (a *app) encodeURL(url string) []byte {
	return []byte(base64.StdEncoding.EncodeToString([]byte(url)))
}

func (a *app) SaveURL(short, original string) error {

	if config.DatabaseDsn != "" {
		logger.Log.Info("Try to save url into Database: " + original)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		err := a.store.SaveURL(ctx, short, original)

		if err != nil {
			return err
		}
	}

	if config.StoragePath != "" {

		file, err := os.OpenFile(config.StoragePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0775)

		if err != nil {
			return err
		}

		encoder := json2.NewEncoder(file)

		reader := bufio.NewReader(file)

		var lastLine string

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break // Достигнут конец файла
			}
			lastLine = line
			if strings.TrimSpace(line) == "}" {
				break
			}
		}

		lastRow := &json.DBRow{}
		lastID := 1
		if lastLine != "" {
			if err := json2.Unmarshal([]byte(lastLine), &lastRow); err != nil {
				return err
			}

			lastID = lastRow.ID + 1
		}

		row := &json.DBRow{
			ID:          lastID,
			OriginalURL: original,
			ShortURL:    short,
		}

		err = encoder.Encode(row)

		if err != nil {
			return err
		}
	}

	return nil
}
