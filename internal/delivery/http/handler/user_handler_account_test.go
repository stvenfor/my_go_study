package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/middleware"
	"github.com/stvenfor/my_go_study/internal/usecase"
	pkgauth "github.com/stvenfor/my_go_study/pkg/auth"
)

type fakeSession struct {
	registerEmail string
	deactivated   string
}

func (f *fakeSession) Register(_ context.Context, input usecase.RegisterInput) (*usecase.SupabaseAuthOutput, error) {
	f.registerEmail = input.Email
	return &usecase.SupabaseAuthOutput{UserID: "server-user", Username: input.Username, Email: input.Email}, nil
}

func (f *fakeSession) Login(context.Context, usecase.LoginInput) (*usecase.SupabaseAuthOutput, error) {
	return nil, usecase.ErrInvalidCredentials
}

func (f *fakeSession) RefreshToken(context.Context, string) (*usecase.SupabaseAuthOutput, error) {
	return nil, usecase.ErrInvalidCredentials
}

func (f *fakeSession) Logout(context.Context, string) error { return nil }

func (f *fakeSession) Deactivate(_ context.Context, userID string) error {
	f.deactivated = userID
	return nil
}

func TestRegisterIgnoresClientUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fake := &fakeSession{}
	h := NewUserHandler(fake, nil, nil)
	r := gin.New()
	r.POST("/register", h.Register)

	body := `{"username":"ann","password":"secret1","email":"Ann@Example.com","device_id":"d","platform":"ios","user_id":"client-supplied"}`
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if fake.registerEmail != "Ann@Example.com" {
		t.Fatalf("email=%s", fake.registerEmail)
	}
	var payload struct {
		Data struct {
			UserID string `json:"user_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Data.UserID != "server-user" {
		t.Fatalf("user_id=%s body=%s", payload.Data.UserID, w.Body.String())
	}
}

func TestDeactivateUsesSessionUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fake := &fakeSession{}
	h := NewUserHandler(fake, nil, nil)
	r := gin.New()
	r.POST("/deactivate", func(c *gin.Context) {
		c.Set(middleware.ContextSupabaseUserKey, pkgauth.SupabaseUser{ID: "server-user"})
		c.Next()
	}, h.Deactivate)

	req := httptest.NewRequest(http.MethodPost, "/deactivate", strings.NewReader(`{"user_id":"server-user"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if fake.deactivated != "server-user" {
		t.Fatalf("deactivated=%s", fake.deactivated)
	}
}
