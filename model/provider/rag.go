// rag.go RAG 相关接口：embedding 与检索。
// 放在 model 层以保持模块边界（app 负责实现，thunder 仅提供 embedding 适配）。
package provider

import "context"

// EmbeddingProvider 把文本批量转为向量。
// 返回顺序与输入一致；向量维度由底层模型固定（如 text-embedding-3-small = 1536）。
type EmbeddingProvider interface {
	// Embed 批量向量化；texts 为空时返回空切片。
	Embed(ctx context.Context, texts []string) ([][]float64, error)
}

// RetrievedChunk 检索命中的知识块。
type RetrievedChunk struct {
	ID         string  // 文档唯一标识（通常为 source_file#idx）
	SourceFile string  // 来源文件
	Title      string  // 块标题（取所属标题或文件名）
	Content    string  // 块正文
	Score      float64 // 相似度得分（ES _score）
}

// Retriever 按查询召回相关知识块。
type Retriever interface {
	// Retrieve 返回 topK 个最相关块；query 为空或不可用返回空。
	Retrieve(ctx context.Context, query string, topK int) ([]RetrievedChunk, error)
}
