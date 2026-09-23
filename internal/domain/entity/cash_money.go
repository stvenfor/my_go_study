package entity

import (
	"fmt"
	"math/big"
	"strings"
)

// ParseYuanToFen 将 "12.34" 转为分；最多两位小数。
func ParseYuanToFen(s string) (int64, error) {
	s = strings.TrimSpace(s)
	r, ok := new(big.Rat).SetString(s)
	if !ok || r.Sign() < 0 {
		return 0, fmt.Errorf("invalid money")
	}
	// *100 then require integer fen
	r.Mul(r, big.NewRat(100, 1))
	if !r.IsInt() {
		return 0, fmt.Errorf("invalid money scale")
	}
	return r.Num().Int64(), nil
}

// FormatFenToYuan 分转 "12.34"。
func FormatFenToYuan(fen int64) string {
	neg := fen < 0
	if neg {
		fen = -fen
	}
	yuan := fen / 100
	cent := fen % 100
	s := fmt.Sprintf("%d.%02d", yuan, cent)
	if neg {
		return "-" + s
	}
	return s
}

// ValidWalletRechargeChannel 本地模拟充值渠道。
func ValidWalletRechargeChannel(ch int16) bool {
	switch ch {
	case WalletRechargeAlipay, WalletRechargeWeChat, WalletRechargeCard:
		return true
	default:
		return false
	}
}
