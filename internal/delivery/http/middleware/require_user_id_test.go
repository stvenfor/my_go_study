package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/middleware"
	pkgauth "github.com/stvenfor/my_go_study/pkg/auth"
)

func TestRequireUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const account = "a1111111-1111-4111-8111-111111111111"

	newRouter := func() *gin.Engine {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set(middleware.ContextSupabaseUserKey, pkgauth.SupabaseUser{ID: account})
			c.Next()
		})
		r.GET("/me", middleware.RequireUserID(), func(c *gin.Context) { c.Status(http.StatusOK) })
		r.PATCH("/me", middleware.RequireUserID(), func(c *gin.Context) { c.Status(http.StatusOK) })
		return r
	}

	t.Run("missing", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		newRouter().ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("got %d %s", w.Code, w.Body.String())
		}
	})

	t.Run("mismatch", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/me?user_id=other", nil)
		newRouter().ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("got %d %s", w.Code, w.Body.String())
		}
	})

	t.Run("query match", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/me?user_id="+account, nil)
		newRouter().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("got %d %s", w.Code, w.Body.String())
		}
	})

	t.Run("body match", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := strings.NewReader(`{"user_id":"` + account + `","display_name":"n"}`)
		req := httptest.NewRequest(http.MethodPatch, "/me", body)
		req.Header.Set("Content-Type", "application/json")
		newRouter().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("got %d %s", w.Code, w.Body.String())
		}
	})
}
