package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jayjaytrn/URLShortener/config"
	"github.com/jayjaytrn/URLShortener/internal/types"
)

type Manager struct {
	db *sql.DB
}

// NewManager создает новый экземпляр Manager и устанавливает подключение к БД
func NewManager(cfg *config.Config) (*Manager, error) {
	// Устанавливаем подключение к базе данных
	db, err := sql.Open("pgx", cfg.DatabaseDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Проверяем соединение
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Manager{
		db: db,
	}, nil
}

func (m *Manager) GetOriginal(shortURL string) (string, error) {
	var originalURL string
	err := m.db.QueryRow("SELECT original_url FROM urls WHERE short_url = $1", shortURL).Scan(&originalURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("URL not found")
		}
		return "", fmt.Errorf("failed to get original URL: %w", err)
	}
	return originalURL, nil
}

// Put добавляет новую запись в базу данных
func (m *Manager) Put(urlData types.URLData) error {
	_, err := m.db.Exec("INSERT INTO urls (uuid, short_url, original_url) VALUES ($1, $2, $3)",
		urlData.UUID, urlData.ShortURL, urlData.OriginalURL)
	if err != nil {
		return fmt.Errorf("failed to insert URL: %w", err)
	}
	return nil
}

// Exists проверяет, существует ли короткий URL в базе данных
func (m *Manager) Exists(shortURL string) (bool, error) {
	var exists bool
	if err := m.db.QueryRow("SELECT EXISTS(SELECT 1 FROM urls WHERE short_url = $1)", shortURL).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check if URL exists: %w", err)
	}

	return true, nil
}

func (m *Manager) GetNextUUID() (string, error) {
	var count int
	err := m.db.QueryRow("SELECT COUNT(*) FROM urls").Scan(&count)
	if err != nil {
		return "", fmt.Errorf("failed to get next UUID: %w", err)
	}

	// UUID — это текущее количество записей, увеличенное на 1
	nextUUID := strconv.Itoa(count)
	return nextUUID, nil
}

func (m *Manager) Ping(ctx context.Context) error {
	return m.db.PingContext(ctx)
}

// Close закрывает соединение с базой данных
func (m *Manager) Close(ctx context.Context) error {
	return m.db.Close()
}
