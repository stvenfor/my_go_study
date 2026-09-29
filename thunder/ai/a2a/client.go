// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 a2a
package a2a

// import 多行导入块开始
import (
	// 语句或表达式
	"context"

	// 语句或表达式
	"github.com/cloudwego/eino-ext/a2a/client"
	// 语句或表达式
	"github.com/cloudwego/eino-ext/a2a/models"
	// 语句或表达式
	"github.com/cloudwego/eino-ext/a2a/transport/jsonrpc"
	// 闭合括号（import/var/参数列表等）
)

// 定义函数 GetAgentCard（签名开始）
func GetAgentCard(ctx context.Context, url string, path string) (*models.AgentCard, error) {
	// 短变量声明并赋值
	t, err := jsonrpc.NewTransport(ctx, &jsonrpc.ClientConfig{
		// 列表项（字段/参数/元素），行末逗号
		BaseURL: url,
		// 列表项（字段/参数/元素），行末逗号
		HandlerPath: path,
		// 语句或表达式
	})
	// 条件判断 if
	if err != nil {
		// 返回语句
		return nil, err
		// 语句或表达式
	}
	// 短变量声明并赋值
	aClient, err := client.NewA2AClient(ctx, &client.Config{
		// 列表项（字段/参数/元素），行末逗号
		Transport: t,
		// 语句或表达式
	})
	// 条件判断 if
	if err != nil {
		// 返回语句
		return nil, err
		// 语句或表达式
	}
	// 短变量声明并赋值
	agentCard, err := aClient.AgentCard(ctx)
	// 返回语句
	return agentCard, err
	// 语句或表达式
}
