package store

import (
	"context"
)

// Store описывает абстрактное хранилище сообщений пользователей
type Store interface {
	SaveUrl(ctx context.Context, short, original string) error
	PingContext(ctx context.Context) error
}
