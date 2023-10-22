package main

import (
	"encoding/base64"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/laiker/shortener/cmd/config"
	logger "github.com/laiker/shortener/internal"
	"github.com/laiker/shortener/internal/json"
	"github.com/mailru/easyjson"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/url"
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

	r.Use(logger.RequestLogger)
	r.HandleFunc("/{id}", decodeHandler)
	r.HandleFunc("/", encodeHandler)
	r.HandleFunc("/api/shorten", shortenHandler)

	logger.Log.Info("Server runs at: ", zap.String("address", config.FlagRunAddr))
	http.ListenAndServe(config.FlagRunAddr, r)
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

	urlType := &json.Url{}
	easyjson.Unmarshal(body, urlType)

	uri, err := url.ParseRequestURI(urlType.Url)

	if err != nil {
		http.Error(w, "Invalid Url", http.StatusBadRequest)
		return
	}

	encodedUrl := encodeURL(uri.String())

	result := &json.Result{}

	result.Result = string(encodedUrl)

	response, err := easyjson.Marshal(result)

	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
	}

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

	uri, err := url.ParseRequestURI(string(reqURL))
	if err != nil {
		http.Error(w, "Invalid Url", http.StatusBadRequest)
		return
	}

	base64.StdEncoding.EncodeToString([]byte(uri.String()))

	response := encodeURL(uri.String())

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
