package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jayjaytrn/URLShortener/config"
	"github.com/jayjaytrn/URLShortener/internal/types"
	"github.com/lib/pq"
)

type OriginalExistError struct {
	ShortURL string
}

func (e *OriginalExistError) Error() string {
	return fmt.Sprintf("original URL already exists, short URL for it is: %s", e.ShortURL)
}

type Manager struct {
	db  *sql.DB
	cfg *config.Config
}

// NewManager создает новый экземпляр Manager и устанавливает подключение к БД
func NewManager(cfg *config.Config) (*Manager, error) {
	db, err := sql.Open("pgx", cfg.DatabaseDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	manager := &Manager{
		db:  db,
		cfg: cfg,
	}

	if err := manager.createShortenerTable(); err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return manager, nil
}

func (m *Manager) GetOriginal(shortURL string) (string, error) {
	var originalURL string
	var isDeleted bool
	err := m.db.QueryRow("SELECT original_url, is_deleted FROM shortener WHERE short_url = $1", shortURL).Scan(&originalURL, &isDeleted)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("URL not found")
		}
		return "", fmt.Errorf("failed to get original URL: %w", err)
	}
	if isDeleted {
		return "", fmt.Errorf("URL has been deleted")
	}
	return originalURL, nil
}

// GetURLsByUserID возвращает все URL, сокращённые пользователем
func (m *Manager) GetURLsByUserID(userID string) ([]types.URLData, error) {
	rows, err := m.db.Query("SELECT short_url, original_url FROM shortener WHERE user_id = $1", userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get URLs for user: %w", err)
	}
	defer rows.Close()

	var urls []types.URLData
	for rows.Next() {
		var shortURL, originalURL string
		if err := rows.Scan(&shortURL, &originalURL); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		urls = append(urls, types.URLData{
			ShortURL:    m.cfg.BaseURL + "/" + shortURL,
			OriginalURL: originalURL,
		})
	}

	if len(urls) == 0 {
		return nil, fmt.Errorf("no URLs found for userID: %s", userID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return urls, nil
}

// Put добавляет новую запись в базу данных
func (m *Manager) Put(urlData types.URLData) error {
	var alreadyExistedShortURL string

	err := m.db.QueryRow(`
        WITH ins AS (
            INSERT INTO shortener (short_url, original_url, user_id)
            VALUES ($1, $2, $3)
            ON CONFLICT (original_url) DO NOTHING
        )
        SELECT short_url FROM shortener WHERE original_url = $2;
    `, urlData.ShortURL, urlData.OriginalURL, urlData.UserID).Scan(&alreadyExistedShortURL)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("failed to insert URL: %w", err)
		}
	}

	if alreadyExistedShortURL != "" {
		return &OriginalExistError{ShortURL: alreadyExistedShortURL}
	}

	return nil
}

func (m *Manager) PutBatch(ctx context.Context, batchData []types.URLData) error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	for _, b := range batchData {
		// все изменения записываются в транзакцию
		_, err = tx.ExecContext(ctx,
			"INSERT INTO shortener (short_url, original_url, user_id) VALUES ($1, $2, $3)",
			b.ShortURL, b.OriginalURL, b.UserID)
		if err != nil {
			// если ошибка, то откатываем изменения
			tx.Rollback()
			return err
		}
	}
	// завершаем транзакцию
	return tx.Commit()
}

// Exists проверяет, существует ли короткий URL в базе данных
func (m *Manager) Exists(shortURL string) (bool, error) {
	var exists bool
	if err := m.db.QueryRow("SELECT EXISTS(SELECT 1 FROM shortener WHERE short_url = $1)", shortURL).Scan(&exists); err != nil {
		return false, fmt.Errorf("failed to check if URL exists: %w", err)
	}
	return exists, nil
}

// GenerateNewUserID генерирует новый уникальный идентификатор пользователя
func (m *Manager) GenerateNewUserID() string {
	return uuid.New().String()
}

func (m *Manager) BatchDelete(urlChannel chan string, userID string) {
	const batchSize = 10
	var urlsBatch []string

	for urlID := range urlChannel {
		urlsBatch = append(urlsBatch, urlID)

		// Когда набирается batchSize элементов, вызываем updateBatch
		if len(urlsBatch) == batchSize {
			m.updateBatch(urlsBatch, userID)
			urlsBatch = urlsBatch[:0] // очищаем батч после обновления
		}
	}

	// Обрабатываем оставшиеся элементы, если они остались
	if len(urlsBatch) > 0 {
		m.updateBatch(urlsBatch, userID)
	}
}

func (m *Manager) updateBatch(urlsBatch []string, userID string) error {
	query := "UPDATE shortener SET is_deleted = TRUE WHERE short_url = ANY($1) AND user_id = $2"
	_, err := m.db.Exec(query, pq.Array(urlsBatch), userID)
	if err != nil {
		return fmt.Errorf("failed to batch delete URLs: %w", err)
	}
	return nil
}

func (m *Manager) Ping(ctx context.Context) error {
	return m.db.PingContext(ctx)
}

// Close закрывает соединение с базой данных
func (m *Manager) Close(ctx context.Context) error {
	return m.db.Close()
}

// createTable создаёт таблицу shortener, если она не существует
func (m *Manager) createShortenerTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS shortener (
		user_id VARCHAR(255) NOT NULL,
		short_url VARCHAR(255) NOT NULL UNIQUE,
		original_url TEXT NOT NULL UNIQUE,
	    is_deleted BOOLEAN DEFAULT FALSE
	);`

	_, err := m.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}
	return nil
}
