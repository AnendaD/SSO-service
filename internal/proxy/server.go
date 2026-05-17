// proxy/server.go
package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	ssov1 "github.com/AnendaD/protos/gen/go/sso"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type ProxyServer struct {
	grpcAddr string
	client   ssov1.AuthClient
	conn     *grpc.ClientConn
	log      *slog.Logger
}

func NewProxyServer(grpcAddr string, log *slog.Logger) (*ProxyServer, error) {
	conn, err := grpc.NewClient(grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	}

	return &ProxyServer{
		grpcAddr: grpcAddr,
		client:   ssov1.NewAuthClient(conn),
		conn:     conn,
		log:      log,
	}, nil
}

func (s *ProxyServer) Close() error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

func (s *ProxyServer) Start(port string, log *slog.Logger) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/api/login", corsMiddleware(s.handleLogin))
	mux.HandleFunc("/api/register", corsMiddleware(s.handleRegister))
	mux.HandleFunc("/api/refresh", corsMiddleware(s.handleRefresh))
	mux.HandleFunc("/api/logout", corsMiddleware(s.handleLogout))
	mux.HandleFunc("/api/logoutAll", corsMiddleware(s.handleLogoutAll))
	mux.HandleFunc("/api/isAdmin", corsMiddleware(s.handleIsAdmin))

	s.log.Info("HTTP прокси запущен",
		slog.String("port", port),
		slog.String("grpc_addr", s.grpcAddr),
	)
	return http.ListenAndServe(":"+port, mux)
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func (s *ProxyServer) getGRPCClient() ssov1.AuthClient {
	return s.client
}

func (s *ProxyServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		AppID    int32  `json:"appId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	client := s.getGRPCClient()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Login(ctx, &ssov1.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
		AppId:    req.AppID,
	})
	if err != nil {
		s.handleGRPCError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":        resp.GetToken(),
		"refreshToken": resp.GetRefreshToken(),
	})
}

func (s *ProxyServer) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	client := s.getGRPCClient()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Register(ctx, &ssov1.RegisterRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		s.handleGRPCError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"userId": resp.GetUserId(),
	})
}

func (s *ProxyServer) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		RefreshToken string `json:"refreshToken"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	client := s.getGRPCClient()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Refresh(ctx, &ssov1.RefreshRequest{
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		s.handleGRPCError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":        resp.GetToken(),
		"refreshToken": resp.GetRefreshToken(),
	})
}

func (s *ProxyServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		RefreshToken string `json:"refreshToken"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	client := s.getGRPCClient()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Logout(ctx, &ssov1.LogoutRequest{
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		s.handleGRPCError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": resp.GetSuccess(),
	})
}

func (s *ProxyServer) handleLogoutAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID int64 `json:"userId"`
		AppID  int32 `json:"appId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	authHeader := r.Header.Get("Authorization")
	client := s.getGRPCClient()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if authHeader != "" {
		authHeader = strings.TrimSpace(authHeader)
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", authHeader)
	}

	resp, err := client.LogoutAll(ctx, &ssov1.LogoutAllRequest{
		UserId: req.UserID,
		AppId:  req.AppID,
	})
	if err != nil {
		s.handleGRPCError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": resp.GetSuccess(),
	})
}

func (s *ProxyServer) handleIsAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var reqBody struct {
		UserID int64 `json:"user_id"`
	}

	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &reqBody); err != nil {
			reqBody.UserID = 0
		}
	}

	authHeader := r.Header.Get("Authorization")
	client := s.getGRPCClient()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	grpcReq := &ssov1.IsAdminRequest{}
	if reqBody.UserID != 0 {
		grpcReq.UserId = reqBody.UserID
	}

	if authHeader != "" {
		authHeader = strings.TrimSpace(authHeader)
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", authHeader)
		s.log.Debug("Proxy: Forwarding auth header to gRPC", "auth", authHeader)
	}

	resp, err := client.IsAdmin(ctx, grpcReq)
	if err != nil {
		s.handleGRPCError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"isAdmin": resp.GetIsAdmin(),
	})
}

func (s *ProxyServer) handleGRPCError(w http.ResponseWriter, err error) {
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.Unauthenticated:
			http.Error(w, "Unauthorized: "+st.Message(), http.StatusUnauthorized)
		case codes.NotFound:
			http.Error(w, "Not found: "+st.Message(), http.StatusNotFound)
		case codes.InvalidArgument:
			http.Error(w, "Bad request: "+st.Message(), http.StatusBadRequest)
		case codes.AlreadyExists:
			http.Error(w, "Already exists: "+st.Message(), http.StatusConflict)
		case codes.Internal:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		default:
			http.Error(w, "Error: "+st.Message(), http.StatusInternalServerError)
		}
	} else {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
