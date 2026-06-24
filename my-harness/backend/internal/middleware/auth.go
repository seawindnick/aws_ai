package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/workshop/wrong-question/internal/apperr"
	"github.com/workshop/wrong-question/internal/handler"
)

type contextKey string

const userIDKey contextKey = "user_id"

func CognitoJWTAuth(region, userPoolID string) func(http.Handler) http.Handler {
	jwksURL := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s/.well-known/jwks.json", region, userPoolID)
	keySet, _ := jwk.Fetch(context.Background(), jwksURL)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				handler.WriteError(w, apperr.ErrUnauthorized)
				return
			}
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			token, err := jwt.Parse([]byte(tokenStr), jwt.WithKeySet(keySet), jwt.WithValidate(true))
			if err != nil {
				handler.WriteError(w, apperr.ErrUnauthorized)
				return
			}

			sub, ok := token.Get("sub")
			if !ok {
				handler.WriteError(w, apperr.ErrUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, sub.(string))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromCtx(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok
}
