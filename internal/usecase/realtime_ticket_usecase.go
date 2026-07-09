package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
	"github.com/stvenfor/my_go_study/pkg/config"
)

// RealtimeTicketInput 换票请求。
type RealtimeTicketInput struct {
	UserID   string
	Platform string
	ConnID   string
}

// RealtimeTicketOutput 换票响应。
type RealtimeTicketOutput struct {
	Ticket           string
	WSURL            string
	ExpiresInSeconds int
	ConnID           string
}

// RealtimeTicketUsecase WS ticket 用例。
type RealtimeTicketUsecase struct {
	tickets repository.WSTicketRepository
	cfg     config.Config
}

// NewRealtimeTicketUsecase 创建 ticket 用例。
func NewRealtimeTicketUsecase(tickets repository.WSTicketRepository, cfg config.Config) *RealtimeTicketUsecase {
	return &RealtimeTicketUsecase{tickets: tickets, cfg: cfg}
}

// Issue 签发短期 WS ticket。
func (u *RealtimeTicketUsecase) Issue(ctx context.Context, input RealtimeTicketInput) (*RealtimeTicketOutput, error) {
	if input.UserID == "" {
		return nil, fmt.Errorf("userId 不能为空")
	}
	connID := input.ConnID
	if connID == "" {
		connID = fmt.Sprintf("conn_%d", time.Now().UnixMilli())
	}
	ticket := uuid.NewString()
	ttl := u.cfg.Realtime.TicketTTL()
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

// Consume 校验并消费 ticket（WS auth）。
func (u *RealtimeTicketUsecase) Consume(ctx context.Context, ticket string) (*repository.WSTicket, error) {
	return u.tickets.Consume(ctx, ticket)
}
