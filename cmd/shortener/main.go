package main

import (
	"encoding/base64"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"net/http"
	"net/url"
)

func main() {
	mux := mux.NewRouter()
	mux.HandleFunc("/{id}", decodeHandler)
	mux.HandleFunc("/", encodeHandler)
	http.ListenAndServe(`:8080`, mux)
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

	w.WriteHeader(http.StatusCreated)
	w.Write(encodeURL(uri.String(), r))
}

func decodeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	params := mux.Vars(r)

	result, err := decodeURL(params["id"])

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

func encodeURL(url string, r *http.Request) []byte {
	encodeStr := base64.StdEncoding.EncodeToString([]byte(url))
	return []byte(fmt.Sprintf("%v://%v/%v", getScheme(r), r.Host, encodeStr))
}

func getScheme(r *http.Request) string {
	if r.TLS == nil {
		return "http"
	}
	return "https"
}
