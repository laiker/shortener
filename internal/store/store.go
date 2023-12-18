package store

import (
	"context"
	"github.com/laiker/shortener/internal/json"
)

// Store описывает абстрактное хранилище сообщений пользователей
type Store interface {
	SaveURL(ctx context.Context, short, original string) error
	SaveBatchURL(ctx context.Context, rows json.BatchURLSlice) error
	PingContext(ctx context.Context) error
	Bootstrap(ctx context.Context) error
	GetURL(ctx context.Context, short string) (json.DBRow, error)
}
