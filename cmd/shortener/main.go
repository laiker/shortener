package main

import (
	"context"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/laiker/shortener/cmd/config"
	logger "github.com/laiker/shortener/internal"
	store "github.com/laiker/shortener/internal/store"
	"github.com/laiker/shortener/internal/store/file"
	"github.com/laiker/shortener/internal/store/memory"
	"github.com/laiker/shortener/internal/store/pg"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
	"net/http"
	"time"
)

func main() {
	config.ParseFlags()
	run()
}

func run() {
	var db *pgxpool.Pool
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
		logger.Log.Info("DSN " + config.DatabaseDsn)
		logger.Log.Info("Store postgres")

		db, err = pgxpool.New(context.Background(), config.DatabaseDsn)

		if err != nil {
			logger.Log.Info(err.Error())
			return
		}

		cstore = pg.NewStore(db)

		defer db.Close()
	}

	err = cstore.Bootstrap(ctx)

	if err != nil {
		logger.Log.Info(err.Error())
		return
	}

	appInstance := newApp(cstore)

	r.Use(logger.RequestLogger, appInstance.gzipMiddleware)
	r.HandleFunc("/api/shorten/batch", appInstance.shortenBatchHandler)
	r.HandleFunc("/api/shorten", appInstance.shortenHandler)
	r.HandleFunc("/{id}", appInstance.decodeHandler)
	r.HandleFunc("/ping", appInstance.pingHandler)
	r.HandleFunc("/", appInstance.encodeHandler)

	logger.Log.Info("Server runs at: ", zap.String("address", config.FlagRunAddr))
	http.ListenAndServe(config.FlagRunAddr, r)
}
