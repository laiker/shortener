package store

import (
	"context"
	"github.com/laiker/shortener/internal/json"
)

// Store описывает абстрактное хранилище сообщений пользователей
type Store interface {
	SaveUrl(ctx context.Context, short, original string) error
	PingContext(ctx context.Context) error
	Bootstrap(ctx context.Context) error
	GetUrl(ctx context.Context, short string) (json.DBRow, error)
}
