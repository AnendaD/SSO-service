package postgres

import (
	"context"
	"errors"
	"fmt"
	"sso/internal/domain/models"
	"sso/internal/storage"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewPool(ctx context.Context, databaseURL string, maxConns int, minConns int) (*pgxpool.Pool, error) {
	const op = "storage.postgres.NewPool"

	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	cfg.MinConns = int32(minConns)
	cfg.MaxConns = int32(maxConns)

	return pgxpool.NewWithConfig(ctx, cfg)
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool}
}

func (r *Repository) SaveApp(ctx context.Context, name, secret string) (int64, error) {
	const op = "storage.postgres.SaveApp"
	var id int64
	err := r.pool.QueryRow(ctx, `
        INSERT INTO apps (name, secret)
        VALUES ($1, $2)
        RETURNING id
    `, name, secret).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

func (r *Repository) SaveUser(ctx context.Context, email string, passHash []byte) (int64, error) {
	const op = "storage.postgres.SaveUser"

	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users(email, pass_hash) 
		VALUES($1, $2) 
		ON CONFLICT (email) DO NOTHING
		RETURNING id
	`, email, passHash).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

// SaveRefreshToken сохраняет refresh токен
func (r *Repository) SaveRefreshToken(ctx context.Context, token string, userID int64, appID int, expiresAt time.Time) error {
	const op = "storage.postgres.SaveRefreshToken"

	tag, err := r.pool.Exec(ctx, `
        INSERT INTO refresh_tokens(token, user_id, app_id, expires_at, created_at) 
        VALUES($1, $2, $3, $4, now())
		ON CONFLICT (token) DO NOTHING
    `, token, userID, appID, expiresAt)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if tag.RowsAffected() != 1 {
		return fmt.Errorf("%s: %w", op, storage.ErrInvalidRefreshToken)
	}

	return nil
}

// GetRefreshToken получает refresh токен
func (r *Repository) GetRefreshToken(ctx context.Context, token string) (models.RefreshToken, error) {
	const op = "storage.postgres.GetRefreshToken"

	var tkn models.RefreshToken
	err := r.pool.QueryRow(ctx, `SELECT token, user_id, app_id, expires_at, created_at FROM refresh_tokens WHERE token = $1`, token).Scan(&tkn.Token, &tkn.UserID, &tkn.AppID, &tkn.ExpiresAt, &tkn.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.RefreshToken{}, fmt.Errorf("%s: %w", op, storage.ErrRefreshTokenNotFound)
		}
		return models.RefreshToken{}, fmt.Errorf("%s: %w", op, err)
	}

	return tkn, nil
}

// DeleteRefreshToken удаляет refresh токен
func (r *Repository) DeleteRefreshToken(ctx context.Context, token string) error {
	const op = "storage.postgres.DeleteRefreshToken"

	tag, err := r.pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE token = $1`, token)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if tag.RowsAffected() != 1 {
		return fmt.Errorf("%s: %w", op, storage.ErrRefreshTokenNotFound)
	}

	return nil
}

// DeleteAllUserRefreshTokens удаляет все refresh токены пользователя для приложения
func (r *Repository) DeleteAllUserRefreshTokens(ctx context.Context, userID int64, appID int) error {
	const op = "storage.postgres.DeleteAllUserRefreshTokens"

	tag, err := r.pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_id = $1 AND app_id = $2`, userID, appID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, storage.ErrRefreshTokenNotFound)
	}

	return nil
}

// User returns user by email.
func (r *Repository) User(ctx context.Context, email string) (models.User, error) {
	const op = "storage.postgres.User"

	var user models.User
	err := r.pool.QueryRow(ctx, `SELECT id, email, pass_hash FROM users WHERE email = $1`, email).Scan(&user.ID, &user.Email, &user.PassHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return user, nil
}

// App returns app by id.
func (r *Repository) App(ctx context.Context, id int) (models.App, error) {
	const op = "storage.postgres.App"

	var app models.App
	err := r.pool.QueryRow(ctx, `SELECT id, name, secret FROM apps WHERE id = $1`, id).Scan(&app.ID, &app.Name, &app.Secret)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.App{}, fmt.Errorf("%s: %w", op, storage.ErrAppNotFound)
		}
		return models.App{}, fmt.Errorf("%s: %w", op, err)
	}

	return app, nil
}

func (r *Repository) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	const op = "storage.postgres.IsAdmin"

	var isAdmin bool
	err := r.pool.QueryRow(ctx, `SELECT is_admin FROM users WHERE id = $1`, userID).Scan(&isAdmin)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return isAdmin, nil
}

// UserByID returns user by ID
func (r *Repository) UserByID(ctx context.Context, userID int64) (models.User, error) {
	const op = "storage.postgres.UserByID"

	var user models.User
	err := r.pool.QueryRow(ctx, `SELECT id, email, pass_hash FROM users WHERE id = $1`, userID).Scan(&user.ID, &user.Email, &user.PassHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return user, nil
}
