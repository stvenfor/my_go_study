// es_vector_store.go 手写 Elasticsearch 向量库（无官方 Go client），
// 复用交易全文索引（app/internal/transaction/repo_es.go）的同款 net/http 风格。
// 向量字段为 dense_vector + kNN 检索（需 ES 8.x）。
package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"model/provider"
	"model/repository"
)

// ragVectorDims text-embedding-3-small 维度（与 embedding 模型强绑定）。
const ragVectorDims = 1536

// ESVectorStore 知识块向量索引读写。
type ESVectorStore struct {
	base   string
	index  string
	client *http.Client
}

// NewESVectorStore baseURL 如 http://elasticsearch:9200；index 为向量索引名。
func NewESVectorStore(baseURL, index string) *ESVectorStore {
	return &ESVectorStore{
		base:   strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		index:  strings.TrimSpace(index),
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// EnsureIndex 创建索引与 mapping（已存在则忽略）。
func (s *ESVectorStore) EnsureIndex(ctx context.Context) error {
	body := fmt.Sprintf(`{
  "mappings": {
    "properties": {
      "source_file": { "type": "keyword" },
      "title":       { "type": "text" },
      "content":     { "type": "text" },
      "vector":      { "type": "dense_vector", "dims": %d }
    }
  }
}`, ragVectorDims)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, s.base+"/"+s.index, strings.NewReader(body))
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
	return fmt.Errorf("es ensure rag index: %s: %s", res.Status, truncate(b, 200))
}

// Upsert 写入/覆盖文档（按 ID 幂等）。
func (s *ESVectorStore) Upsert(ctx context.Context, doc repository.RagDoc) error {
	if doc.ID == "" {
		return fmt.Errorf("rag doc id required")
	}
	payload, err := json.Marshal(map[string]any{
		"source_file": doc.SourceFile,
		"title":       doc.Title,
		"content":     doc.Content,
		"vector":      doc.Vector,
	})
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/%s/_doc/%s", s.base, s.index, doc.ID)
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
	return fmt.Errorf("es upsert rag: %s: %s", res.Status, truncate(b, 200))
}

// SearchKNN 按向量近邻召回 topK 块，返回相似度降序。
func (s *ESVectorStore) SearchKNN(ctx context.Context, vec []float64, topK int) ([]provider.RetrievedChunk, error) {
	if topK <= 0 {
		topK = 4
	}
	body, err := json.Marshal(map[string]any{
		"size": topK,
		"knn": map[string]any{
			"field":          "vector",
			"query_vector":   vec,
			"k":              topK,
			"num_candidates": topK * 10,
		},
	})
	if err != nil {
		return nil, err
	}
	url := s.base + "/" + s.index + "/_search"
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
		return nil, fmt.Errorf("es rag search: %s: %s", res.Status, truncate(b, 200))
	}
	var parsed struct {
		Hits struct {
			Hits []struct {
				Score  float64 `json:"_score"`
				Source struct {
					SourceFile string  `json:"source_file"`
					Title      string  `json:"title"`
					Content    string  `json:"content"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(b, &parsed); err != nil {
		return nil, err
	}
	out := make([]provider.RetrievedChunk, 0, len(parsed.Hits.Hits))
	for _, h := range parsed.Hits.Hits {
		out = append(out, provider.RetrievedChunk{
			SourceFile: h.Source.SourceFile,
			Title:      h.Source.Title,
			Content:    h.Source.Content,
			Score:      h.Score,
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
