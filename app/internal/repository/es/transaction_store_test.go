package es

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"model/entity"
)

func TestSearchFiltersByUserID(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/_search") {
			t.Fatalf("path=%s", r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"hits":{"hits":[{"_source":{"id":1,"user_id":"user-a","type":"expense","category":"餐饮","note":"午饭","amount":12.5,"date":"2026-09-28"}}]}}`))
	}))
	defer srv.Close()

	store := NewTransactionStore(srv.URL)
	items, err := store.Search(context.Background(), "user-a", "午饭", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != 1 {
		t.Fatalf("items=%v", items)
	}
	q, _ := gotBody["query"].(map[string]any)
	boolQ, _ := q["bool"].(map[string]any)
	filter, _ := boolQ["filter"].([]any)
	if len(filter) == 0 {
		t.Fatal("missing user_id filter")
	}
	term := filter[0].(map[string]any)["term"].(map[string]any)
	if term["user_id"] != "user-a" {
		t.Fatalf("filter=%v", term)
	}
}

func TestIndexPayload(t *testing.T) {
	var method, path string
	var doc map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &doc)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	uid := "u1"
	note := "hello"
	store := NewTransactionStore(srv.URL)
	err := store.Index(context.Background(), &entity.Transaction{
		ID: 42, UserID: &uid, Type: "expense", Category: "餐饮", Amount: 9, Date: "2026-01-01", Note: &note,
	})
	if err != nil {
		t.Fatal(err)
	}
	if method != http.MethodPut || !strings.HasSuffix(path, "/transactions/_doc/42") {
		t.Fatalf("%s %s", method, path)
	}
	if doc["user_id"] != "u1" || doc["note"] != "hello" {
		t.Fatalf("doc=%v", doc)
	}
}
