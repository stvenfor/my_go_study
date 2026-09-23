package usecase

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

var (
	ErrCashInsufficient     = repository.ErrCashInsufficient
	ErrCashInvalidAmount    = repository.ErrCashInvalidAmount
	ErrCashInvalidChannel   = repository.ErrCashInvalidChannel
	ErrCashCardNotFound     = repository.ErrCashCardNotFound
	ErrCashCardRequired     = repository.ErrCashCardRequired
	ErrCashInvalidCardLast4 = repository.ErrCashInvalidCardLast4
	ErrCashInvalidBankName  = errors.New("银行名称不合法")
)

var cardLast4Re = regexp.MustCompile(`^\d{4}$`)

// CashWalletUsecase 人民币钱包。
type CashWalletUsecase struct {
	repo repository.CashWalletRepository
}

func NewCashWalletUsecase(repo repository.CashWalletRepository) *CashWalletUsecase {
	return &CashWalletUsecase{repo: repo}
}

func (u *CashWalletUsecase) GetSummary(ctx context.Context, userID string) (*entity.WalletSummary, error) {
	fen, err := u.repo.GetBalanceFen(ctx, userID)
	if err != nil {
		return nil, err
	}
	cards, err := u.repo.ListCards(ctx, userID)
	if err != nil {
		return nil, err
	}
	if cards == nil {
		cards = []entity.WysBankCard{}
	}
	return &entity.WalletSummary{
		Balance:    entity.FormatFenToYuan(fen),
		BalanceFen: fen,
		Cards:      cards,
	}, nil
}

func (u *CashWalletUsecase) ListLedger(ctx context.Context, userID string, limit, offset int) (*entity.WalletLedgerPage, error) {
	rows, total, err := u.repo.ListLedger(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []entity.WysCashLedger{}
	}
	return &entity.WalletLedgerPage{Items: rows, Total: total}, nil
}

func (u *CashWalletUsecase) ListCards(ctx context.Context, userID string) ([]entity.WysBankCard, error) {
	cards, err := u.repo.ListCards(ctx, userID)
	if err != nil {
		return nil, err
	}
	if cards == nil {
		return []entity.WysBankCard{}, nil
	}
	return cards, nil
}

type BindCardInput struct {
	BankName   string
	CardLast4  string
	HolderName string
	IsDefault  bool
}

func (u *CashWalletUsecase) BindCard(ctx context.Context, userID string, in BindCardInput) (*entity.WysBankCard, error) {
	bank := strings.TrimSpace(in.BankName)
	if bank == "" || utf8.RuneCountInString(bank) > 64 {
		return nil, ErrCashInvalidBankName
	}
	last4 := strings.TrimSpace(in.CardLast4)
	if !cardLast4Re.MatchString(last4) {
		return nil, ErrCashInvalidCardLast4
	}
	holder := strings.TrimSpace(in.HolderName)
	if utf8.RuneCountInString(holder) > 64 {
		holder = string([]rune(holder)[:64])
	}
	card := &entity.WysBankCard{
		UserID:     userID,
		BankName:   bank,
		CardLast4:  last4,
		HolderName: holder,
		IsDefault:  in.IsDefault,
	}
	if err := u.repo.CreateCard(ctx, card); err != nil {
		return nil, err
	}
	return card, nil
}

func (u *CashWalletUsecase) SetDefaultCard(ctx context.Context, userID string, cardID int64) error {
	return u.repo.SetDefaultCard(ctx, userID, cardID)
}

func (u *CashWalletUsecase) DeleteCard(ctx context.Context, userID string, cardID int64) error {
	return u.repo.DeleteCard(ctx, userID, cardID)
}

type RechargeInput struct {
	Amount  string
	Channel int16
	CardID  *int64
}

func (u *CashWalletUsecase) Recharge(ctx context.Context, userID string, in RechargeInput) (*entity.WalletRechargeResult, error) {
	if !entity.ValidWalletRechargeChannel(in.Channel) {
		return nil, ErrCashInvalidChannel
	}
	fen, err := entity.ParseYuanToFen(in.Amount)
	if err != nil || fen < entity.WalletRechargeMinFen || fen > entity.WalletRechargeMaxFen {
		return nil, ErrCashInvalidAmount
	}
	if in.Channel == entity.WalletRechargeCard {
		if in.CardID == nil || *in.CardID <= 0 {
			return nil, ErrCashCardRequired
		}
		card, err := u.repo.GetCard(ctx, userID, *in.CardID)
		if err != nil {
			return nil, err
		}
		if card == nil {
			return nil, ErrCashCardNotFound
		}
	}
	ref := "alipay"
	switch in.Channel {
	case entity.WalletRechargeWeChat:
		ref = "wechat"
	case entity.WalletRechargeCard:
		ref = "card"
	}
	bal, err := u.repo.Credit(ctx, userID, fen, entity.CashReasonRecharge, ref)
	if err != nil {
		return nil, err
	}
	return &entity.WalletRechargeResult{
		Balance:    entity.FormatFenToYuan(bal),
		BalanceFen: bal,
		DeltaFen:   fen,
	}, nil
}

// Credit / Debit 供商城调用（分）。
func (u *CashWalletUsecase) Credit(ctx context.Context, userID string, deltaFen int64, reason, refID string) (int64, error) {
	return u.repo.Credit(ctx, userID, deltaFen, reason, refID)
}

func (u *CashWalletUsecase) Debit(ctx context.Context, userID string, deltaFen int64, reason, refID string) (int64, error) {
	return u.repo.Debit(ctx, userID, deltaFen, reason, refID)
}

func (u *CashWalletUsecase) GetBalanceFen(ctx context.Context, userID string) (int64, error) {
	return u.repo.GetBalanceFen(ctx, userID)
}
