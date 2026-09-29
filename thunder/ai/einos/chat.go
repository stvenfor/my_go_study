// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 einos
package einos

// import 多行导入块开始
import (
	// 语句或表达式
	"context"

	// 语句或表达式
	"github.com/cloudwego/eino-ext/components/model/ark"
	// 语句或表达式
	"github.com/cloudwego/eino-ext/components/model/claude"
	// 语句或表达式
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	// 语句或表达式
	"github.com/cloudwego/eino-ext/components/model/gemini"
	// 语句或表达式
	"github.com/cloudwego/eino-ext/components/model/ollama"
	// 语句或表达式
	"github.com/cloudwego/eino-ext/components/model/openai"
	// 语句或表达式
	"github.com/cloudwego/eino-ext/components/model/qianfan"
	// 语句或表达式
	"github.com/cloudwego/eino-ext/components/model/qwen"
	// 语句或表达式
	"github.com/cloudwego/eino/components/model"
	// 闭合括号（import/var/参数列表等）
)

// const 常量块开始
const (
	// 赋值语句
	OLLAMA = "ollama"
	// 赋值语句
	OPENAI = "openai"
	// 赋值语句
	CLAUDE = "claude"
	// 赋值语句
	DEEPSEEK = "deepseek"
	// 赋值语句
	GEMINI = "gemini"
	// 赋值语句
	QIANFAN = "qianfan"
	// 赋值语句
	QWEN = "qwen"
	// 赋值语句
	ARK = "ark"

// 闭合括号（import/var/参数列表等）
)

// 定义结构体类型 ChatModelConfig
type ChatModelConfig struct {
	// 语句或表达式
	OllamaConfig *ollama.ChatModelConfig
	// 语句或表达式
	ArkConfig *ark.ChatModelConfig
	// 语句或表达式
	ClaudeConfig *claude.Config
	// 语句或表达式
	OpenaiConfig *openai.ChatModelConfig
	// 语句或表达式
	DeepseekConfig *deepseek.ChatModelConfig
	// 语句或表达式
	QianfanConfig *qianfan.ChatModelConfig
	// 语句或表达式
	QwenConfig *qwen.ChatModelConfig
	// 语句或表达式
	GeminiConfig *gemini.Config
	// 语句或表达式
}

// 定义函数 LoadChatModel（签名开始）
func LoadChatModel(ctx context.Context, chatType string, config *ChatModelConfig) (model.ToolCallingChatModel, error) {
	// 变量声明
	var chatModel model.ToolCallingChatModel
	// 变量声明
	var err error
	// 条件判断 if
	if chatType == OLLAMA {
		// 语句或表达式
		chatModel, err = ollama.NewChatModel(ctx, config.OllamaConfig)
		// 返回语句
		return chatModel, err
		// 语句或表达式
	}
	// 条件判断 if
	if chatType == ARK {
		// 语句或表达式
		chatModel, err = ark.NewChatModel(ctx, config.ArkConfig)
		// 返回语句
		return chatModel, err
		// 语句或表达式
	}
	// 条件判断 if
	if chatType == CLAUDE {
		// 语句或表达式
		chatModel, err = claude.NewChatModel(ctx, config.ClaudeConfig)
		// 返回语句
		return chatModel, err
		// 语句或表达式
	}
	// 条件判断 if
	if chatType == OPENAI {
		// 语句或表达式
		chatModel, err = openai.NewChatModel(ctx, config.OpenaiConfig)
		// 返回语句
		return chatModel, err
		// 语句或表达式
	}
	// 条件判断 if
	if chatType == DEEPSEEK {
		// 语句或表达式
		chatModel, err = deepseek.NewChatModel(ctx, config.DeepseekConfig)
		// 返回语句
		return chatModel, err
		// 语句或表达式
	}
	// 条件判断 if
	if chatType == GEMINI {
		// 语句或表达式
		chatModel, err = gemini.NewChatModel(ctx, config.GeminiConfig)
		// 返回语句
		return chatModel, err
		// 语句或表达式
	}
	// 条件判断 if
	if chatType == QIANFAN {
		// 语句或表达式
		chatModel, err = qianfan.NewChatModel(ctx, config.QianfanConfig)
		// 返回语句
		return chatModel, err
		// 语句或表达式
	}
	// 条件判断 if
	if chatType == QWEN {
		// 语句或表达式
		chatModel, err = qwen.NewChatModel(ctx, config.QwenConfig)
		// 返回语句
		return chatModel, err
		// 语句或表达式
	}
	// 语句或表达式
	chatModel, err = openai.NewChatModel(ctx, config.OpenaiConfig)
	// 返回语句
	return chatModel, err
	// 语句或表达式
}
