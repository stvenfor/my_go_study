// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 httputils
package httputils

// import 多行导入块开始
import (
	// 语句或表达式
	"bytes"
	// 语句或表达式
	"encoding/json"
	// 语句或表达式
	"fmt"
	// 语句或表达式
	"io"
	// 语句或表达式
	"net/http"
	// 语句或表达式
	"time"
	// 闭合括号（import/var/参数列表等）
)

// HTTPClient 定义一个 HTTP 客户端工具类
// 定义结构体类型 HTTPClient
type HTTPClient struct {
	// 语句或表达式
	client *http.Client
	// 语句或表达式
}

// NewHTTPClient 创建一个新的 HTTP 客户端
// 定义函数 NewHTTPClient（签名开始）
func NewHTTPClient(timeout time.Duration) *HTTPClient {
	// 返回语句
	return &HTTPClient{
		// 列表项（字段/参数/元素），行末逗号
		client: &http.Client{Timeout: timeout},
		// 语句或表达式
	}
	// 语句或表达式
}

// GET 发送 GET 请求
// 定义函数 GET（签名开始）
func (h *HTTPClient) GET(url string, headers map[string]string) (string, error) {
	// 创建请求
	// 短变量声明并赋值
	req, err := http.NewRequest(http.MethodGet, url, nil)
	// 条件判断 if
	if err != nil {
		// 返回语句
		return "", fmt.Errorf("创建 GET 请求失败: %w", err)
		// 语句或表达式
	}

	// 设置请求头
	// for 循环
	for key, value := range headers {
		// 语句或表达式
		req.Header.Set(key, value)
		// 语句或表达式
	}

	// 执行请求
	// 短变量声明并赋值
	resp, err := h.client.Do(req)
	// 条件判断 if
	if err != nil {
		// 返回语句
		return "", fmt.Errorf("发送 GET 请求失败: %w", err)
		// 语句或表达式
	}
	// defer 延迟执行
	defer resp.Body.Close()

	// 读取响应体
	// 短变量声明并赋值
	body, err := io.ReadAll(resp.Body)
	// 条件判断 if
	if err != nil {
		// 返回语句
		return "", fmt.Errorf("读取响应体失败: %w", err)
		// 语句或表达式
	}

	// 返回语句
	return string(body), nil
	// 语句或表达式
}

// POST 发送 POST 请求（支持 JSON 数据）
// 定义函数 POST（签名开始）
func (h *HTTPClient) POST(url string, headers map[string]string, data interface{}) (string, error) {
	// 将数据编码为 JSON
	// 短变量声明并赋值
	jsonData, err := json.Marshal(data)
	// 条件判断 if
	if err != nil {
		// 返回语句
		return "", fmt.Errorf("JSON 编码失败: %w", err)
		// 语句或表达式
	}

	// 创建请求
	// 短变量声明并赋值
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	// 条件判断 if
	if err != nil {
		// 返回语句
		return "", fmt.Errorf("创建 POST 请求失败: %w", err)
		// 语句或表达式
	}

	// 设置请求头
	// for 循环
	for key, value := range headers {
		// 语句或表达式
		req.Header.Set(key, value)
		// 语句或表达式
	}
	// 语句或表达式
	req.Header.Set("Content-Type", "application/json")

	// 执行请求
	// 短变量声明并赋值
	resp, err := h.client.Do(req)
	// 条件判断 if
	if err != nil {
		// 返回语句
		return "", fmt.Errorf("发送 POST 请求失败: %w", err)
		// 语句或表达式
	}
	// defer 延迟执行
	defer resp.Body.Close()

	// 读取响应体
	// 短变量声明并赋值
	body, err := io.ReadAll(resp.Body)
	// 条件判断 if
	if err != nil {
		// 返回语句
		return "", fmt.Errorf("读取响应体失败: %w", err)
		// 语句或表达式
	}

	// 返回语句
	return string(body), nil
	// 语句或表达式
}
