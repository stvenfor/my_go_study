package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
)

func TestWriteResourceCRUDError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	invalid := errors.New("参数无效")
	notFound := errors.New("gone")
	forbidden := errors.New("nope")

	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   int
		wantMsg    string
	}{
		{"invalid", invalid, http.StatusBadRequest, response.CodeInvalidParams, "参数无效"},
		{"notFound", notFound, http.StatusNotFound, response.CodeNotFound, "资源不存在"},
		{"forbidden", forbidden, http.StatusForbidden, response.CodeForbidden, "无权限"},
		{"other", errors.New("boom"), http.StatusInternalServerError, response.CodeInternalError, "操作失败"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			writeResourceCRUDError(c, tc.err, invalid, notFound, forbidden)
			if w.Code != tc.wantStatus {
				t.Fatalf("status=%d want %d", w.Code, tc.wantStatus)
			}
			var body response.Response
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Code != tc.wantCode || body.Message != tc.wantMsg {
				t.Fatalf("got code=%d msg=%q want %d %q", body.Code, body.Message, tc.wantCode, tc.wantMsg)
			}
		})
	}
}
