// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 einos
package einos

// import 多行导入块开始
import (
	// 语句或表达式
	"context"
	// 语句或表达式
	"fmt"

	// 语句或表达式
	mcpp "github.com/cloudwego/eino-ext/components/tool/mcp"
	// 语句或表达式
	"github.com/cloudwego/eino/components/tool"
	// 语句或表达式
	"github.com/cloudwego/eino/schema"
	// 语句或表达式
	"github.com/mark3labs/mcp-go/client"
	// 语句或表达式
	"github.com/mark3labs/mcp-go/client/transport"
	// 语句或表达式
	"github.com/mark3labs/mcp-go/mcp"
	// 闭合括号（import/var/参数列表等）
)

// 定义结构体类型 McpConfig
type McpConfig struct {
	// 语句或表达式
	BaseUrl string
	// 语句或表达式
	Token string
	// 语句或表达式
	Name string
	// 语句或表达式
	Version string
	// 语句或表达式
}

// 定义函数 GetEinoBaseTools（签名开始）
func GetEinoBaseTools(ctx context.Context, config *McpConfig) ([]tool.BaseTool, error) {
	// 短变量声明并赋值
	headers := make(map[string]string)
	// 条件判断 if
	if config.Token != "" {
		// 语句或表达式
		headers["Authorization"] = fmt.Sprintf("Bearer %s", config.Token)
		// 语句或表达式
	}
	// 短变量声明并赋值
	options := transport.WithHeaders(headers)
	// 短变量声明并赋值
	cli, err := client.NewSSEMCPClient(config.BaseUrl, options)
	// 条件判断 if
	if err != nil {
		// 返回语句
		return nil, err
		// 语句或表达式
	}
	// 赋值语句
	err = cli.Start(ctx)
	// 条件判断 if
	if err != nil {
		// 返回语句
		return nil, err
		// 语句或表达式
	}

	// 短变量声明并赋值
	initRequest := mcp.InitializeRequest{}
	// 语句或表达式
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	// 语句或表达式
	initRequest.Params.ClientInfo = mcp.Implementation{
		// 列表项（字段/参数/元素），行末逗号
		Name: config.Name,
		// 列表项（字段/参数/元素），行末逗号
		Version: config.Version,
		// 语句或表达式
	}

	// 语句或表达式
	_, err = cli.Initialize(ctx, initRequest)
	// 条件判断 if
	if err != nil {
		// 返回语句
		return nil, err
		// 语句或表达式
	}
	// 短变量声明并赋值
	tools, err := mcpp.GetTools(ctx, &mcpp.Config{Cli: cli})
	// 条件判断 if
	if err != nil {
		// 返回语句
		return nil, err
		// 语句或表达式
	}

	// 返回语句
	return tools, nil
	// 语句或表达式
}

// 定义函数 GetMCPTool（签名开始）
func GetMCPTool(ctx context.Context, config *McpConfig) ([]mcp.Tool, error) {
	// 短变量声明并赋值
	headers := make(map[string]string)
	// 条件判断 if
	if config.Token != "" {
		// 语句或表达式
		headers["Authorization"] = fmt.Sprintf("Bearer %s", config.Token)
		// 语句或表达式
	}
	// 短变量声明并赋值
	options := transport.WithHeaders(headers)
	// 短变量声明并赋值
	cli, err := client.NewSSEMCPClient(config.BaseUrl, options)
	// 条件判断 if
	if err != nil {
		// 返回语句
		return nil, err
		// 语句或表达式
	}
	// 赋值语句
	err = cli.Start(ctx)
	// 条件判断 if
	if err != nil {
		// 返回语句
		return nil, err
		// 语句或表达式
	}

	// 短变量声明并赋值
	initRequest := mcp.InitializeRequest{}
	// 语句或表达式
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	// 语句或表达式
	initRequest.Params.ClientInfo = mcp.Implementation{
		// 列表项（字段/参数/元素），行末逗号
		Name: config.Name,
		// 列表项（字段/参数/元素），行末逗号
		Version: config.Version,
		// 语句或表达式
	}

	// 语句或表达式
	_, err = cli.Initialize(ctx, initRequest)
	// 条件判断 if
	if err != nil {
		// 返回语句
		return nil, err
		// 语句或表达式
	}
	// 短变量声明并赋值
	tools, err := cli.ListTools(ctx, mcp.ListToolsRequest{})
	// 条件判断 if
	if err != nil {
		// 返回语句
		return nil, err
		// 语句或表达式
	}

	// 返回语句
	return tools.Tools, nil
	// 语句或表达式
}

