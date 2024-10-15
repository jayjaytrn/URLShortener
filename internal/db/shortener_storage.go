package db

import (
	"context"

	"github.com/jayjaytrn/URLShortener/internal/types"
)

type ShortenerStorage interface {
	// GetOriginal возвращает оригинальный URL по короткому URL
	GetOriginal(shortURL string) (string, error)
	// Put добавляет новую запись в БД
	Put(urlData types.URLData) error
	// Exists возвращает true если запись найдена
	Exists(url string) (bool, error)
	// GetNextUUID возвращает следующий доступный UUID для новой записи
	GetNextUUID() (string, error)
	// Close закрывает соединение с базой
	Close(ctx context.Context) error
}
