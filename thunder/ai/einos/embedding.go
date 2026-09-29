// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 einos
package einos

// import 多行导入块开始
import (
	// 语句或表达式
	"context"

	// 语句或表达式
	"github.com/cloudwego/eino-ext/components/embedding/ark"
	// 语句或表达式
	"github.com/cloudwego/eino-ext/components/embedding/dashscope"
	// 语句或表达式
	"github.com/cloudwego/eino-ext/components/embedding/gemini"
	// 语句或表达式
	"github.com/cloudwego/eino-ext/components/embedding/ollama"
	// 语句或表达式
	"github.com/cloudwego/eino-ext/components/embedding/openai"
	// 语句或表达式
	"github.com/cloudwego/eino-ext/components/embedding/qianfan"
	// 语句或表达式
	"github.com/cloudwego/eino-ext/components/embedding/tencentcloud"
	// 语句或表达式
	"github.com/cloudwego/eino/components/embedding"
	// 闭合括号（import/var/参数列表等）
)

// const 常量块开始
const (
	// 赋值语句
	EmbeddingOllama = "ollama"
	// 赋值语句
	EmbeddingDashscope = "dashscope"
	// 赋值语句
	EmbeddingOpenai = "openai"
	// 赋值语句
	EmbeddingArk = "ark"
	// 赋值语句
	EmbeddingQianfan = "qianfan"
	// 赋值语句
	EmbeddingTencentCloud = "tencentcloud"
	// 赋值语句
	EmbeddingGemini = "gemini"

	// 闭合括号（import/var/参数列表等）
)

// 定义结构体类型 EmbeddingModelConfig
type EmbeddingModelConfig struct {
	// 语句或表达式
	OllamaConfig *ollama.EmbeddingConfig
	// 语句或表达式
	DashscopeConfig *dashscope.EmbeddingConfig
	// 语句或表达式
	ArkConfig *ark.EmbeddingConfig
	// 语句或表达式
	QianfanConfig *qianfan.EmbeddingConfig
	// 语句或表达式
	TencentCloudConfig *tencentcloud.EmbeddingConfig
	// 语句或表达式
	OpenaiConfig *openai.EmbeddingConfig
	// 语句或表达式
	GeminiConfig *gemini.EmbeddingConfig
	// 语句或表达式
}

// 定义函数 LoadEmbedding（签名开始）
func LoadEmbedding(ctx context.Context, embeddingType string, config *EmbeddingModelConfig) (embedding.Embedder, error) {
	// 不同的提供商返回不同的配置
	var embedder embedding.Embedder
	// 变量声明
	var err error
	// switch 分支
	switch embeddingType {
	// case 分支
	case EmbeddingOllama:
		// 语句或表达式
		embedder, err = ollama.NewEmbedder(ctx, config.OllamaConfig)
	// case 分支
	case EmbeddingDashscope:
		// 语句或表达式
		embedder, err = dashscope.NewEmbedder(ctx, config.DashscopeConfig)
	// case 分支
	case EmbeddingArk:
		// 语句或表达式
		embedder, err = ark.NewEmbedder(ctx, config.ArkConfig)
	// case 分支
	case EmbeddingQianfan:
		// 语句或表达式
		embedder, err = qianfan.NewEmbedder(ctx, config.QianfanConfig)
	// case 分支
	case EmbeddingTencentCloud:
		// 语句或表达式
		embedder, err = tencentcloud.NewEmbedder(ctx, config.TencentCloudConfig)
	// case 分支
	case EmbeddingOpenai:
		// 语句或表达式
		embedder, err = openai.NewEmbedder(ctx, config.OpenaiConfig)
	// case 分支
	case EmbeddingGemini:
		// 语句或表达式
		embedder, err = gemini.NewEmbedder(ctx, config.GeminiConfig)
	// default 分支
	default:
		// 语句或表达式
		embedder, err = openai.NewEmbedder(ctx, config.OpenaiConfig)
		// 语句或表达式
	}
	// 返回语句
	return embedder, err
	// 语句或表达式
}
