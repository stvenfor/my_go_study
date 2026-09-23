package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/pkg/config"
)

// LiveAppleReceiptVerifier 调用 Apple verifyReceipt（生产 + sandbox 回退）。
type LiveAppleReceiptVerifier struct {
	cfg    config.AppleIAPConfig
	client *http.Client
}

func NewLiveAppleReceiptVerifier(cfg config.AppleIAPConfig) *LiveAppleReceiptVerifier {
	return &LiveAppleReceiptVerifier{
		cfg: cfg,
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (v *LiveAppleReceiptVerifier) Configured() bool {
	return v != nil && v.cfg.Configured()
}

func (v *LiveAppleReceiptVerifier) Verify(ctx context.Context, in AppleVerifyInput, plan entity.MembershipPlan) (*AppleVerifyResult, error) {
	if !v.Configured() {
		return nil, ErrMembershipAppleNotConfig
	}
	receipt := strings.TrimSpace(in.ReceiptData)
	if receipt == "" {
		return nil, fmt.Errorf("%w: 缺少 receipt_data", ErrMembershipAppleInvalid)
	}
	body, err := v.postVerify(ctx, "https://buy.itunes.apple.com/verifyReceipt", receipt)
	if err != nil {
		return nil, err
	}
	// 21007 = sandbox receipt sent to production
	if body.Status == 21007 {
		body, err = v.postVerify(ctx, "https://sandbox.itunes.apple.com/verifyReceipt", receipt)
		if err != nil {
			return nil, err
		}
	}
	if body.Status != 0 {
		return nil, fmt.Errorf("%w: status=%d", ErrMembershipAppleInvalid, body.Status)
	}
	wantPID := plan.AppleProductID
	txID := strings.TrimSpace(in.TransactionID)
	orig := strings.TrimSpace(in.OriginalTransactionID)
	matched := false
	for _, item := range append(body.LatestReceiptInfo, body.Receipt.InApp...) {
		if item.ProductID != wantPID {
			continue
		}
		if txID != "" && item.TransactionID != txID && item.OriginalTransactionID != txID {
			continue
		}
		matched = true
		if orig == "" {
			orig = item.OriginalTransactionID
		}
		if txID == "" {
			txID = item.TransactionID
		}
		break
	}
	if !matched {
		return nil, fmt.Errorf("%w: receipt 中无匹配商品/交易", ErrMembershipAppleInvalid)
	}
	if orig == "" {
		orig = txID
	}
	return &AppleVerifyResult{
		OriginalTransactionID: orig,
		TransactionID:         txID,
	}, nil
}

type appleVerifyBody struct {
	Status            int                 `json:"status"`
	LatestReceiptInfo []appleReceiptItem  `json:"latest_receipt_info"`
	Receipt           appleReceiptWrapper `json:"receipt"`
}

type appleReceiptWrapper struct {
	InApp []appleReceiptItem `json:"in_app"`
}

type appleReceiptItem struct {
	ProductID             string `json:"product_id"`
	TransactionID         string `json:"transaction_id"`
	OriginalTransactionID string `json:"original_transaction_id"`
}

func (v *LiveAppleReceiptVerifier) postVerify(ctx context.Context, url, receipt string) (*appleVerifyBody, error) {
	payload := map[string]any{
		"receipt-data": receipt,
		"password":     strings.TrimSpace(v.cfg.SharedSecret),
		"exclude-old-transactions": true,
	}
	raw, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMembershipAppleInvalid, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: http %d", ErrMembershipAppleInvalid, resp.StatusCode)
	}
	var out appleVerifyBody
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("%w: parse: %v", ErrMembershipAppleInvalid, err)
	}
	return &out, nil
}
