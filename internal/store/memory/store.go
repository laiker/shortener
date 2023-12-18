package memory

import (
	"context"
	"errors"
	"github.com/laiker/shortener/internal/json"
)

type data map[string]json.DBRow

type Store struct {
	data
}

// NewStore возвращает новый экземпляр PostgreSQL хранилища
func NewStore() *Store {
	return &Store{
		data: make(data, 0),
	}
}

func (s *Store) PingContext(ctx context.Context) error {
	return nil
}

func (s *Store) Bootstrap(ctx context.Context) error {
	s.data = make(data, 0)
	return nil
}

func (s *Store) SaveURL(ctx context.Context, original, short string) error {

	url := json.DBRow{
		ID:          len(s.data) + 1,
		OriginalURL: original,
		ShortURL:    short,
	}

	s.data[short] = url

	return nil
}

func (s *Store) SaveBatchURL(ctx context.Context, urls json.BatchURLSlice) error {
	for i := 0; i < len(urls); i++ {
		err := s.SaveURL(ctx, urls[i].ShortURL, urls[i].OriginalURL)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) GetURL(ctx context.Context, short string) (json.DBRow, error) {

	dbRow := s.data[short]

	if len(dbRow.OriginalURL) <= 0 {
		return dbRow, errors.New("url not fund")
	}

	return dbRow, nil
}
