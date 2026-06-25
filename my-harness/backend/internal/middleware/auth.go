package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/workshop/wrong-question/internal/repository"
)

type contextKey string

const (
	userIDKey contextKey = "user_id"
	roleKey   contextKey = "role"
)

func CognitoJWTAuth(region, userPoolID string, userRepo *repository.UserRepo) func(http.Handler) http.Handler {
	jwksURL := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s/.well-known/jwks.json", region, userPoolID)

	cache := jwk.NewCache(context.Background())
	if err := cache.Register(jwksURL, jwk.WithMinRefreshInterval(15*time.Minute)); err != nil {
		slog.Error("register jwks cache", "url", jwksURL, "error", err)
		panic(fmt.Sprintf("cannot register JWKS cache: %v", err))
	}
	// Warm up the cache on startup; fail fast if Cognito is unreachable.
	if _, err := cache.Refresh(context.Background(), jwksURL); err != nil {
		slog.Error("initial jwks fetch failed", "url", jwksURL, "error", err)
		panic(fmt.Sprintf("cannot fetch JWKS on startup: %v", err))
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				writeErrorJSON(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			keySet, err := cache.Get(r.Context(), jwksURL)
			if err != nil {
				slog.Error("get jwks from cache", "error", err)
				writeErrorJSON(w, http.StatusServiceUnavailable, "authentication service unavailable")
				return
			}

			token, err := jwt.Parse([]byte(tokenStr), jwt.WithKeySet(keySet), jwt.WithValidate(true))
			if err != nil {
				writeErrorJSON(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			sub, ok := token.Get("sub")
			if !ok {
				writeErrorJSON(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			user, err := userRepo.GetByCognitoSub(r.Context(), sub.(string))
			if err != nil {
				writeErrorJSON(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			if user.Status == "inactive" {
				writeErrorJSON(w, http.StatusUnauthorized, "account is deactivated")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, user.ID)
			ctx = context.WithValue(ctx, roleKey, string(user.Role))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromCtx(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok
}

func RoleFromCtx(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(roleKey).(string)
	return role, ok
}
