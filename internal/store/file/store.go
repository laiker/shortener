package file

import (
	"bufio"
	"context"
	json2 "encoding/json"
	"errors"
	"github.com/laiker/shortener/cmd/config"
	"github.com/laiker/shortener/internal/json"
	"os"
	"strings"
)

type Store struct {
	filename string
	file     *os.File
}

func NewStore(filename string) *Store {
	return &Store{
		filename: filename,
		file:     nil,
	}
}

func (s *Store) PingContext(ctx context.Context) error {
	_, err := os.Stat(s.filename)

	if os.IsNotExist(err) {
		return errors.New("file is not exists")
	}

	return nil
}

func (s *Store) Bootstrap(ctx context.Context) error {
	file, err := os.OpenFile(config.StoragePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0775)

	if err != nil {
		return err
	}

	s.file = file

	return nil
}

func (s *Store) SaveURL(ctx context.Context, original, short string) error {

	encoder := json2.NewEncoder(s.file)
	reader := bufio.NewReader(s.file)

	var lastLine string

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break // Достигнут конец файла
		}
		lastLine = line
		if strings.TrimSpace(line) == "}" {
			break
		}
	}

	lastRow := &json.DBRow{}
	lastID := 1
	if lastLine != "" {
		if err := json2.Unmarshal([]byte(lastLine), &lastRow); err != nil {
			return err
		}

		lastID = lastRow.ID + 1
	}

	row := &json.DBRow{
		ID:          lastID,
		OriginalURL: original,
		ShortURL:    short,
	}

	err := encoder.Encode(row)

	if err != nil {
		return err
	}

	return nil
}

func (s *Store) GetURL(ctx context.Context, short string) (json.DBRow, error) {
	reader := bufio.NewReader(s.file)

	row := json.DBRow{}

	for {
		line, err := reader.ReadString('\n')
		switch err {
		case nil:
			if err := json2.Unmarshal([]byte(line), &row); err != nil {
				return row, err // Ошибка декодирования строки в JSON
			}

			// Если найдено совпадение, возвращаем OriginalURL
			if row.ShortURL == strings.TrimSpace(short) {
				return row, nil
			}
		default:
			return row, errors.New("url not found") // Другая ошибка чтения файла
		}
	}
}
