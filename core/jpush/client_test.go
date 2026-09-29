package jpush

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConfigConfigured(t *testing.T) {
	if (Config{AppKey: "DEV_JPUSH_APP_KEY_PLACEHOLDER", MasterSecret: "x"}).Configured() {
		t.Fatal("placeholder app key must not count as configured")
	}
	if !(Config{AppKey: "a1b2c3d4e5f6", MasterSecret: "secret"}).Configured() {
		t.Fatal("real key should be configured")
	}
}

func TestClientSendBuildsAudienceAndExtras(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			t.Error("missing Authorization")
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"msg_id":"1001","sendno":"0"}`))
	}))
	defer srv.Close()

	c := NewClient(Config{
		AppKey:       "testkey",
		MasterSecret: "testsecret",
		PushURL:      srv.URL,
		HTTPClient:   srv.Client(),
	})
	res, err := c.Send(context.Background(), Notification{
		Title:         "t",
		Alert:         "hello",
		Deeplink:      "xiaomao://app/community",
		AudienceAlias: []string{"user_1"},
		Extras:        map[string]any{"foo": "bar"},
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if res.MsgID != "1001" {
		t.Fatalf("msg_id=%q", res.MsgID)
	}
	aud, _ := gotBody["audience"].(map[string]any)
	aliases, _ := aud["alias"].([]any)
	if len(aliases) != 1 || aliases[0] != "user_1" {
		t.Fatalf("audience alias=%v", aliases)
	}
	notif, _ := gotBody["notification"].(map[string]any)
	android, _ := notif["android"].(map[string]any)
	extras, _ := android["extras"].(map[string]any)
	if extras["deeplink"] != "xiaomao://app/community" {
		t.Fatalf("deeplink extras=%v", extras)
	}
	if extras["foo"] != "bar" {
		t.Fatalf("custom extras=%v", extras)
	}
}

func TestClientSendNotConfigured(t *testing.T) {
	c := NewClient(Config{})
	_, err := c.Send(context.Background(), Notification{Alert: "x", AudienceAlias: []string{"a"}})
	if err != ErrNotConfigured {
		t.Fatalf("want ErrNotConfigured, got %v", err)
	}
}
