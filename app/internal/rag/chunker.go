// chunker.go Markdown 切块：按标题分块，超长再滑动窗口切分。
package rag

import "strings"

// Chunk 知识块。
type Chunk struct {
	Title   string // 所属标题（就近的 #/##/###）
	Content string // 块正文（含标题行）
}

// ChunkMarkdown 切分单个 markdown 文档为若干块。
// chunkSize / chunkOverlap 以 rune（字符）计，对中文友好。
func ChunkMarkdown(content string, chunkSize, chunkOverlap int) []Chunk {
	if chunkSize <= 0 {
		chunkSize = 800
	}
	if chunkOverlap < 0 || chunkOverlap >= chunkSize {
		chunkOverlap = chunkSize / 8
	}

	lines := strings.Split(content, "\n")
	var chunks []Chunk
	var curTitle string
	var buf strings.Builder

	flush := func() {
		text := strings.TrimSpace(buf.String())
		buf.Reset()
		if text == "" {
			return
		}
		for _, piece := range splitLong(text, chunkSize, chunkOverlap) {
			chunks = append(chunks, Chunk{Title: curTitle, Content: piece})
		}
	}

	for _, line := range lines {
		if title := headingTitle(line); title != "" {
			flush()
			curTitle = title
		}
		buf.WriteString(line)
		buf.WriteString("\n")
	}
	flush()
	return chunks
}

// headingTitle 若 line 是 markdown 标题（# ~ ######）返回标题文本，否则返回空。
func headingTitle(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return ""
	}
	i := 0
	for i < len(trimmed) && trimmed[i] == '#' {
		i++
	}
	if i == 0 || i > 6 {
		return ""
	}
	if i < len(trimmed) && trimmed[i] != ' ' && trimmed[i] != '\t' {
		return ""
	}
	return strings.TrimSpace(trimmed[i:])
}

// splitLong 把超长文本按滑动窗口切成 ≤ chunkSize 的片段，重叠 overlap。
func splitLong(text string, chunkSize, overlap int) []string {
	runes := []rune(text)
	if len(runes) <= chunkSize {
		return []string{text}
	}
	step := chunkSize - overlap
	if step <= 0 {
		step = chunkSize
	}
	var out []string
	for start := 0; start < len(runes); start += step {
		end := start + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		out = append(out, string(runes[start:end]))
		if end == len(runes) {
			break
		}
	}
	return out
}
