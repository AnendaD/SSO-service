package auth

import (
	"context"
	"errors"
	"log"
	"sso/internal/lib/jwt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func AuthInterceptor(authService interface {
	ValidateToken(ctx context.Context, token string) (int64, error)
}) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Публичные методы, не требующие авторизации
		publicMethods := map[string]bool{
			"/auth.Auth/Login":     true,
			"/auth.Auth/Register":  true,
			"/auth.Auth/Refresh":   true,
			"/auth.Auth/Logout":    true,
			"/auth.Auth/LogoutAll": true,
		}

		// Логируем каждый запрос для отладки
		log.Printf("[AUTH MIDDLEWARE] Request to: %s", info.FullMethod)

		// Если это публичный метод, пропускаем проверку
		if publicMethods[info.FullMethod] {
			log.Printf("[AUTH MIDDLEWARE] Public method, skipping auth")
			return handler(ctx, req)
		}

		log.Printf("[AUTH MIDDLEWARE] Protected method, checking auth")

		// Получаем метаданные из контекста
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			log.Printf("[AUTH MIDDLEWARE] No metadata found")
			return nil, status.Error(codes.Unauthenticated, "authorization token not provided")
		}

		// Получаем заголовок авторизации
		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			log.Printf("[AUTH MIDDLEWARE] No authorization header")
			return nil, status.Error(codes.Unauthenticated, "authorization token not provided")
		}

		log.Printf("[AUTH MIDDLEWARE] Auth header: %s", authHeaders[0])

		// Извлекаем токен из заголовка
		token := strings.TrimPrefix(authHeaders[0], "Bearer ")
		if token == "" || token == authHeaders[0] {
			log.Printf("[AUTH MIDDLEWARE] Invalid Bearer format")
			return nil, status.Error(codes.Unauthenticated, "invalid authorization header format")
		}

		log.Printf("[AUTH MIDDLEWARE] Validating token...")

		// Проверяем токен
		userID, err := authService.ValidateToken(ctx, token)
		if err != nil {
			log.Printf("[AUTH MIDDLEWARE] Token validation failed: %v", err)
			if errors.Is(err, jwt.ErrExpiredToken) {
				return nil, status.Error(codes.Unauthenticated, "token expired")
			}
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		log.Printf("[AUTH MIDDLEWARE] Token valid for user: %d", userID)

		// Добавляем userID в контекст
		ctx = context.WithValue(ctx, "userID", userID)

		return handler(ctx, req)
	}
}
