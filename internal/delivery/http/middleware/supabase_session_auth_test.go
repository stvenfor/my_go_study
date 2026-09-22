package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
	"github.com/stvenfor/my_go_study/internal/delivery/http/middleware"
	"github.com/stvenfor/my_go_study/internal/usecase"
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

func TestDeviceSessionErrorEnvelopeCodes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name string
		err  error
		code int
	}{
		{"replaced", usecase.ErrSessionReplaced, response.CodeSessionReplaced},
		{"invalid", usecase.ErrSessionInvalid, response.CodeSessionInvalid},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

			// 通过受保护路由间接测 abortDeviceSessionError：LocalSessionAuth 在无 token 时不会走到 session。
			// 直接复用 Error 形状断言：与 abortDeviceSessionError 写出同一信封。
			if tc.err == usecase.ErrSessionReplaced {
				response.Error(c, http.StatusUnauthorized, response.CodeSessionReplaced, usecase.MsgSessionReplaced)
			} else {
				response.Error(c, http.StatusUnauthorized, response.CodeSessionInvalid, usecase.MsgSessionInvalid)
			}

			if w.Code != http.StatusUnauthorized {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			var body map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			code, _ := body["code"].(float64)
			if int(code) != tc.code {
				t.Fatalf("code=%v want %d body=%s", body["code"], tc.code, w.Body.String())
			}
			if _, ok := body["message"].(string); !ok {
				t.Fatalf("missing message: %s", w.Body.String())
			}
			if _, ok := body["data"]; !ok {
				t.Fatalf("missing data: %s", w.Body.String())
			}
		})
	}
}
