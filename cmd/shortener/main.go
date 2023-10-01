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

	reqUrl, err := io.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	uri, err := url.ParseRequestURI(string(reqUrl))
	if err != nil {
		http.Error(w, "Invalid Url", http.StatusBadRequest)
		return
	}

	str := base64.StdEncoding.EncodeToString([]byte(uri.String()))
	fmt.Println(str)

	w.WriteHeader(http.StatusCreated)
	w.Write(encodeUrl(uri.String(), r))
}

func decodeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	params := mux.Vars(r)
	result, err := decodeUrl(params["id"])

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Location", string(result))
	w.WriteHeader(http.StatusTemporaryRedirect)

}

func decodeUrl(code string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(code)

	if err != nil {
		return data, fmt.Errorf("wrong decode")
	}

	return data, nil
}

func encodeUrl(url string, r *http.Request) []byte {
	encodeStr := base64.StdEncoding.EncodeToString([]byte(url))
	return []byte(fmt.Sprintf("%v://%v/%v", getScheme(r), r.Host, encodeStr))
}

func getScheme(r *http.Request) string {
	if r.TLS == nil {
		return "http"
	}
	return "https"
}
