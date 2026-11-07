package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"app/internal/delivery/http/dto/response"
	"app/internal/usecase"
	"common/config"
	"github.com/gin-gonic/gin"
)

// recordingEnqueuer 是 RealtimePushUsecase.PushToUser（异步模式）的唯一外部副作用。
// controller 一旦调用 usecase.PushToUser，这里必然收到一次入队；反之 calls==0 即证明
// usecase 未被调用、未入队。lastInput 用于断言目标锁定为会话用户。
type recordingEnqueuer struct {
	calls     int
	lastInput usecase.RealtimePushInput
}

func (e *recordingEnqueuer) EnqueueRealtimePush(_ context.Context, input usecase.RealtimePushInput) (string, error) {
	e.calls++
	e.lastInput = input
	return "task-1", nil
}

func newSpyRealtimeController() (*RealtimeController, *recordingEnqueuer) {
	cfg := config.Config{Queue: config.QueueConfig{Enabled: true}} // UseAsyncPush() == true
	enq := &recordingEnqueuer{}
	uc := usecase.NewRealtimePushUsecase(nil, cfg, nil, enq)
	return NewRealtimeController(nil, nil, uc), enq
}

func realtimePushRouter(ctrl *RealtimeController) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/realtime/push", withSupabaseContext("user-1", "token-1"), ctrl.Push)
	return r
}

func doRealtimePush(r *gin.Engine, payload string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/realtime/push", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// 他人 userId → 403，且 usecase 未被调用、未入队。
func TestRealtimeController_Push_RejectsOtherUser(t *testing.T) {
	ctrl, enq := newSpyRealtimeController()
	w := doRealtimePush(realtimePushRouter(ctrl), `{"userId":"other-user","title":"t","body":"b"}`)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var got struct {
		Code int `json:"code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.Code != response.CodeForbidden {
		t.Fatalf("code = %d, want %d", got.Code, response.CodeForbidden)
	}
	if enq.calls != 0 {
		t.Fatalf("usecase.PushToUser 被调用了 %d 次，越权请求不应入队", enq.calls)
	}
}

// 自身路径：userId 留空 → 200，且 usecase 被调用、入队目标为本人。
func TestRealtimeController_Push_SelfEmptyUserID(t *testing.T) {
	ctrl, enq := newSpyRealtimeController()
	w := doRealtimePush(realtimePushRouter(ctrl), `{"title":"t","body":"b"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if enq.calls != 1 {
		t.Fatalf("usecase.PushToUser calls = %d, want 1 (自身路径不应回归)", enq.calls)
	}
	if enq.lastInput.UserID != "user-1" {
		t.Fatalf("enqueued UserID = %q, want user-1", enq.lastInput.UserID)
	}
}

// 自身路径：userId 填自己 → 200，且 usecase 被调用。
func TestRealtimeController_Push_SelfExplicitUserID(t *testing.T) {
	ctrl, enq := newSpyRealtimeController()
	w := doRealtimePush(realtimePushRouter(ctrl), `{"userId":"user-1","title":"t","body":"b"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if enq.calls != 1 {
		t.Fatalf("usecase.PushToUser calls = %d, want 1", enq.calls)
	}
	if enq.lastInput.UserID != "user-1" {
		t.Fatalf("enqueued UserID = %q, want user-1", enq.lastInput.UserID)
	}
}
