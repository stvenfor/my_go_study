package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/middleware"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/usecase"
	pkgauth "github.com/stvenfor/my_go_study/pkg/auth"
)

type mockProfileRepo struct {
	getFn    func(accessToken, userID string) (*entity.Profile, error)
	updateFn func(accessToken, userID string, input entity.UpdateProfileInput) (*entity.Profile, error)
}

func (m *mockProfileRepo) GetByUserID(_ context.Context, accessToken, userID string) (*entity.Profile, error) {
	return m.getFn(accessToken, userID)
}

func (m *mockProfileRepo) UpdateByUserID(_ context.Context, accessToken, userID string, input entity.UpdateProfileInput) (*entity.Profile, error) {
	return m.updateFn(accessToken, userID, input)
}

type mockTransactionRepo struct {
	listFn     func(accessToken, userID string, filter entity.TransactionFilter) ([]entity.Transaction, error)
	listPageFn func(accessToken, userID string, filter entity.TransactionFilter) ([]entity.Transaction, int64, error)
	getFn      func(accessToken, userID string, id int64) (*entity.Transaction, error)
	createFn   func(accessToken, userID string, input entity.CreateTransactionInput) (*entity.Transaction, error)
	updateFn   func(accessToken, userID string, id int64, input entity.UpdateTransactionInput) (*entity.Transaction, error)
	deleteFn   func(accessToken, userID string, id int64) error
}

func (m *mockTransactionRepo) List(_ context.Context, accessToken, userID string, filter entity.TransactionFilter) ([]entity.Transaction, error) {
	return m.listFn(accessToken, userID, filter)
}

func (m *mockTransactionRepo) ListPage(_ context.Context, accessToken, userID string, filter entity.TransactionFilter) ([]entity.Transaction, int64, error) {
	if m.listPageFn != nil {
		return m.listPageFn(accessToken, userID, filter)
	}
	items, err := m.listFn(accessToken, userID, filter)
	return items, int64(len(items)), err
}

func (m *mockTransactionRepo) GetByID(_ context.Context, accessToken, userID string, id int64) (*entity.Transaction, error) {
	return m.getFn(accessToken, userID, id)
}

func (m *mockTransactionRepo) Create(_ context.Context, accessToken, userID string, input entity.CreateTransactionInput) (*entity.Transaction, error) {
	return m.createFn(accessToken, userID, input)
}

func (m *mockTransactionRepo) Update(_ context.Context, accessToken, userID string, id int64, input entity.UpdateTransactionInput) (*entity.Transaction, error) {
	return m.updateFn(accessToken, userID, id, input)
}

func (m *mockTransactionRepo) Delete(_ context.Context, accessToken, userID string, id int64) error {
	return m.deleteFn(accessToken, userID, id)
}

func withSupabaseContext(userID, token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.ContextSupabaseUserKey, pkgauth.SupabaseUser{ID: userID})
		c.Set(middleware.ContextAccessTokenKey, token)
		c.Next()
	}
}

func TestProfileController_GetMe(t *testing.T) {
	gin.SetMode(gin.TestMode)
	displayName := "测试用户"
	repo := &mockProfileRepo{
		getFn: func(accessToken, userID string) (*entity.Profile, error) {
			return &entity.Profile{ID: userID, DisplayName: &displayName}, nil
		},
	}
	ctrl := NewProfileController(usecase.NewProfileUsecase(repo))

	r := gin.New()
	r.GET("/profiles/me", withSupabaseContext("user-1", "token-1"), ctrl.GetMe)

	req := httptest.NewRequest(http.MethodGet, "/profiles/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != 0 || body.Data.ID != "user-1" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestTransactionController_ListLegacy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &mockTransactionRepo{
		listFn: func(accessToken, userID string, filter entity.TransactionFilter) ([]entity.Transaction, error) {
			return []entity.Transaction{{ID: 1, Type: "income", Category: "工资", Amount: 100, Date: "2026-01-01"}}, nil
		},
	}
	ctrl := NewTransactionController(usecase.NewTransactionUsecase(repo))

	r := gin.New()
	r.GET("/transactions", withSupabaseContext("user-1", "token-1"), ctrl.List)

	req := httptest.NewRequest(http.MethodGet, "/transactions?limit=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var body struct {
		Items []entity.Transaction `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 {
		t.Fatalf("unexpected items: %+v", body.Items)
	}
}

func TestTransactionController_ListPage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &mockTransactionRepo{
		listPageFn: func(accessToken, userID string, filter entity.TransactionFilter) ([]entity.Transaction, int64, error) {
			return []entity.Transaction{{ID: 2, Type: "expense", Category: "餐饮", Amount: 50, Date: "2026-01-02"}}, 1, nil
		},
	}
	ctrl := NewTransactionController(usecase.NewTransactionUsecase(repo))

	r := gin.New()
	r.GET("/transactions/manage", withSupabaseContext("user-1", "token-1"), ctrl.ListPage)

	req := httptest.NewRequest(http.MethodGet, "/transactions/manage?page=1&size=20", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestTransactionController_CreateLegacy_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &mockTransactionRepo{}
	ctrl := NewTransactionController(usecase.NewTransactionUsecase(repo))

	r := gin.New()
	r.POST("/transactions", withSupabaseContext("user-1", "token-1"), ctrl.CreateLegacy)

	req := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewBufferString(`{"type":"income"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestTransactionController_DeleteLegacy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deleted := false
	repo := &mockTransactionRepo{
		deleteFn: func(accessToken, userID string, id int64) error {
			if id == 9 {
				deleted = true
			}
			return nil
		},
	}
	ctrl := NewTransactionController(usecase.NewTransactionUsecase(repo))

	r := gin.New()
	r.DELETE("/transactions/:id", withSupabaseContext("user-1", "token-1"), ctrl.DeleteLegacy)

	req := httptest.NewRequest(http.MethodDelete, "/transactions/9", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent || !deleted {
		t.Fatalf("status = %d, deleted = %v", w.Code, deleted)
	}
}
