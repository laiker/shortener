package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/laiker/shortener/internal/json"
	"time"
)

// Store реализует интерфейс store.Store и позволяет взаимодействовать с СУБД PostgreSQL
type Store struct {
	// Поле conn содержит объект соединения с СУБД
	conn *sql.DB
}

// NewStore возвращает новый экземпляр PostgreSQL хранилища
func NewStore(conn *sql.DB) *Store {
	return &Store{conn: conn}
}

func (s Store) PingContext(ctx context.Context) error {
	return s.conn.PingContext(ctx)
}

// Bootstrap подготавливает БД к работе, создавая необходимые таблицы и индексы
func (s Store) Bootstrap(ctx context.Context) error {
	// запускаем транзакцию
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// в случае неуспешного коммита все изменения транзакции будут отменены
	defer tx.Rollback()

	// создаём таблицу пользователей и необходимые индексы
	tx.ExecContext(ctx, `
        CREATE TABLE users (
            id int4 NOT NULL PRIMARY KEY,
			original_url varchar NOT NULL,
			short_url varchar NOT NULL
        )
    `)

	// коммитим транзакцию
	return tx.Commit()
}

func (s Store) SaveUrl(ctx context.Context, original string, short string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	result, errexec := s.conn.ExecContext(ctx, "INSERT INTO urls (original_url, short_url) VALUES ($1, $2)", original, short)

	if errexec != nil {
		return errexec
	}

	rows, err := result.RowsAffected()

	if err != nil {
		return err
	}

	if rows != 1 {
		return fmt.Errorf("expected to affect 1 row, affected %d", rows)
	}

	return nil
}

func (s Store) GetUrl(ctx context.Context, short string) (json.DBRow, error) {

	URLRow := json.DBRow{}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	row, err := s.conn.QueryContext(ctx, "SELECT id, original_url, short_url FROM urls WHERE short_url = $1", short)

	if err != nil {
		return URLRow, err
	}

	row.Close()

	if err != nil {
		return URLRow, err
	}

	for row.Next() {
		if err := row.Scan(&URLRow.ID, &URLRow.OriginalURL, &URLRow.ShortURL); err != nil {
			return URLRow, err
		}
	}

	if URLRow.OriginalURL == "" {
		return URLRow, errors.New("url not found")
	}

	return URLRow, nil
}
