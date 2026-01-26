package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sso/internal/domain/models"
	"sso/internal/storage"
	"time"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

type Storage struct {
	db *sql.DB
}

func New(connStr string) (*Storage, error) {
	const op = "storage.postgres.New"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil

}

func (s *Storage) Stop() error {
	return s.db.Close()
}

func (s *Storage) SaveApp(ctx context.Context, name, secret string) (int64, error) {
	const op = "storage.postgres.SaveApp"

	stmt, err := s.db.Prepare("INSERT INTO apps(name, secret) VALUES($1, $2) RETURNING id")

	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	row := stmt.QueryRowContext(ctx, name, secret)

	var id int64
	err = row.Scan(&id)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrAppExists)
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

func (s *Storage) SaveUser(ctx context.Context, email string, passHash []byte) (int64, error) {
	const op = "storage.postgres.SaveUser"

	stmt, err := s.db.Prepare("INSERT INTO users(email, pass_hash) VALUES($1, $2) RETURNING id")

	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	row := stmt.QueryRowContext(ctx, email, passHash)

	var id int64
	err = row.Scan(&id)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrUserExists)
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

// SaveRefreshToken сохраняет refresh токен
func (s *Storage) SaveRefreshToken(ctx context.Context, token string, userID int64, appID int, expiresAt time.Time) error {
	const op = "storage.postgres.SaveRefreshToken"

	stmt, err := s.db.Prepare(`
        INSERT INTO refresh_tokens(token, user_id, app_id, expires_at) 
        VALUES($1, $2, $3, $4)
    `)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.ExecContext(ctx, token, userID, appID, expiresAt)
	if err != nil {
		// Проверяем, если это нарушение уникальности токена (должно быть очень редким)
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			// Токен уже существует - генерируем новый и пробуем снова
			return fmt.Errorf("%s: %w", op, errors.New("token collision, please retry"))
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// GetRefreshToken получает refresh токен
func (s *Storage) GetRefreshToken(ctx context.Context, token string) (models.RefreshToken, error) {
	const op = "storage.postgres.GetRefreshToken"

	stmt, err := s.db.Prepare(`SELECT token, user_id, app_id, expires_at, created_at FROM refresh_tokens WHERE token = $1`)
	if err != nil {
		return models.RefreshToken{}, fmt.Errorf("%s: %w ", op, err)
	}

	row := stmt.QueryRowContext(ctx, token)
	var rt models.RefreshToken
	err = row.Scan(&rt.Token, &rt.UserID, &rt.AppID, &rt.ExpiresAt, &rt.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.RefreshToken{}, fmt.Errorf("%s: %w ", op, storage.ErrRefreshTokenNotFound)

		}
		return models.RefreshToken{}, fmt.Errorf("%s: %w ", op, err)
	}

	return rt, nil
}

// DeleteRefreshToken удаляет refresh токен
func (s *Storage) DeleteRefreshToken(ctx context.Context, token string) error {
	const op = "storage.postgres.DeleteRefreshToken"

	stmt, err := s.db.Prepare("DELETE FROM refresh_tokens WHERE token = $1")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.ExecContext(ctx, token)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// DeleteAllUserRefreshTokens удаляет все refresh токены пользователя для приложения
func (s *Storage) DeleteAllUserRefreshTokens(ctx context.Context, userID int64, appID int) error {
	const op = "storage.postgres.DeleteAllUserRefreshTokens"

	stmt, err := s.db.Prepare("DELETE FROM refresh_tokens WHERE user_id = $1 AND app_id = $2")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.ExecContext(ctx, userID, appID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// User returns user by email.
func (s *Storage) User(ctx context.Context, email string) (models.User, error) {
	const op = "storage.postgres.User"

	stmt, err := s.db.Prepare("SELECT id, email, pass_hash FROM users WHERE email = $1")
	if err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	row := stmt.QueryRowContext(ctx, email)
	var user models.User
	err = row.Scan(&user.ID, &user.Email, &user.PassHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}

		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return user, nil
}

// App returns app by id.
func (s *Storage) App(ctx context.Context, id int) (models.App, error) {
	const op = "storage.postgres.App"

	stmt, err := s.db.Prepare("SELECT id, name, secret FROM apps WHERE id = $1")
	if err != nil {
		return models.App{}, fmt.Errorf("%s: %w", op, err)
	}

	row := stmt.QueryRowContext(ctx, id)
	var app models.App
	err = row.Scan(&app.ID, &app.Name, &app.Secret)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.App{}, fmt.Errorf("%s: %w", op, storage.ErrAppNotFound)
		}

		return models.App{}, fmt.Errorf("%s: %w", op, err)
	}

	return app, nil
}

// App returns app by id.
func (s *Storage) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	const op = "storage.postgres.IsAdmin"

	stmt, err := s.db.Prepare("SELECT is_admin FROM users WHERE id = $1")
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	row := stmt.QueryRowContext(ctx, userID)

	var isAdmin bool

	err = row.Scan(&isAdmin)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}

		return false, fmt.Errorf("%s: %w", op, err)
	}

	return isAdmin, nil
}
