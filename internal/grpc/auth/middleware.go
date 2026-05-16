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
		publicMethods := map[string]bool{
			"/auth.Auth/Login":     true,
			"/auth.Auth/Register":  true,
			"/auth.Auth/Refresh":   true,
			"/auth.Auth/Logout":    true,
			"/auth.Auth/LogoutAll": true,
		}

		log.Printf("request to: %s", info.FullMethod)

		if publicMethods[info.FullMethod] {
			log.Printf("public method, skipping auth")
			return handler(ctx, req)
		}

		log.Printf("protected method, checking auth")

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			log.Printf("no metadata found")
			return nil, status.Error(codes.Unauthenticated, "authorization token not provided")
		}

		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			log.Printf("no authorization header")
			return nil, status.Error(codes.Unauthenticated, "authorization token not provided")
		}

		log.Printf("auth header: %s", authHeaders[0])

		token := strings.TrimPrefix(authHeaders[0], "Bearer ")
		if token == "" || token == authHeaders[0] {
			log.Printf("invalid Bearer format")
			return nil, status.Error(codes.Unauthenticated, "invalid authorization header format")
		}

		log.Printf("validating token...")

		userID, err := authService.ValidateToken(ctx, token)
		if err != nil {
			log.Printf("token validation failed: %v", err)
			if errors.Is(err, jwt.ErrExpiredToken) {
				return nil, status.Error(codes.Unauthenticated, "token expired")
			}
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		log.Printf("token valid for user: %d", userID)

		ctx = context.WithValue(ctx, "userID", userID)

		return handler(ctx, req)
	}
}
