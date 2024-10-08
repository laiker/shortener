package config

import (
	"flag"
	"os"
)

type ContextKey string

var FlagRunAddr string
var FlagOutputURL string
var FlagLogLevel string
var StoragePath string
var DatabaseDsn string

const UserIDKey ContextKey = "userID"

func ParseFlags() {
	flag.StringVar(&FlagRunAddr, "a", "localhost:8080", "Initial webserver URL")
	flag.StringVar(&FlagOutputURL, "b", "http://localhost:8080", "Output short url host")
	flag.StringVar(&FlagLogLevel, "l", "info", "log level")
	flag.StringVar(&StoragePath, "file-storage-path", "/tmp/V23vlAC", "File urls storage path")
	flag.StringVar(&DatabaseDsn, "d", "", "Database url")
	flag.Parse()

	if envRunAddr := os.Getenv("SERVER_ADDRESS"); envRunAddr != "" {
		FlagRunAddr = envRunAddr
	}

	if envOutputURL := os.Getenv("BASE_URL"); envOutputURL != "" {
		FlagOutputURL = envOutputURL
	}

	if envStoragePath := os.Getenv("FILE_STORAGE_PATH"); envStoragePath != "" {
		StoragePath = envStoragePath
	}

	if envDatabaseDsn := os.Getenv("DATABASE_DSN"); envDatabaseDsn != "" {
		DatabaseDsn = envDatabaseDsn
	}

}
