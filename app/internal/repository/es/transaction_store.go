// Package es 用最小 HTTP REST 访问 Elasticsearch（不引入官方 Go client）。
package es

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"model/entity"
)

const TransactionsIndex = "transactions"

// TransactionStore 交易索引读写（REST）。
type TransactionStore struct {
	base   string
	client *http.Client
}

// NewTransactionStore baseURL 如 http://elasticsearch:9200（无尾斜杠亦可）。
func NewTransactionStore(baseURL string) *TransactionStore {
	return &TransactionStore{
		base: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// EnsureIndex 创建索引与 mapping（已存在则忽略）。
func (s *TransactionStore) EnsureIndex(ctx context.Context) error {
	body := `{
  "mappings": {
    "properties": {
      "id": { "type": "long" },
      "user_id": { "type": "keyword" },
      "type": { "type": "text", "fields": { "keyword": { "type": "keyword" } } },
      "category": { "type": "text", "fields": { "keyword": { "type": "keyword" } } },
      "note": { "type": "text" },
      "amount": { "type": "double" },
      "date": { "type": "keyword" }
    }
  }
}`
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, s.base+"/"+TransactionsIndex, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusCreated {
		return nil
	}
	b, _ := io.ReadAll(res.Body)
	// resource_already_exists_exception
	if res.StatusCode == http.StatusBadRequest && bytes.Contains(b, []byte("resource_already_exists")) {
		return nil
	}
	return fmt.Errorf("es ensure index: %s: %s", res.Status, truncate(b, 200))
}

// Index 写入/覆盖文档；id 用交易主键字符串。
func (s *TransactionStore) Index(ctx context.Context, t *entity.Transaction) error {
	if t == nil {
		return nil
	}
	userID := ""
	if t.UserID != nil {
		userID = *t.UserID
	}
	note := ""
	if t.Note != nil {
		note = *t.Note
	}
	doc := map[string]any{
		"id":       t.ID,
		"user_id":  userID,
		"type":     t.Type,
		"category": t.Category,
		"note":     note,
		"amount":   t.Amount,
		"date":     t.Date,
	}
	payload, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/%s/_doc/%d", s.base, TransactionsIndex, t.ID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return nil
	}
	b, _ := io.ReadAll(res.Body)
	return fmt.Errorf("es index: %s: %s", res.Status, truncate(b, 200))
}

// Delete 按交易 id 删文档；404 视为成功。
func (s *TransactionStore) Delete(ctx context.Context, id int64) error {
	url := fmt.Sprintf("%s/%s/_doc/%d", s.base, TransactionsIndex, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound || (res.StatusCode >= 200 && res.StatusCode < 300) {
		return nil
	}
	b, _ := io.ReadAll(res.Body)
	return fmt.Errorf("es delete: %s: %s", res.Status, truncate(b, 200))
}

// Search 仅过滤 session user_id；全文 note/category/type。
func (s *TransactionStore) Search(ctx context.Context, userID, q string, limit int) ([]entity.Transaction, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, fmt.Errorf("user_id required")
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	q = strings.TrimSpace(q)
	var query any
	if q == "" {
		query = map[string]any{
			"bool": map[string]any{
				"filter": []any{map[string]any{"term": map[string]any{"user_id": userID}}},
			},
		}
	} else {
		query = map[string]any{
			"bool": map[string]any{
				"filter": []any{map[string]any{"term": map[string]any{"user_id": userID}}},
				"must": []any{
					map[string]any{
						"multi_match": map[string]any{
							"query":  q,
							"fields": []string{"note", "category", "type"},
						},
					},
				},
			},
		}
	}
	body, err := json.Marshal(map[string]any{
		"size":  limit,
		"query": query,
		"sort":  []any{map[string]any{"id": "desc"}},
	})
	if err != nil {
		return nil, err
	}
	url := s.base + "/" + TransactionsIndex + "/_search"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("es search: %s: %s", res.Status, truncate(b, 200))
	}
	var parsed struct {
		Hits struct {
			Hits []struct {
				Source struct {
					ID       int64   `json:"id"`
					UserID   string  `json:"user_id"`
					Type     string  `json:"type"`
					Category string  `json:"category"`
					Note     string  `json:"note"`
					Amount   float64 `json:"amount"`
					Date     string  `json:"date"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(b, &parsed); err != nil {
		return nil, err
	}
	out := make([]entity.Transaction, 0, len(parsed.Hits.Hits))
	for _, h := range parsed.Hits.Hits {
		uid := h.Source.UserID
		var note *string
		if h.Source.Note != "" {
			n := h.Source.Note
			note = &n
		}
		out = append(out, entity.Transaction{
			ID:       h.Source.ID,
			UserID:   &uid,
			Type:     h.Source.Type,
			Category: h.Source.Category,
			Amount:   h.Source.Amount,
			Date:     h.Source.Date,
			Note:     note,
		})
	}
	return out, nil
}

func truncate(b []byte, n int) string {
	s := string(b)
	if len(s) <= n {
		return s
	}
	return s[:n]
}
