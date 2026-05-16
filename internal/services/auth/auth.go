package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sso/internal/domain/models"
	"sso/internal/lib/jwt"
	refreshtoken "sso/internal/lib/refresh_token"
	"sso/internal/storage"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Auth struct {
	log             *slog.Logger
	usrSaver        UserSaver
	usrProvider     UserProvider
	appProvider     AppProvider
	tokenTTL        time.Duration
	refreshTokenTTL time.Duration
	tokenSaver      RefreshTokenSaver
	tokenProvider   RefreshTokenProvider
}

type UserSaver interface {
	SaveUser(
		ctx context.Context,
		email string,
		passHash []byte,
	) (uid int64, err error)
}

type UserProvider interface {
	User(ctx context.Context, email string) (models.User, error)
	UserByID(ctx context.Context, userID int64) (models.User, error)
	IsAdmin(ctx context.Context, userID int64) (bool, error)
}

type AppProvider interface {
	App(ctx context.Context, appID int) (models.App, error)
}

type RefreshTokenSaver interface {
	SaveRefreshToken(
		ctx context.Context,
		token string,
		userID int64,
		appID int,
		expiresAt time.Time,
	) error
}

type RefreshTokenProvider interface {
	GetRefreshToken(ctx context.Context, token string) (models.RefreshToken, error)
	DeleteRefreshToken(ctx context.Context, token string) error
	DeleteAllUserRefreshTokens(ctx context.Context, userID int64, appID int) error
}

var (
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrInvalidAppId        = errors.New("invalid app id")
	ErrUserExists          = errors.New("user already exists")
	ErrUserNotFound        = errors.New("user not found")
	ErrRefreshTokenExpired = errors.New("refresh token expired")
	ErrRefreshTokenInvalid = errors.New("refresh token invalid")
	ErrAccessTokenNotFound = errors.New("access token not found")
)

// New returns a new instance of the Auth service
func New(
	log *slog.Logger,
	userSaver UserSaver,
	userProvider UserProvider,
	appProvider AppProvider,
	tokenSaver RefreshTokenSaver,
	tokenProvider RefreshTokenProvider,
	tokenTTL time.Duration,
	refreshTokenTTL time.Duration,
) *Auth {
	return &Auth{
		usrSaver:        userSaver,
		usrProvider:     userProvider,
		log:             log,
		appProvider:     appProvider,
		tokenSaver:      tokenSaver,
		tokenProvider:   tokenProvider,
		tokenTTL:        tokenTTL,
		refreshTokenTTL: refreshTokenTTL,
	}
}

