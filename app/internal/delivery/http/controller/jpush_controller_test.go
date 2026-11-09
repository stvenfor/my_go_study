package controller

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"app/internal/delivery/http/dto/response"
	"app/internal/usecase"
	"core/jpush"
	"github.com/gin-gonic/gin"
	"model/entity"
)

// jpushRecordingTransport 是 JPushUsecase.Send → jpush.Client.Send 的唯一外部副作用。
// controller 一旦调用 usecase.Send，这里必然收到一次 HTTP 请求；反之 calls==0 即证明
// usecase 未被调用（本用例 client 已配置，Send 没有提前 return 的分支）。
// bodies 记录下发体，用于断言目标被锁定为会话用户、alias/registration_ids 被忽略。
type jpushRecordingTransport struct {
	calls  int
	bodies []map[string]any
}

func (t *jpushRecordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.calls++
	var body map[string]any
	if req.Body != nil {
		raw, _ := io.ReadAll(req.Body)
		_ = json.Unmarshal(raw, &body)
	}
	t.bodies = append(t.bodies, body)
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(`{"msg_id":"m1","sendno":"1"}`)),
		Request:    req,
	}, nil
}

func newSpyJPushController() (*JPushController, *jpushRecordingTransport) {
	rt := &jpushRecordingTransport{}
	client := jpush.NewClient(jpush.Config{
		AppKey:       "test-app-key",
		MasterSecret: "test-master-secret",
		HTTPClient:   &http.Client{Transport: rt},
	})
	return NewJPushController(usecase.NewJPushUsecase(nil, client)), rt
}

func jpushSendRouter(ctrl *JPushController) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/push/send", withSupabaseContext("user-1", "token-1"), ctrl.Send)
	return r
}

func doJPushSend(r *gin.Engine, payload string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/push/send", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// assertSelfAudience 断言极光下发目标只剩会话用户本人别名，registration_id 不存在。
func assertSelfAudience(t *testing.T, rt *jpushRecordingTransport) {
	t.Helper()
	if len(rt.bodies) != 1 {
		t.Fatalf("expected 1 captured body, got %d", len(rt.bodies))
	}
	aud, _ := rt.bodies[0]["audience"].(map[string]any)
	if aud == nil {
		t.Fatalf("audience missing in jpush body: %+v", rt.bodies[0])
	}
	if _, present := aud["registration_id"]; present {
		t.Fatalf("registration_id must be absent, got audience=%+v", aud)
	}
	alias, _ := aud["alias"].([]any)
	if len(alias) != 1 || alias[0] != "user-1" {
		t.Fatalf("alias must equal [user-1], got %+v", aud["alias"])
	}
}

// 他人 user_id → 403，且 usecase 未被调用（不入极光下发）。
func TestJPushController_Send_RejectsOtherUser(t *testing.T) {
	for _, payload := range []string{
		`{"user_id":"other-user","body":"hi"}`,
		`{"user_id":"other-user","alias":"other-alias","registration_ids":["other-rid"],"body":"hi"}`,
	} {
		ctrl, rt := newSpyJPushController()
		w := doJPushSend(jpushSendRouter(ctrl), payload)

		if w.Code != http.StatusForbidden {
			t.Fatalf("payload=%s status = %d, body = %s", payload, w.Code, w.Body.String())
		}
		var got struct {
			Code int `json:"code"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &got)
		if got.Code != response.CodeForbidden {
			t.Fatalf("payload=%s code = %d, want %d", payload, got.Code, response.CodeForbidden)
		}
		if rt.calls != 0 {
			t.Fatalf("payload=%s usecase.Send 被调用了 %d 次，越权请求不应下发", payload, rt.calls)
		}
	}
}

// 自身路径：user_id 留空 → 200，且 usecase 被调用，目标锁定本人。
func TestJPushController_Send_SelfEmptyUserID(t *testing.T) {
	ctrl, rt := newSpyJPushController()
	w := doJPushSend(jpushSendRouter(ctrl), `{"body":"hi"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if rt.calls != 1 {
		t.Fatalf("usecase.Send calls = %d, want 1 (自身路径不应回归)", rt.calls)
	}
	assertSelfAudience(t, rt)
}

// 自身路径：user_id 填自己 → 200，且 usecase 被调用。
func TestJPushController_Send_SelfExplicitUserID(t *testing.T) {
	ctrl, rt := newSpyJPushController()
	w := doJPushSend(jpushSendRouter(ctrl), `{"user_id":"user-1","body":"hi"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if rt.calls != 1 {
		t.Fatalf("usecase.Send calls = %d, want 1", rt.calls)
	}
	assertSelfAudience(t, rt)
}

// 客户端指定 alias / registration_ids（他人目标）→ 忽略并留 warn，仍只向本人下发。
func TestJPushController_Send_IgnoresAliasAndRegistrationIDs(t *testing.T) {
	ctrl, rt := newSpyJPushController()
	payload := `{"alias":"other-user-alias","registration_ids":["someone-else-rid-1","someone-else-rid-2"],"body":"hi"}`
	w := doJPushSend(jpushSendRouter(ctrl), payload)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if rt.calls != 1 {
		t.Fatalf("usecase.Send calls = %d, want 1", rt.calls)
	}
	// 关键断言：极光受众里既无他人 alias，也无客户端传入的 registration_ids。
	assertSelfAudience(t, rt)
}

// jpushCapturingRepo 捕获 RegisterDevice 落库实体，用于断言 alias 被强制为会话用户。
type jpushCapturingRepo struct {
	got entity.WysPushDevice
}

func (r *jpushCapturingRepo) Upsert(_ context.Context, device entity.WysPushDevice) (*entity.WysPushDevice, error) {
	r.got = device
	return &device, nil
}

func jpushRegisterRouter(ctrl *JPushController) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/push/devices", withSupabaseContext("user-1", "token-1"), ctrl.RegisterDevice)
	return r
}

// 客户端传他人 alias → 落库/入参 alias 仍为会话 user.ID（h4）；不传 alias 也一致。
func TestJPushController_RegisterDevice_ForcesSessionAlias(t *testing.T) {
	for _, payload := range []string{
		`{"device_id":"d1","platform":"ios","registration_id":"rid-1","alias":"other-user-alias"}`,
		`{"device_id":"d1","platform":"ios","registration_id":"rid-1"}`,
	} {
		repo := &jpushCapturingRepo{}
		ctrl := NewJPushController(usecase.NewJPushUsecase(repo, nil))
		req := httptest.NewRequest(http.MethodPost, "/push/devices", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		jpushRegisterRouter(ctrl).ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("payload=%s status = %d, body = %s", payload, w.Code, w.Body.String())
		}
		if repo.got.Alias != "user-1" {
			t.Fatalf("payload=%s 落库 alias = %q, want 会话用户 user-1", payload, repo.got.Alias)
		}
		if repo.got.UserID != "user-1" || repo.got.RegistrationID != "rid-1" {
			t.Fatalf("payload=%s 其它字段被改：%+v", payload, repo.got)
		}
		var got struct {
			Data struct {
				Alias string `json:"alias"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatalf("payload=%s 响应解析失败: %v", payload, err)
		}
		if got.Data.Alias != "user-1" {
			t.Fatalf("payload=%s 响应 alias = %q, want user-1", payload, got.Data.Alias)
		}
	}
}
