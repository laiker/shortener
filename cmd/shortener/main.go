package main

import (
	"encoding/base64"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/laiker/shortener/cmd/config"
	logger "github.com/laiker/shortener/internal"
	compresser "github.com/laiker/shortener/internal/gzip"
	"github.com/laiker/shortener/internal/json"
	"github.com/mailru/easyjson"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func main() {
	config.ParseFlags()
	run()
}

func run() {
	r := chi.NewRouter()

	err := logger.Initialize(config.FlagLogLevel)
	if err != nil {
		fmt.Println(err)
	}

	r.Use(logger.RequestLogger, gzipMiddleware)
	r.HandleFunc("/api/shorten", shortenHandler)
	r.HandleFunc("/{id}", decodeHandler)
	r.HandleFunc("/", encodeHandler)

	logger.Log.Info("Server runs at: ", zap.String("address", config.FlagRunAddr))
	http.ListenAndServe(config.FlagRunAddr, r)
}

func gzipMiddleware(h http.Handler) http.Handler {
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

func shortenHandler(w http.ResponseWriter, r *http.Request) {

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
		http.Error(w, "Invalid Url", http.StatusBadRequest)
		return
	}

	encodedURL := encodeURL(uri.String())

	result := &json.Result{}

	result.Result = string(encodedURL)

	response, err := easyjson.Marshal(result)

	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	w.Write(response)

}

func encodeHandler(w http.ResponseWriter, r *http.Request) {
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

	bodyUrl := string(reqURL)

	_, err = url.ParseRequestURI(bodyUrl)

	if err != nil {
		http.Error(w, "Invalid Url", http.StatusBadRequest)
		return
	}

	response := encodeURL(bodyUrl)

	w.WriteHeader(http.StatusCreated)
	w.Write(response)

}

func decodeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	id := chi.URLParam(r, "id")

	result, err := decodeURL(id)

	if err != nil {
		http.Error(w, "Error: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Location", result)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func decodeURL(code string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(code)

	if err != nil {
		return "error", fmt.Errorf("wrong decode")
	}

	return string(data), nil
}

func encodeURL(url string) []byte {
	encodeStr := base64.StdEncoding.EncodeToString([]byte(url))
	return []byte(fmt.Sprintf("%v/%v", config.FlagOutputURL, encodeStr))
}
