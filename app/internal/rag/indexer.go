// indexer.go 把本地 KB 目录下的 markdown 文档切分、向量化并写入向量库。
package rag

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"model/provider"
	"model/repository"
)

// IndexKB 读取 kbDir 下所有 *.md，切块 → 向量化 → 幂等写入 store。
// 返回索引的文档总数（块数）。
func IndexKB(ctx context.Context, kbDir string, emb provider.EmbeddingProvider, store repository.VectorRepository, chunkSize, overlap int) (int, error) {
	matches, err := filepath.Glob(filepath.Join(kbDir, "*.md"))
	if err != nil {
		return 0, err
	}
	if len(matches) == 0 {
		return 0, fmt.Errorf("kb dir %q has no *.md files", kbDir)
	}
	total := 0
	for _, file := range matches {
		data, err := os.ReadFile(file)
		if err != nil {
			return total, fmt.Errorf("read %s: %w", file, err)
		}
		chunks := ChunkMarkdown(string(data), chunkSize, overlap)
		if len(chunks) == 0 {
			continue
		}
		texts := make([]string, len(chunks))
		for i, c := range chunks {
			texts[i] = c.Content
		}
		vectors, err := emb.Embed(ctx, texts)
		if err != nil {
			return total, fmt.Errorf("embed %s: %w", file, err)
		}
		for i, c := range chunks {
			doc := repository.RagDoc{
				ID:         fmt.Sprintf("%s#%d", filepath.Base(file), i),
				SourceFile: filepath.Base(file),
				Title:      c.Title,
				Content:    c.Content,
			}
			if i < len(vectors) {
				doc.Vector = vectors[i]
			}
			if err := store.Upsert(ctx, doc); err != nil {
				return total, fmt.Errorf("upsert %s#%d: %w", file, i, err)
			}
			total++
		}
	}
	return total, nil
}
