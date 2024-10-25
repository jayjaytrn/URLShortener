package db

import (
	"context"

	"github.com/jayjaytrn/URLShortener/internal/types"
)

type ShortenerStorage interface {
	// GetOriginal возвращает оригинальный URL по короткому URL
	GetOriginal(shortURL string) (string, error)
	// Put добавляет новую запись в БД, возвращает true если запись была добавлена
	Put(urlData types.URLData) error
	// Exists возвращает true если запись найдена
	Exists(url string) (bool, error)
	// PutBatch добавляет пачку новых записей в БД, если одна из записей не удалась, не записывается вся пачка
	PutBatch(ctx context.Context, batchData []types.URLData) error
	// Close закрывает соединение с базой
	Close(ctx context.Context) error
	// Ping проверяет доступность хранилища
	Ping(ctx context.Context) error
	// GenerateNewUserID возвращает новый ID для пользователя
	GenerateNewUserID() string
	// GetURLsByUserID возвращает все url для userID
	GetURLsByUserID(userID string) ([]types.URLData, error)
}