func (a *Auth) Login(
	ctx context.Context,
	email string,
	password string,
	appID int,
) (string, string, error) {
	const op = "Auth.Login"

	log := a.log.With(
		slog.String("op", op),
		slog.String("username", email),
	)

	log.Info("attempting to login user")

	user, err := a.usrProvider.User(ctx, email)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			a.log.Warn("user not found", "error", err)

			return "", "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
		}

		a.log.Error("failed to get user", "error", err)

		return "", "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}

	if err := bcrypt.CompareHashAndPassword(user.PassHash, []byte(password)); err != nil {
		a.log.Info("invalid credentials", "error", err)

		return "", "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}
	app, err := a.appProvider.App(ctx, appID)
	if err != nil {
		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	accesstoken, err := jwt.NewToken(user, app, a.tokenTTL)
	if err != nil {
		a.log.Error("failed to generate access token", "error", err)

		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	// Генерация refresh токена
	refreshToken, err := refreshtoken.NewRefreshToken()
	if err != nil {
		a.log.Error("failed to generate refresh token", "error", err)
		return "", "", fmt.Errorf("%s: %w ", op, err)
	}

	// Сохранение refresh токена
	expiresAt := time.Now().Add(a.refreshTokenTTL)
	if err := a.tokenSaver.SaveRefreshToken(ctx, refreshToken, user.ID, appID, expiresAt); err != nil {
		a.log.Error("failed to save refresh token", "error", err)
		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	log.Info("user logged in successfully")

	return accesstoken, refreshToken, nil
}

func (a *Auth) RefreshTokens(
	ctx context.Context,
	refreshToken string,
) (string, string, error) {
	const op = "auth.RefreshTokens"

	log := a.log.With(slog.String("op", op))

	// Получение refresh токена из бд
	rt, err := a.tokenProvider.GetRefreshToken(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, storage.ErrRefreshTokenNotFound) {
			return "", "", fmt.Errorf("%s: %w", op, ErrRefreshTokenInvalid)
		}
		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	// Проверка  срока действия
	if time.Now().After(rt.ExpiresAt) {
		// Удаляем просроченный токен
		_ = a.tokenProvider.DeleteRefreshToken(ctx, refreshToken)
		return "", "", fmt.Errorf("%s: %w", op, ErrRefreshTokenExpired)
	}

	// Получение пользователя и приложения
	user, err := a.usrProvider.UserByID(ctx, rt.UserID)
	if err != nil {
		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	app, err := a.appProvider.App(ctx, rt.AppID)
	if err != nil {
		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	// Генерация нового access токена
	newAccessToken, err := jwt.NewToken(user, app, a.tokenTTL)
	if err != nil {
		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	// Генерация нового refresh токена
	newRefreshToken, err := refreshtoken.NewRefreshToken()
	if err != nil {
		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	// Удаление старого refresh токена
	if err := a.tokenProvider.DeleteRefreshToken(ctx, refreshToken); err != nil {
		log.Warn("failed to delete old refresh token", "error", err)
	}

	// Сохранение нового refresh токена
	expiresAt := time.Now().Add(a.refreshTokenTTL)
	if err := a.tokenSaver.SaveRefreshToken(ctx, newRefreshToken, user.ID, app.ID, expiresAt); err != nil {
		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	log.Info("token refreshed successfully")

	return newAccessToken, newRefreshToken, nil
}

func (a *Auth) ValidateToken(ctx context.Context, tokenString string) (int64, error) {
	const op = "auth.ValidateToken"

	claims, err := jwt.ParseTokenWithoutValidation(tokenString)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, jwt.ErrInvalidToken)
	}

	app, err := a.appProvider.App(ctx, claims.AppID)
	if err != nil {
		if errors.Is(err, storage.ErrAppNotFound) {
			return 0, fmt.Errorf("%s: %w", op, jwt.ErrInvalidToken)
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	validatedClaims, err := jwt.ValidateToken(tokenString, app.Secret)
	if err != nil {
		if errors.Is(err, jwt.ErrExpiredToken) {
			// Токен истек, но мы можем использовать claims для refresh
			return claims.UserID, jwt.ErrExpiredToken
		}
		return 0, fmt.Errorf("%s: %w", op, jwt.ErrInvalidToken)
	}

	return validatedClaims.UserID, nil
}

func (a *Auth) Logout(
	ctx context.Context,
	refreshToken string,
) error {
	const op = "auth.Logout"

	// Просто удаляем refresh токен
	if err := a.tokenProvider.DeleteRefreshToken(ctx, refreshToken); err != nil {
		if errors.Is(err, storage.ErrRefreshTokenNotFound) {
			return nil // Уже не существует
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (a *Auth) LogoutAll(
	ctx context.Context,
	userID int64,
	appID int,
) error {
	const op = "auth.LogoutAll"

	if err := a.tokenProvider.DeleteAllUserRefreshTokens(ctx, userID, appID); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil

}

func (a *Auth) RegisterNewUser(
	ctx context.Context,
	email string,
	pass string,
) (int64, error) {
	const op = "Auth.RegisterNewUser"

	log := a.log.With(
		slog.String("op", op),
		slog.String("email", email),
	)

	log.Info("registering user")

	passHash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to generate password hash", "error", err)

		return 0, fmt.Errorf("%s: %w", op, err)
	}
	id, err := a.usrSaver.SaveUser(ctx, email, passHash)
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			log.Warn("user already exists", "error", err)

			return 0, fmt.Errorf("%s: %w", op, ErrUserExists)
		}
		log.Error("failed to save user", "error", err)

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("user registered")

	return id, nil
}

// IsAdmin checks if user is admin.
func (a *Auth) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	const op = "Auth.IsAdmin"

	log := a.log.With(
		slog.String("op", op),
		slog.Int64("userID", userID),
	)

	log.Info("checking if user is admin")

	isAdmin, err := a.usrProvider.IsAdmin(ctx, userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			log.Warn("user not found", "error", err)

			return false, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}
		log.Error("failed to check if user is admin")
		return false, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("checked if user is admin", slog.Bool("is_admin", isAdmin))

	return isAdmin, nil
}

func (a *Auth) IsAdminByToken(ctx context.Context, tokenString string) (bool, error) {
	const op = "auth.IsAdminByToken"

	userID, err := a.ValidateToken(ctx, tokenString)
	if err != nil {
		// Если токен истек, но мы все равно хотим проверить администратора
		if errors.Is(err, jwt.ErrExpiredToken) {
			// Извлекаем claims из истекшего токена
			claims, parseErr := jwt.ParseTokenWithoutValidation(tokenString)
			if parseErr != nil {
				return false, fmt.Errorf("%s: %w", op, jwt.ErrInvalidToken)
			}
			userID = claims.UserID
		} else {
			return false, fmt.Errorf("%s: %w", op, err)
		}
	}

	return a.IsAdmin(ctx, userID)
}
