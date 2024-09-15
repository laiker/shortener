package store

import (
	"context"
	"errors"
	"github.com/laiker/shortener/internal/json"
)

var ErrUnique = errors.New("original url is already created")

// Store описывает абстрактное хранилище сообщений пользователей
type Store interface {
	SaveURL(ctx context.Context, short, original string) error
	SaveBatchURL(ctx context.Context, rows json.BatchURLSlice) error
	PingContext(ctx context.Context) error
	Bootstrap(ctx context.Context) error
	GetURL(ctx context.Context, short string) (json.DBRow, error)
	GetUserURLs(ctx context.Context, userID string) ([]json.DBRow, error)
}
