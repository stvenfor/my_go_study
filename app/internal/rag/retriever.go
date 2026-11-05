// retriever.go 把用户查询向量化后用 kNN 检索知识块。
package rag

import (
	"context"
	"strings"

	"model/provider"
	"model/repository"
)

// ESRetriever 基于向量库的检索实现。
type ESRetriever struct {
	store repository.VectorRepository
	emb   provider.EmbeddingProvider
	topK  int
}

// NewESRetriever 构造检索器。
func NewESRetriever(store repository.VectorRepository, emb provider.EmbeddingProvider, topK int) *ESRetriever {
	if topK <= 0 {
		topK = 4
	}
	return &ESRetriever{store: store, emb: emb, topK: topK}
}

// Retrieve 向量化查询并召回 topK 块。
func (r *ESRetriever) Retrieve(ctx context.Context, query string, topK int) ([]provider.RetrievedChunk, error) {
	q := ""
	for _, r := range query { // 简单 trim 已由调用方保证；此处做 rune 级去空白
		if r != '\n' && r != '\r' && r != '\t' {
			q += string(r)
		}
	}
	if strings.TrimSpace(q) == "" {
		return nil, nil
	}
	k := r.topK
	if topK > 0 {
		k = topK
	}
	vecs, err := r.emb.Embed(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 || len(vecs[0]) == 0 {
		return nil, nil
	}
	return r.store.SearchKNN(ctx, vecs[0], k)
}
