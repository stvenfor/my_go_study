package repository

import (
	"context"
	"errors"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

// ErrConversationNotFound 会话不存在或已过期（客户端可开新会话）。
var ErrConversationNotFound = errors.New("conversation not found or expired")

// ConversationRepository 停留会话（Visit Conversation）仓储。
type ConversationRepository interface {
	// Create 创建空会话并返回 conversationId。
	Create(ctx context.Context, userID string, ttl time.Duration) (conversationID string, err error)
	// Get 读取会话轮次；不存在返回 ErrConversationNotFound。
	Get(ctx context.Context, userID, conversationID string) ([]entity.ChatTurn, error)
	// SaveTurns 覆写轮次并滑动续期；maxTurns 为最大「轮」数（user+assistant 计为一轮时由调用方裁剪）。
	SaveTurns(ctx context.Context, userID, conversationID string, turns []entity.ChatTurn, ttl time.Duration) error
	// Touch 仅滑动续期。
	Touch(ctx context.Context, userID, conversationID string, ttl time.Duration) error
}
