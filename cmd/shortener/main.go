package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/go-chi/chi"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/laiker/shortener/cmd/config"
	logger "github.com/laiker/shortener/internal"
	store "github.com/laiker/shortener/internal/store"
	"github.com/laiker/shortener/internal/store/file"
	"github.com/laiker/shortener/internal/store/memory"
	"github.com/laiker/shortener/internal/store/pg"
	"go.uber.org/zap"
	"net/http"
	"time"
)

func main() {
	config.ParseFlags()
	run()
}

func run() {
	var db *sql.DB
	var cstore store.Store

	r := chi.NewRouter()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := logger.Initialize(config.FlagLogLevel)
	if err != nil {
		fmt.Println(err)
	}

	cstore = memory.NewStore()
	logger.Log.Info("Store Memory")

	if config.StoragePath != "" {
		cstore = file.NewStore(config.StoragePath)
		logger.Log.Info("Store File")

	}

	if config.DatabaseDsn != "" {
		logger.Log.Info("Store postgres")

		db, err = sql.Open("pg", config.DatabaseDsn)

		if err != nil {
			fmt.Println(err)
		}

		if err != nil {
			return
		}

		cstore = pg.NewStore(db)

		defer db.Close()
	}

	err = cstore.Bootstrap(ctx)

	if err != nil {
		return
	}

	appInstance := newApp(cstore)

	r.Use(logger.RequestLogger, appInstance.gzipMiddleware)
	r.HandleFunc("/api/shorten", appInstance.shortenHandler)
	r.HandleFunc("/{id}", appInstance.decodeHandler)
	r.HandleFunc("/ping", appInstance.pingHandler)
	r.HandleFunc("/", appInstance.encodeHandler)

	logger.Log.Info("Server runs at: ", zap.String("address", config.FlagRunAddr))
	http.ListenAndServe(config.FlagRunAddr, r)
}
