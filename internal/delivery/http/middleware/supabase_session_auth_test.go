package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/middleware"
	"github.com/stvenfor/my_go_study/pkg/config"
)

func TestSupabaseSessionAuthMissingAuthorization(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/protected", middleware.SupabaseSessionAuth(config.SupabaseConfig{}, nil), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", w.Code, w.Body.String())
	}
}
