package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDefaultBackendCode(t *testing.T) {
	if got := defaultBackendCode(http.StatusUnauthorized); got != CodeTokenInvalid {
		t.Fatalf("401 → CodeTokenInvalid, got %d", got)
	}
	if got := defaultBackendCode(http.StatusForbidden); got != CodeForbidden {
		t.Fatalf("403 → CodeForbidden, got %d", got)
	}
}

func TestBackendErrorIncludesCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	BackendError(c, http.StatusUnauthorized, "token 无效")

	var body BackendErrorBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error != "token 无效" {
		t.Fatalf("error=%q", body.Error)
	}
	if body.Code != CodeTokenInvalid {
		t.Fatalf("code=%d want %d", body.Code, CodeTokenInvalid)
	}
}

func TestBackendErrorWithCodeExplicit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	BackendErrorWithCode(c, http.StatusUnauthorized, CodeUnauthorized, "密码错误")

	var body BackendErrorBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != CodeUnauthorized {
		t.Fatalf("code=%d want %d", body.Code, CodeUnauthorized)
	}
}
