package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

// ImBackupUsecase 消息业务备份入库。
type ImBackupUsecase struct {
	repo repository.ImBackupRepository
}

func NewImBackupUsecase(repo repository.ImBackupRepository) *ImBackupUsecase {
	return &ImBackupUsecase{repo: repo}
}

// ImBackupEvent 单条备份事件。
type ImBackupEvent struct {
	MessageUID       string         `json:"message_uid"`
	Direction        string         `json:"direction"`
	ConversationType string         `json:"conversation_type"`
	TargetID         string         `json:"target_id"`
	MessageType      string         `json:"message_type"`
	Payload          map[string]any `json:"payload"`
	SentAt           *time.Time     `json:"sent_at"`
}

// ImBackupBatchInput POST /api/v1/im/messages/backup body。
type ImBackupBatchInput struct {
	UserID string          `json:"-"`
	Events []ImBackupEvent `json:"events"`
}

// Ingest 批量写入；按 message_uid 幂等；返回新插入条数。
func (u *ImBackupUsecase) Ingest(ctx context.Context, in ImBackupBatchInput) (int, error) {
	if u == nil || u.repo == nil {
		return 0, fmt.Errorf("im backup usecase not ready")
	}
	userID := strings.TrimSpace(in.UserID)
	if userID == "" {
		return 0, fmt.Errorf("user_id required")
	}
	n := 0
	for _, ev := range in.Events {
		uid := strings.TrimSpace(ev.MessageUID)
		if uid == "" {
			continue
		}
		dir := strings.TrimSpace(ev.Direction)
		switch dir {
		case entity.ImBackupDirectionOut, entity.ImBackupDirectionIn, entity.ImBackupDirectionRecall:
		default:
			return n, fmt.Errorf("invalid direction %q", dir)
		}
		payload := ""
		if ev.Payload != nil {
			b, err := json.Marshal(ev.Payload)
			if err != nil {
				return n, err
			}
			payload = string(b)
		}
		row := &entity.WysImMessageBackup{
			MessageUID:       uid,
			UserID:           userID,
			Direction:        dir,
			ConversationType: strings.TrimSpace(ev.ConversationType),
			TargetID:         strings.TrimSpace(ev.TargetID),
			MessageType:      strings.TrimSpace(ev.MessageType),
			Payload:          payload,
			SentAt:           ev.SentAt,
			CreatedAt:        time.Now().UTC(),
		}
		inserted, err := u.repo.InsertIgnore(ctx, row)
		if err != nil {
			return n, err
		}
		if inserted {
			n++
		}
	}
	return n, nil
}
