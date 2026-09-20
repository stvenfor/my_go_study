// mock_stream_provider.go 一期 Mock：App/4S 业务向导风格关键词回复 + 慢速吐字。
package llm

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/provider"
)

const mockModelName = "mock-v1"

// MockStreamProvider 本地流式 Mock。
type MockStreamProvider struct {
	// ChunkDelay 每段增量间隔；测试可设为 0。
	ChunkDelay time.Duration
}

// NewMockStreamProvider 创建 Mock Provider。
func NewMockStreamProvider() *MockStreamProvider {
	return &MockStreamProvider{ChunkDelay: 40 * time.Millisecond}
}

// Name 实现 StreamProvider。
func (p *MockStreamProvider) Name() string {
	return mockModelName
}

// Stream 按 rune 慢速输出；尊重 ctx 取消。
func (p *MockStreamProvider) Stream(ctx context.Context, in provider.CompletionInput, out chan<- string) (string, error) {
	answer := mockAnswer(in)
	if in.MaxTokens > 0 {
		runes := []rune(answer)
		if len(runes) > in.MaxTokens {
			answer = string(runes[:in.MaxTokens])
			return p.emit(ctx, answer, out, entity.FinishReasonLength)
		}
	}
	return p.emit(ctx, answer, out, entity.FinishReasonStop)
}

func (p *MockStreamProvider) emit(ctx context.Context, answer string, out chan<- string, reason string) (string, error) {
	delay := p.ChunkDelay
	for _, r := range answer {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		chunk := string(r)
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case out <- chunk:
		}
		if delay > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(delay):
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return reason, nil
}

func mockAnswer(in provider.CompletionInput) string {
	prompt := strings.TrimSpace(in.Prompt)
	lower := strings.ToLower(prompt)

	// 多轮：若上下文已有助手回复且用户追问，给简短续答
	if len(in.Messages) > 0 {
		if strings.Contains(prompt, "然后") || strings.Contains(prompt, "还有") ||
			strings.Contains(prompt, "呢") || strings.Contains(lower, "then") {
			return "好的，结合刚才的说明：你可以继续在当前页操作，或从首页宫格重新进入对应功能。若仍找不到入口，告诉我具体功能名，我再指路。"
		}
	}

	switch {
	case containsAny(prompt, "二手车", "交易", "transaction"):
		return "【AI小石头】二手车入口：首页 →「全部服务」或宫格里的二手车/交易相关入口。进入后可浏览列表；写操作需先登录。未登录时会先跳转登录页，成功后再回到目标页。"
	case containsAny(prompt, "登录", "注册", "账号", "login"):
		return "【AI小石头】登录说明：使用完整邮箱作为用户名，密码至少 6 位。登录态由 Go BFF 校验；过期或未登录时受保护功能会引导重新登录。设置页可查看账号与退出登录。"
	case containsAny(prompt, "数据", "分析", "报表", "analytics"):
		return "【AI小石头】数据分析：首页相关入口可查看图表与报表。数据经 Go BFF 拉取；若页面空白，请确认已登录且后端可用，再下拉刷新重试。"
	case containsAny(prompt, "短视频", "配音", "视频"):
		return "【AI小石头】视频能力：短视频与配音在视频模块。播放页为沉浸式全屏；离开播放页会恢复系统栏。列表页可进入详情或作品页继续浏览。"
	case containsAny(prompt, "会员", "支付", "续费"):
		return "【AI小石头】会员与支付：在「我的」或支付相关入口查看套餐与续费。支付能力走独立支付模块；具体渠道以页面展示为准。"
	case containsAny(prompt, "你好", "介绍", "你是谁", "小石头"):
		return "你好，我是 AI小石头——本 App / 4S 场景的业务向导。你可以问我：二手车入口、登录注册、数据分析、短视频配音、会员支付等怎么用。我会用纯文本逐步说明。"
	default:
		if utf8.RuneCountInString(prompt) == 0 {
			return "请告诉我你想了解的功能，例如「二手车入口在哪里」或「怎么登录」。"
		}
		return "我是 AI小石头，专注本 App / 4S 业务指引。关于「" + truncate(prompt, 40) + "」：建议从首页宫格或「全部服务」查找对应入口；需要登录的功能会先引导登录。也可以换个更具体的功能名再问我。"
	}
}

func containsAny(s string, keywords ...string) bool {
	for _, k := range keywords {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
}

func truncate(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "…"
}

var _ provider.StreamProvider = (*MockStreamProvider)(nil)
