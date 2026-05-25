package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/evadeplayer/api/internal/service"
)

type contextKey string

const userIDKey contextKey = "user_id"
const serviceUserID = "00000000-0000-0000-0000-000000000001"

func CORSMiddleware(origins []string) func(http.Handler) http.Handler {
	allowAll := len(origins) == 1 && origins[0] == "*"

	allowed := make(map[string]bool, len(origins))
	for _, o := range origins {
		allowed[o] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if allowAll {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if origin != "" && allowed[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Service-Key, X-User-ID")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func AuthMiddleware(authSvc *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := extractToken(r, authSvc)
			if !ok {
				writeError(w, http.StatusUnauthorized, "missing or invalid authorization header")
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func OptionalAuthMiddleware(authSvc *service.AuthService, required bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" && !required {
				next.ServeHTTP(w, r)
				return
			}
			userID, ok := extractToken(r, authSvc)
			if !ok {
				writeError(w, http.StatusUnauthorized, "missing or invalid authorization header")
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractToken(r *http.Request, authSvc *service.AuthService) (string, bool) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return "", false
	}
	userID, err := authSvc.ValidateToken(strings.TrimPrefix(header, "Bearer "))
	if err != nil {
		return "", false
	}
	return userID, true
}

func userIDFromCtx(r *http.Request) string {
	v, _ := r.Context().Value(userIDKey).(string)
	return v
}

// ServiceKeyMiddleware requires a valid X-Service-Key header.
func ServiceKeyMiddleware(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Service-Key") != key {
				writeError(w, http.StatusUnauthorized, "invalid or missing service key")
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey, serviceUserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// BFFMiddleware authenticates requests from a trusted BFF backend.
// It validates the service key and reads X-User-ID as the acting user.
func BFFMiddleware(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Service-Key") != key {
				writeError(w, http.StatusUnauthorized, "invalid or missing service key")
				return
			}
			userID := strings.TrimSpace(r.Header.Get("X-User-ID"))
			if userID == "" {
				writeError(w, http.StatusBadRequest, "X-User-ID header is required")
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AnyAuthMiddleware accepts either a valid X-Service-Key header or a user JWT.
func AnyAuthMiddleware(authSvc *service.AuthService, serviceKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if sk := r.Header.Get("X-Service-Key"); sk != "" {
				if sk != serviceKey {
					writeError(w, http.StatusUnauthorized, "invalid service key")
					return
				}
				ctx := context.WithValue(r.Context(), userIDKey, serviceUserID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			userID, ok := extractToken(r, authSvc)
			if !ok {
				writeError(w, http.StatusUnauthorized, "missing or invalid authorization")
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
