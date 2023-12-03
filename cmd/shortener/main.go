package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/base64"
	json2 "encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/laiker/shortener/cmd/config"
	logger "github.com/laiker/shortener/internal"
	compresser "github.com/laiker/shortener/internal/gzip"
	"github.com/laiker/shortener/internal/json"
	"github.com/mailru/easyjson"
	"go.uber.org/zap"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var db *sql.DB

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

	db, err = sql.Open("pgx", config.DatabaseDsn)

	if err != nil {
		fmt.Println(err)
	}

	defer db.Close()

	if err != nil {
		fmt.Println(err)
	}

	r.Use(logger.RequestLogger, gzipMiddleware)
	r.HandleFunc("/api/shorten", shortenHandler)
	r.HandleFunc("/{id}", decodeHandler)
	r.HandleFunc("/ping", pingHandler)
	r.HandleFunc("/", encodeHandler)

	logger.Log.Info("Server runs at: ", zap.String("address", config.FlagRunAddr))
	http.ListenAndServe(config.FlagRunAddr, r)
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("pong"))
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
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	bodyURL := uri.String()

	encodedURL := encodeURL(bodyURL)
	err = SaveURL(string(encodedURL), bodyURL)

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

	bodyURL := string(reqURL)

	_, err = url.ParseRequestURI(bodyURL)

	if err != nil {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	response := encodeURL(bodyURL)

	logger.Log.Info(string(response) + " " + bodyURL)

	err = SaveURL(string(response), bodyURL)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	shortURL := fmt.Sprintf("%s/%s", config.FlagOutputURL, response)

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))

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
		return "", fmt.Errorf("wrong decode")
	}

	return string(data), nil
}

func encodeURL(url string) []byte {
	return []byte(base64.StdEncoding.EncodeToString([]byte(url)))
}

func SaveURL(short, original string) error {

	if config.DatabaseDsn != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		result, errexec := db.ExecContext(ctx, "INSERT INTO urls (original_url, short_url) VALUES ($1, $2)", original, short)

		if errexec != nil {
			return errexec
		}

		rows, err := result.RowsAffected()

		if err != nil {
			log.Fatal(err)
		}

		if rows != 1 {
			log.Fatalf("expected to affect 1 row, affected %d", rows)
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
