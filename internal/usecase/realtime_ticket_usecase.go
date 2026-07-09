// =============================================================================
// 文件：realtime_ticket_usecase.go
// 层级：Usecase（业务用例）—— 「换票」领域逻辑，不直接碰 HTTP/Redis 细节
//
// 【初学者】Clean Architecture 分层：
//   Controller 解析 HTTP → Usecase 执行业务 → Repository 读写 Redis
//   这样换票规则变更时，只改 Usecase，不动 Handler。
// =============================================================================
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
	"github.com/stvenfor/my_go_study/pkg/config"
)

type RealtimeTicketInput struct {
	UserID   string // 来自 Supabase JWT 中间件解析出的用户 UUID
	Platform string // mobile / web，便于统计
	ConnID   string // 可选，客户端不传则服务端生成
}

type RealtimeTicketOutput struct {
	Ticket           string
	WSURL            string
	ExpiresInSeconds int
	ConnID           string
}

type RealtimeTicketUsecase struct {
	tickets repository.WSTicketRepository
	cfg     config.Config
}

func NewRealtimeTicketUsecase(tickets repository.WSTicketRepository, cfg config.Config) *RealtimeTicketUsecase {
	return &RealtimeTicketUsecase{tickets: tickets, cfg: cfg}
}

// Issue 签发短期 ticket：Flutter 登录后调用 HTTP ws-ticket 时执行。
func (u *RealtimeTicketUsecase) Issue(ctx context.Context, input RealtimeTicketInput) (*RealtimeTicketOutput, error) {
	if input.UserID == "" {
		return nil, fmt.Errorf("userId 不能为空")
	}

	connID := input.ConnID
	if connID == "" {
		// 用毫秒时间戳保证唯一，便于日志关联
		connID = fmt.Sprintf("conn_%d", time.Now().UnixMilli())
	}

	ticket := uuid.NewString() // 随机 UUID，不可猜测
	ttl := u.cfg.Realtime.TicketTTL()

	// 写入 Redis，过期自动删除
	if err := u.tickets.Save(ctx, ticket, repository.WSTicket{
		UserID:   input.UserID,
		ConnID:   connID,
		Platform: input.Platform,
	}, ttl); err != nil {
		return nil, err
	}

	return &RealtimeTicketOutput{
		Ticket:           ticket,
		WSURL:            u.cfg.Realtime.WSURL(u.cfg.Server.Port),
		ExpiresInSeconds: int(ttl.Seconds()),
		ConnID:           connID,
	}, nil
}

// Consume WS auth 时调用：校验 ticket 并删除（一次性）。
func (u *RealtimeTicketUsecase) Consume(ctx context.Context, ticket string) (*repository.WSTicket, error) {
	return u.tickets.Consume(ctx, ticket)
}
