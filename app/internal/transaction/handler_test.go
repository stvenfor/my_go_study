// handler_test.go transaction 域 HTTP 层回归测试。
//
// 由 app/internal/delivery/http/controller/controller_test.go 的 4 个
// TestTransactionController_* 用例与 mockTransactionRepo 迁入（该文件仍在
// controller 包，保留 profile 的用例与 mockProfileRepo / mockStoreStatsRepo）。
//
// 只动两处：mock 改名 mockRepository 并对齐域内 Repository 接口；
// 取身份改用 authctx.SetIdentity（与 handler 的 authctx.SupabaseAuthContext 同源）。
package transaction

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"app/internal/delivery/http/authctx"
	pkgauth "common/auth"
	"github.com/gin-gonic/gin"
	"model/entity"
)

type mockRepository struct {
	listFn     func(accessToken, userID string, filter entity.TransactionFilter) ([]entity.Transaction, error)
	listPageFn func(accessToken, userID string, filter entity.TransactionFilter) ([]entity.Transaction, int64, error)
	getFn      func(accessToken, userID string, id int64) (*entity.Transaction, error)
	createFn   func(accessToken, userID string, input entity.CreateTransactionInput) (*entity.Transaction, error)
	updateFn   func(accessToken, userID string, id int64, input entity.UpdateTransactionInput) (*entity.Transaction, error)
	deleteFn   func(accessToken, userID string, id int64) error
}

func (m *mockRepository) List(_ context.Context, accessToken, userID string, filter entity.TransactionFilter) ([]entity.Transaction, error) {
	return m.listFn(accessToken, userID, filter)
}

func (m *mockRepository) ListPage(_ context.Context, accessToken, userID string, filter entity.TransactionFilter) ([]entity.Transaction, int64, error) {
	if m.listPageFn != nil {
		return m.listPageFn(accessToken, userID, filter)
	}
	items, err := m.listFn(accessToken, userID, filter)
	return items, int64(len(items)), err
}

func (m *mockRepository) GetByID(_ context.Context, accessToken, userID string, id int64) (*entity.Transaction, error) {
	return m.getFn(accessToken, userID, id)
}

func (m *mockRepository) Create(_ context.Context, accessToken, userID string, input entity.CreateTransactionInput) (*entity.Transaction, error) {
	return m.createFn(accessToken, userID, input)
}

func (m *mockRepository) Update(_ context.Context, accessToken, userID string, id int64, input entity.UpdateTransactionInput) (*entity.Transaction, error) {
	return m.updateFn(accessToken, userID, id, input)
}

func (m *mockRepository) Delete(_ context.Context, accessToken, userID string, id int64) error {
	return m.deleteFn(accessToken, userID, id)
}

func withSupabaseContext(userID, token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authctx.SetIdentity(c, pkgauth.SupabaseUser{ID: userID}, token)
		c.Next()
	}
}

func TestHandler_ListLegacy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &mockRepository{
		listFn: func(accessToken, userID string, filter entity.TransactionFilter) ([]entity.Transaction, error) {
			return []entity.Transaction{{ID: 1, Type: "income", Category: "工资", Amount: 100, Date: "2026-01-01"}}, nil
		},
	}
	h := newHandler(NewService(repo))

	r := gin.New()
	r.GET("/transactions", withSupabaseContext("user-1", "token-1"), h.List)

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

func TestHandler_ListPage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &mockRepository{
		listPageFn: func(accessToken, userID string, filter entity.TransactionFilter) ([]entity.Transaction, int64, error) {
			return []entity.Transaction{{ID: 2, Type: "expense", Category: "餐饮", Amount: 50, Date: "2026-01-02"}}, 1, nil
		},
	}
	h := newHandler(NewService(repo))

	r := gin.New()
	r.GET("/transactions/manage", withSupabaseContext("user-1", "token-1"), h.ListPage)

	req := httptest.NewRequest(http.MethodGet, "/transactions/manage?page=1&size=20", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestHandler_CreateLegacy_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &mockRepository{}
	h := newHandler(NewService(repo))

	r := gin.New()
	r.POST("/transactions", withSupabaseContext("user-1", "token-1"), h.CreateLegacy)

	req := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewBufferString(`{"type":"income"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestHandler_DeleteLegacy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deleted := false
	repo := &mockRepository{
		deleteFn: func(accessToken, userID string, id int64) error {
			if id == 9 {
				deleted = true
			}
			return nil
		},
	}
	h := newHandler(NewService(repo))

	r := gin.New()
	r.DELETE("/transactions/:id", withSupabaseContext("user-1", "token-1"), h.DeleteLegacy)

	req := httptest.NewRequest(http.MethodDelete, "/transactions/9", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent || !deleted {
		t.Fatalf("status = %d, deleted = %v", w.Code, deleted)
	}
}
