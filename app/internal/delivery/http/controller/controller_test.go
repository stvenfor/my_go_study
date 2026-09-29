package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"app/internal/delivery/http/middleware"
	"app/internal/usecase"
	pkgauth "common/auth"
	"github.com/gin-gonic/gin"
	"model/entity"
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

type mockStoreStatsRepo struct {
	loadFn   func(userID string, storeID int) (entity.UserStoreStats, error)
	switchFn func(userID string, storeID int) (entity.UserStoreStats, error)
	listFn   func(userID string) ([]entity.UserStoreListItem, error)
}

func (m *mockStoreStatsRepo) Load(_ context.Context, userID string, storeID int) (entity.UserStoreStats, error) {
	if m.loadFn == nil {
		return entity.ZeroUserStoreStats(), nil
	}
	return m.loadFn(userID, storeID)
}

func (m *mockStoreStatsRepo) Switch(_ context.Context, userID string, storeID int) (entity.UserStoreStats, error) {
	if m.switchFn == nil {
		return entity.ZeroUserStoreStats(), nil
	}
	return m.switchFn(userID, storeID)
}

func (m *mockStoreStatsRepo) ListMyStores(_ context.Context, userID string) ([]entity.UserStoreListItem, error) {
	if m.listFn == nil {
		return nil, nil
	}
	return m.listFn(userID)
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
	ctrl := NewProfileController(usecase.NewProfileUsecase(repo, nil))

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

func TestProfileController_GetMe_WithStats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	displayName := "Demo"
	repo := &mockProfileRepo{
		getFn: func(accessToken, userID string) (*entity.Profile, error) {
			return &entity.Profile{ID: userID, DisplayName: &displayName}, nil
		},
	}
	statsRepo := &mockStoreStatsRepo{
		loadFn: func(userID string, storeID int) (entity.UserStoreStats, error) {
			if storeID != 1 {
				t.Fatalf("store_id=%d", storeID)
			}
			role := entity.StoreRoleManager
			return entity.UserStoreStats{
				StoreID:        1,
				StoreName:      "[4S]北京沃德龙鼎吉利",
				DaysJoined:     1028,
				EmployeeCount:  28,
				StoreDays:      2059,
				TotalCustomers: 9366,
				Role:           &role,
				RoleLabel:      entity.StoreRoleLabel(entity.StoreRoleManager),
			}, nil
		},
	}
	ctrl := NewProfileController(usecase.NewProfileUsecase(repo, statsRepo))

	r := gin.New()
	r.GET("/profiles/me", withSupabaseContext("user-1", "token-1"), ctrl.GetMe)

	req := httptest.NewRequest(http.MethodGet, "/profiles/me?store_id=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			Stats struct {
				DaysJoined     int    `json:"days_joined"`
				EmployeeCount  int    `json:"employee_count"`
				StoreDays      int    `json:"store_days"`
				TotalCustomers int    `json:"total_customers"`
				Role           int    `json:"role"`
				RoleLabel      string `json:"role_label"`
			} `json:"stats"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Data.Stats.EmployeeCount != 28 || body.Data.Stats.TotalCustomers != 9366 {
		t.Fatalf("unexpected stats: %+v", body.Data.Stats)
	}
	if body.Data.Stats.Role != 1 || body.Data.Stats.RoleLabel != "销售经理" {
		t.Fatalf("unexpected role: %+v", body.Data.Stats)
	}
}

func TestProfileController_GetMe_MissingMembershipZeros(t *testing.T) {
	gin.SetMode(gin.TestMode)
	displayName := "Demo"
	repo := &mockProfileRepo{
		getFn: func(accessToken, userID string) (*entity.Profile, error) {
			return &entity.Profile{ID: userID, DisplayName: &displayName}, nil
		},
	}
	statsRepo := &mockStoreStatsRepo{
		loadFn: func(userID string, storeID int) (entity.UserStoreStats, error) {
			return entity.ZeroUserStoreStats(), nil
		},
	}
	ctrl := NewProfileController(usecase.NewProfileUsecase(repo, statsRepo))

	r := gin.New()
	r.GET("/profiles/me", withSupabaseContext("user-1", "token-1"), ctrl.GetMe)

	req := httptest.NewRequest(http.MethodGet, "/profiles/me?store_id=99", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var body struct {
		Data struct {
			Stats map[string]any `json:"stats"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"days_joined", "employee_count", "store_days", "total_customers"} {
		value, ok := body.Data.Stats[key].(float64)
		if !ok || value != 0 {
			t.Fatalf("key %s = %v ok=%v", key, body.Data.Stats[key], ok)
		}
	}
	if body.Data.Stats["role"] != nil {
		t.Fatalf("role = %v", body.Data.Stats["role"])
	}
	if body.Data.Stats["role_label"] != "" {
		t.Fatalf("role_label = %v", body.Data.Stats["role_label"])
	}
}

func TestProfileController_GetMe_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := NewProfileController(usecase.NewProfileUsecase(&mockProfileRepo{}, nil))
	r := gin.New()
	r.GET("/profiles/me", ctrl.GetMe)

	req := httptest.NewRequest(http.MethodGet, "/profiles/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", w.Code)
	}
}
