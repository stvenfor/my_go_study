package repository

import (
	"context"
	"time"
)

// WSTicket WS 连接票据。
type WSTicket struct {
	UserID   string
	ConnID   string
	Platform string
}

// WSTicketRepository 短期 WS ticket 仓储。
type WSTicketRepository interface {
	Save(ctx context.Context, ticket string, data WSTicket, ttl time.Duration) error
	Consume(ctx context.Context, ticket string) (*WSTicket, error)
}
