// Package openai 聚焦子包：仅暴露 OpenAI 兼容的 embedding 构造器，
// 避免引用 thunder/ai/einos 大包（其一次性 import 了 15 个 eino-ext 组件）。
// 该子包只依赖 eino-ext/components/embedding/openai 一个组件，便于与 app 的 eino v0.9.21 对齐。
package openai

import (
	"context"

	oeo "github.com/cloudwego/eino-ext/components/embedding/openai"
	"github.com/cloudwego/eino/components/embedding"
)

// OpenAIEmbeddingConfig OpenAI 兼容 embedding 的最小配置。
type OpenAIEmbeddingConfig struct {
	APIKey  string // 必填
	BaseURL string // OpenAI 或任意 OpenAI 兼容端点（含 /v1）
	Model   string // 如 text-embedding-3-small
}

// LoadOpenAIEmbedding 构造 OpenAI 兼容的 embedding.Embedder。
func LoadOpenAIEmbedding(ctx context.Context, cfg OpenAIEmbeddingConfig) (embedding.Embedder, error) {
	einoCfg := &oeo.EmbeddingConfig{
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
		Model:   cfg.Model,
	}
	return oeo.NewEmbedder(ctx, einoCfg)
}
