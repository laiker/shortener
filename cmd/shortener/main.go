package main

import (
	"database/sql"
	"fmt"
	"github.com/go-chi/chi"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/laiker/shortener/cmd/config"
	logger "github.com/laiker/shortener/internal"
	"github.com/laiker/shortener/internal/store/pg"
	"go.uber.org/zap"
	"net/http"
)

func main() {
	config.ParseFlags()
	run()
}

func run() {
	var db *sql.DB

	r := chi.NewRouter()

	err := logger.Initialize(config.FlagLogLevel)
	if err != nil {
		fmt.Println(err)
	}

	if config.DatabaseDsn != "" {
		db, err = sql.Open("pg", config.DatabaseDsn)

		if err != nil {
			fmt.Println(err)
		}

		defer db.Close()
	}

	appInstance := newApp(pg.NewStore(db))

	r.Use(logger.RequestLogger, appInstance.gzipMiddleware)
	r.HandleFunc("/api/shorten", appInstance.shortenHandler)
	r.HandleFunc("/{id}", appInstance.decodeHandler)
	r.HandleFunc("/ping", appInstance.pingHandler)
	r.HandleFunc("/", appInstance.encodeHandler)

	logger.Log.Info("Server runs at: ", zap.String("address", config.FlagRunAddr))
	http.ListenAndServe(config.FlagRunAddr, r)
}
