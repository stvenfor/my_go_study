// vector_repository.go 向量存储接口（RAG 文档向量落 Elasticsearch）。
// 放在 model 层以保持模块边界；app 负责实现手写 ES 客户端。
package repository

import (
	"context"

	"model/provider"
)

// RagDoc 一条待索引的知识块文档。
type RagDoc struct {
	ID         string    // 唯一标识（source_file#idx），用于幂等 upsert
	SourceFile string    // 来源文件
	Title      string    // 块标题
	Content    string    // 块正文
	Vector     []float64 // 向量（维度需与索引 mapping 一致）
}

// VectorRepository 向量索引读写。
type VectorRepository interface {
	// EnsureIndex 创建索引与 mapping（已存在忽略）。
	EnsureIndex(ctx context.Context) error
	// Upsert 写入/覆盖文档（按 ID 幂等）。
	Upsert(ctx context.Context, doc RagDoc) error
	// SearchKNN 按向量近邻召回 topK 块。
	SearchKNN(ctx context.Context, vec []float64, topK int) ([]provider.RetrievedChunk, error)
}