// 定义函数 GetMCPToolAndCli（签名开始）
func GetMCPToolAndCli(ctx context.Context, config *McpConfig) ([]mcp.Tool, *client.Client, error) {
	// 短变量声明并赋值
	headers := make(map[string]string)
	// 条件判断 if
	if config.Token != "" {
		// 语句或表达式
		headers["Authorization"] = fmt.Sprintf("Bearer %s", config.Token)
		// 语句或表达式
	}
	// 短变量声明并赋值
	options := transport.WithHeaders(headers)
	// 短变量声明并赋值
	cli, err := client.NewSSEMCPClient(config.BaseUrl, options)
	// 条件判断 if
	if err != nil {
		// 返回语句
		return nil, nil, err
		// 语句或表达式
	}
	// 赋值语句
	err = cli.Start(ctx)
	// 条件判断 if
	if err != nil {
		// 返回语句
		return nil, nil, err
		// 语句或表达式
	}

	// 短变量声明并赋值
	initRequest := mcp.InitializeRequest{}
	// 语句或表达式
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	// 语句或表达式
	initRequest.Params.ClientInfo = mcp.Implementation{
		// 列表项（字段/参数/元素），行末逗号
		Name: config.Name,
		// 列表项（字段/参数/元素），行末逗号
		Version: config.Version,
		// 语句或表达式
	}

	// 语句或表达式
	_, err = cli.Initialize(ctx, initRequest)
	// 条件判断 if
	if err != nil {
		// 返回语句
		return nil, nil, err
		// 语句或表达式
	}
	// 短变量声明并赋值
	tools, err := cli.ListTools(ctx, mcp.ListToolsRequest{})
	// 条件判断 if
	if err != nil {
		// 返回语句
		return nil, nil, err
		// 语句或表达式
	}

	// 返回语句
	return tools.Tools, cli, nil
	// 语句或表达式
}

// ConvertSchema converts mcp.ToolInputSchema to a map of ParameterInfo
// 定义函数 ConvertSchema（签名开始）
func ConvertSchema(inputSchema mcp.ToolInputSchema) map[string]*schema.ParameterInfo {
	// 短变量声明并赋值
	params := make(map[string]*schema.ParameterInfo)

	// 遍历schema中的属性
	// for 循环
	for key, value := range inputSchema.Properties {
		// 短变量声明并赋值
		paramInfo := &schema.ParameterInfo{}

		// 解析属性信息
		// 条件判断 if
		if propMap, ok := value.(map[string]any); ok {
			// 处理类型
			// 条件判断 if
			if typeVal, exists := propMap["type"]; exists {
				// 条件判断 if
				if typeStr, ok := typeVal.(string); ok {
					// 需要将MCP类型映射到Eino类型
					// switch 分支
					switch typeStr {
					// case 分支
					case "string":
						// 语句或表达式
						paramInfo.Type = schema.String
					// case 分支
					case "integer":
						// 语句或表达式
						paramInfo.Type = schema.Integer
					// case 分支
					case "number":
						// 语句或表达式
						paramInfo.Type = schema.Number
					// case 分支
					case "boolean":
						// 语句或表达式
						paramInfo.Type = schema.Boolean
					// case 分支
					case "array":
						// 语句或表达式
						paramInfo.Type = schema.Array
					// case 分支
					case "object":
						// 语句或表达式
						paramInfo.Type = schema.Object
					// default 分支
					default:
						// 语句或表达式
						paramInfo.Type = schema.String // 默认为字符串类型
						// 语句或表达式
					}
					// 语句或表达式
				}
				// 语句或表达式
			}

			// 处理描述
			// 条件判断 if
			if descVal, exists := propMap["description"]; exists {
				// 条件判断 if
				if descStr, ok := descVal.(string); ok {
					// 语句或表达式
					paramInfo.Desc = descStr
					// 语句或表达式
				}
				// 语句或表达式
			}

			// 处理枚举值
			// 条件判断 if
			if enumVal, exists := propMap["enum"]; exists {
				// 条件判断 if
				if enumSlice, ok := enumVal.([]interface{}); ok {
					// for 循环
					for _, enumItem := range enumSlice {
						// 条件判断 if
						if enumStr, ok := enumItem.(string); ok {
							// 语句或表达式
							paramInfo.Enum = append(paramInfo.Enum, enumStr)
							// 语句或表达式
						}
						// 语句或表达式
					}
					// 语句或表达式
				}
				// 语句或表达式
			}
			// 语句或表达式
		}

		// 检查是否为必填字段
		// for 循环
		for _, requiredField := range inputSchema.Required {
			// 条件判断 if
			if requiredField == key {
				// 语句或表达式
				paramInfo.Required = true
				// 语句或表达式
				break
				// 语句或表达式
			}
			// 语句或表达式
		}

		// 语句或表达式
		params[key] = paramInfo
		// 语句或表达式
	}

	// 返回语句
	return params
	// 语句或表达式
}
