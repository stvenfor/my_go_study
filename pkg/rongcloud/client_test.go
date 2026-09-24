package rongcloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConfig_Configured(t *testing.T) {
	if (Config{}).Configured() {
		t.Fatal("empty should not be configured")
	}
	if (Config{AppKey: "PLACEHOLDER_KEY", AppSecret: "secret"}).Configured() {
		t.Fatal("placeholder key should not count")
	}
	if !(Config{AppKey: "realkey", AppSecret: "secret"}).Configured() {
		t.Fatal("want configured")
	}
}

func TestClient_GetToken_NotConfigured(t *testing.T) {
	c := NewClient(Config{})
	_, err := c.GetToken(context.Background(), "u1", "n", "")
	if err != ErrNotConfigured {
		t.Fatalf("want ErrNotConfigured, got %v", err)
	}
}

func TestClient_GetToken_OK(t *testing.T) {
	var sawAuth bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user/getToken.json" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.Header.Get("App-Key") == "" || r.Header.Get("Signature") == "" {
			t.Fatal("missing auth headers")
		}
		sawAuth = true
		_ = r.ParseForm()
		if r.Form.Get("userId") != "user-uuid-1" {
			t.Fatalf("userId %q", r.Form.Get("userId"))
		}
		if r.Form.Get("portraitUri") != "https://cdn.example/a.png" {
			t.Fatalf("portraitUri %q", r.Form.Get("portraitUri"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 200, "token": "tok_abc", "userId": "user-uuid-1",
		})
	}))
	defer srv.Close()

	c := NewClient(Config{
		AppKey:     "appkey",
		AppSecret:  "secret",
		APIBaseURL: srv.URL,
		HTTPClient: srv.Client(),
	})
	out, err := c.GetToken(context.Background(), "user-uuid-1", "Alice", "https://cdn.example/a.png")
	if err != nil {
		t.Fatal(err)
	}
	if !sawAuth || out.Token != "tok_abc" || out.UserID != "user-uuid-1" {
		t.Fatalf("got %+v sawAuth=%v", out, sawAuth)
	}
}

func TestClient_GetToken_SkipsDataPortrait(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("portraitUri") != "" {
			t.Fatalf("data: portrait should be skipped, got %q", r.Form.Get("portraitUri"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 200, "token": "tok", "userId": "u",
		})
	}))
	defer srv.Close()
	c := NewClient(Config{AppKey: "k", AppSecret: "s", APIBaseURL: srv.URL, HTTPClient: srv.Client()})
	if _, err := c.GetToken(context.Background(), "u", "n", "data:image/png;base64,xx"); err != nil {
		t.Fatal(err)
	}
}

func TestClient_GetToken_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 1002, "token": ""})
	}))
	defer srv.Close()
	c := NewClient(Config{AppKey: "k", AppSecret: "s", APIBaseURL: srv.URL, HTTPClient: srv.Client()})
	_, err := c.GetToken(context.Background(), "u", "n", "")
	if err == nil || !strings.Contains(err.Error(), "code=1002") {
		t.Fatalf("want code error, got %v", err)
	}
}
