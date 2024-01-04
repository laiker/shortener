package pg

import (
	"context"
	"errors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	logger "github.com/laiker/shortener/internal"
	"github.com/laiker/shortener/internal/json"
	"github.com/laiker/shortener/internal/store"
)

// Store реализует интерфейс store.Store и позволяет взаимодействовать с СУБД PostgreSQL
type Store struct {
	// Поле conn содержит объект соединения с СУБД
	conn *pgxpool.Pool
}

// NewStore возвращает новый экземпляр PostgreSQL хранилища
func NewStore(conn *pgxpool.Pool) *Store {
	return &Store{conn: conn}
}

func (s *Store) PingContext(ctx context.Context) error {
	return s.conn.Ping(ctx)
}

// Bootstrap подготавливает БД к работе, создавая необходимые таблицы и индексы
func (s *Store) Bootstrap(ctx context.Context) error {
	// запускаем транзакцию
	options := pgx.TxOptions{}
	tx, err := s.conn.BeginTx(ctx, options)
	if err != nil {
		return err
	}

	// в случае неуспешного коммита все изменения транзакции будут отменены
	defer tx.Rollback(ctx)

	// создаём таблицу пользователей и необходимые индексы
	_, err = tx.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS urls (
				id serial PRIMARY KEY,
				original_url varchar NOT NULL,
				short_url varchar NOT NULL
			)
	    `)

	logger.Log.Info("BOOTSTRAP")

	if err != nil {
		return err
	}

	// коммитим транзакцию
	return tx.Commit(ctx)
}

func (s *Store) SaveURL(ctx context.Context, original string, short string) error {

	result := s.conn.QueryRow(ctx, "SELECT COUNT(*) as count FROM urls WHERE original_url LIKE $1", original)

	var countValues int
	err := result.Scan(&countValues)

	if err != nil {
		return err
	}

	if countValues > 0 {
		return store.ErrUnique
	}

	_, errexec := s.conn.Exec(ctx, "INSERT INTO urls(original_url, short_url) VALUES($1, $2)", original, short)

	if errexec != nil {
		var pgErr *pgconn.PgError
		isPgError := errors.As(errexec, &pgErr)
		if isPgError && pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
			return store.ErrUnique
		}
		return errexec
	}

	return nil
}

func (s *Store) SaveBatchURL(ctx context.Context, urls json.BatchURLSlice) error {
	options := pgx.TxOptions{}
	tx, err := s.conn.BeginTx(ctx, options)

	if err != nil {
		return err
	}

	// в случае неуспешного коммита все изменения транзакции будут отменены
	defer tx.Rollback(ctx)

	for i := 0; i < len(urls); i++ {

		err := s.SaveURL(ctx, urls[i].OriginalURL, urls[i].ShortURL)

		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (s *Store) GetURL(ctx context.Context, short string) (json.DBRow, error) {

	URLRow := json.DBRow{}

	row, err := s.conn.Query(ctx, "SELECT id, original_url, short_url FROM urls WHERE short_url = $1", short)

	if err != nil {
		return URLRow, err
	}

	row.Close()

	for row.Next() {
		if err := row.Scan(&URLRow.ID, &URLRow.OriginalURL, &URLRow.ShortURL); err != nil {
			return URLRow, err
		}
	}

	if err := row.Err(); err != nil {
		return URLRow, err
	}

	if URLRow.OriginalURL == "" {
		return URLRow, errors.New("url not found")
	}

	return URLRow, nil
}
