package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/stvenfor/my_go_study/internal/auth"
	"github.com/stvenfor/my_go_study/internal/config"
	"github.com/stvenfor/my_go_study/internal/httpx"
)

type contextKey string

const (
	AccessTokenKey contextKey = "accessToken"
	UserKey        contextKey = "user"
)

func Auth(cfg config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r.Header.Get("Authorization"))
			user, err := auth.ValidateAccessToken(r.Context(), cfg.Supabase, token)
			if err != nil {
				httpx.Error(w, http.StatusUnauthorized, err.Error())
				return
			}

			ctx := context.WithValue(r.Context(), AccessTokenKey, token)
			ctx = context.WithValue(ctx, UserKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AccessToken(ctx context.Context) string {
	token, _ := ctx.Value(AccessTokenKey).(string)
	return token
}

func User(ctx context.Context) auth.User {
	user, _ := ctx.Value(UserKey).(auth.User)
	return user
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if strings.HasPrefix(header, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(header, prefix))
	}
	return strings.TrimSpace(header)
}
