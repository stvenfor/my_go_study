package repository

import (
	"context"
	"errors"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

var (
	ErrCashInsufficient     = errors.New("余额不足")
	ErrCashInvalidAmount    = errors.New("金额不合法")
	ErrCashInvalidChannel   = errors.New("充值渠道不合法")
	ErrCashCardNotFound     = errors.New("银行卡不存在")
	ErrCashCardRequired     = errors.New("绑卡充值须指定银行卡")
	ErrCashInvalidCardLast4 = errors.New("卡号后四位不合法")
)

// CashWalletRepository 人民币钱包持久化。
type CashWalletRepository interface {
	GetBalanceFen(ctx context.Context, userID string) (int64, error)
	Credit(ctx context.Context, userID string, deltaFen int64, reason, refID string) (int64, error)
	Debit(ctx context.Context, userID string, deltaFen int64, reason, refID string) (int64, error)

	ListLedger(ctx context.Context, userID string, limit, offset int) ([]entity.WysCashLedger, int64, error)

	ListCards(ctx context.Context, userID string) ([]entity.WysBankCard, error)
	GetCard(ctx context.Context, userID string, cardID int64) (*entity.WysBankCard, error)
	CreateCard(ctx context.Context, card *entity.WysBankCard) error
	SetDefaultCard(ctx context.Context, userID string, cardID int64) error
	DeleteCard(ctx context.Context, userID string, cardID int64) error
}
