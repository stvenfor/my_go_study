// embedding_adapter.go embedding 适配：优先复用 thunder 聚焦子包（eino-ext/openai），
// 并提供 app 原生极简回退（POST /embeddings），两者都实现 model/provider.EmbeddingProvider。
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

	teo "github.com/mszlu521/thunder/ai/einos/openai"
	"github.com/cloudwego/eino/components/embedding"
	"model/provider"
)

// NewThunderEmbedding 用 thunder 聚焦子包构造 EmbeddingProvider（主路径）。
func NewThunderEmbedding(ctx context.Context, baseURL, apiKey, model string) (provider.EmbeddingProvider, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("embedding api key required")
	}
	emb, err := teo.LoadOpenAIEmbedding(ctx, teo.OpenAIEmbeddingConfig{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
	})
	if err != nil {
		return nil, err
	}
	return &thunderEmbedder{emb: emb}, nil
}

type thunderEmbedder struct {
	emb embedding.Embedder
}

func (t *thunderEmbedder) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	return t.emb.EmbedStrings(ctx, texts)
}

// NewNativeEmbedding 回退路径：极简 OpenAI 兼容 /embeddings 客户端（零额外依赖）。
func NewNativeEmbedding(baseURL, apiKey, model string) provider.EmbeddingProvider {
	return &nativeEmbedder{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

type nativeEmbedder struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

func (n *nativeEmbedder) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	if strings.TrimSpace(n.apiKey) == "" {
		return nil, fmt.Errorf("embedding api key required")
	}
	payload, err := json.Marshal(map[string]any{
		"model":  n.model,
		"input":  texts,
	})
	if err != nil {
		return nil, err
	}
	url := n.baseURL + "/embeddings"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+n.apiKey)
	res, err := n.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("embeddings http %d: %s", res.StatusCode, truncate(b, 200))
	}
	var parsed struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &parsed); err != nil {
		return nil, err
	}
	out := make([][]float64, 0, len(parsed.Data))
	for _, d := range parsed.Data {
		out = append(out, d.Embedding)
	}
	return out, nil
}
