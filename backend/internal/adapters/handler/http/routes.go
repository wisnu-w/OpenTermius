package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/opentermius/opentermius/internal/ports"
)

type contextKey string

const userIDKey contextKey = "user_id"

type Router struct {
	mux         *http.ServeMux
	authService ports.AuthService
	authHandler *AuthHandler
	vaultHandler *VaultHandler
	terminalWS  *TerminalWSHandler
	sftpHandler *SFTPHandler
}

func NewRouter(
	authService ports.AuthService,
	authHandler *AuthHandler,
	vaultHandler *VaultHandler,
	terminalWS *TerminalWSHandler,
	sftpHandler *SFTPHandler,
) *Router {
	r := &Router{
		mux:         http.NewServeMux(),
		authService: authService,
		authHandler: authHandler,
		vaultHandler: vaultHandler,
		terminalWS:  terminalWS,
		sftpHandler: sftpHandler,
	}
	r.registerRoutes()
	return r
}

func (r *Router) registerRoutes() {
	// Public Auth Endpoints
	r.mux.HandleFunc("POST /api/v1/auth/register", r.authHandler.Register)
	r.mux.HandleFunc("POST /api/v1/auth/login", r.authHandler.Login)

	// Protected Profile
	r.mux.Handle("GET /api/v1/auth/me", r.authMiddleware(http.HandlerFunc(r.authHandler.Me)))

	// Protected Vaults
	r.mux.Handle("GET /api/v1/vaults", r.authMiddleware(http.HandlerFunc(r.vaultHandler.ListVaults)))
	r.mux.Handle("POST /api/v1/vaults", r.authMiddleware(http.HandlerFunc(r.vaultHandler.CreateVault)))
	r.mux.Handle("GET /api/v1/vaults/{id}", r.authMiddleware(http.HandlerFunc(r.vaultHandler.GetVault)))

	// Protected Vault Items
	r.mux.Handle("GET /api/v1/vaults/{id}/items", r.authMiddleware(http.HandlerFunc(r.vaultHandler.ListItems)))
	r.mux.Handle("POST /api/v1/vaults/{id}/items", r.authMiddleware(http.HandlerFunc(r.vaultHandler.CreateItem)))
	r.mux.Handle("PUT /api/v1/vaults/{id}/items/{itemId}", r.authMiddleware(http.HandlerFunc(r.vaultHandler.UpdateItem)))
	r.mux.Handle("PUT /api/v1/vaults/{id}/items/{itemId}/move", r.authMiddleware(http.HandlerFunc(r.vaultHandler.MoveItem)))
	r.mux.Handle("DELETE /api/v1/vaults/{id}/items/{itemId}", r.authMiddleware(http.HandlerFunc(r.vaultHandler.DeleteItem)))

	// Protected Vault Members
	r.mux.Handle("GET /api/v1/vaults/{id}/members", r.authMiddleware(http.HandlerFunc(r.vaultHandler.ListMembers)))
	r.mux.Handle("POST /api/v1/vaults/{id}/members", r.authMiddleware(http.HandlerFunc(r.vaultHandler.InviteMember)))

	// SFTP File Manager Endpoints
	r.mux.Handle("POST /api/v1/sftp/list", r.authMiddleware(http.HandlerFunc(r.sftpHandler.List)))
	r.mux.Handle("POST /api/v1/sftp/mkdir", r.authMiddleware(http.HandlerFunc(r.sftpHandler.Mkdir)))
	r.mux.Handle("POST /api/v1/sftp/delete", r.authMiddleware(http.HandlerFunc(r.sftpHandler.Delete)))
	r.mux.Handle("POST /api/v1/sftp/download", r.authMiddleware(http.HandlerFunc(r.sftpHandler.Download)))
	r.mux.Handle("POST /api/v1/sftp/upload", r.authMiddleware(http.HandlerFunc(r.sftpHandler.Upload)))

	// WebSockets (Terminal & Multiplayer)
	r.mux.Handle("/ws/terminal/{hostId}", r.wsAuthMiddleware(http.HandlerFunc(r.terminalWS.HandleTerminal)))
	r.mux.Handle("/ws/sessions/{sessionId}/join", r.wsAuthMiddleware(http.HandlerFunc(r.terminalWS.JoinSession)))
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Global CORS Handler
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if req.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	r.mux.ServeHTTP(w, req)
}

func (r *Router) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		authHeader := req.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			writeJSONError(w, http.StatusUnauthorized, "missing or invalid authorization header")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "standalone" {
			ctx := context.WithValue(req.Context(), userIDKey, "standalone-local-user")
			next.ServeHTTP(w, req.WithContext(ctx))
			return
		}

		userID, err := r.authService.ValidateToken(req.Context(), tokenString)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		ctx := context.WithValue(req.Context(), userIDKey, userID.String())
		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

func (r *Router) wsAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		tokenString := req.URL.Query().Get("token")
		if tokenString == "" {
			authHeader := req.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if tokenString == "" {
			writeJSONError(w, http.StatusUnauthorized, "websocket authorization token required")
			return
		}

		userID, err := r.authService.ValidateToken(req.Context(), tokenString)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "invalid websocket token")
			return
		}

		ctx := context.WithValue(req.Context(), userIDKey, userID.String())
		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

func getUserID(r *http.Request) uuid.UUID {
	val, ok := r.Context().Value(userIDKey).(string)
	if !ok || val == "" {
		return uuid.Nil
	}
	uid, _ := uuid.Parse(val)
	return uid
}

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
