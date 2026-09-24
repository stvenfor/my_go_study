package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
	"github.com/stvenfor/my_go_study/pkg/jpush"
)

// JPushUsecase 设备登记 + 极光下发。
type JPushUsecase struct {
	repo   repository.PushDeviceRepository
	client *jpush.Client
}

func NewJPushUsecase(repo repository.PushDeviceRepository, client *jpush.Client) *JPushUsecase {
	return &JPushUsecase{repo: repo, client: client}
}

// RegisterDeviceInput 客户端上报 RegistrationID。
type RegisterDeviceInput struct {
	UserID         string
	DeviceID       string
	Platform       string
	RegistrationID string
	Alias          string
	Mock           bool
}

func (u *JPushUsecase) RegisterDevice(ctx context.Context, in RegisterDeviceInput) (*entity.WysPushDevice, error) {
	if u == nil || u.repo == nil {
		return nil, fmt.Errorf("jpush usecase not ready")
	}
	alias := strings.TrimSpace(in.Alias)
	if alias == "" {
		alias = strings.TrimSpace(in.UserID)
	}
	return u.repo.Upsert(ctx, entity.WysPushDevice{
		UserID:         strings.TrimSpace(in.UserID),
		DeviceID:       strings.TrimSpace(in.DeviceID),
		Platform:       in.Platform,
		RegistrationID: strings.TrimSpace(in.RegistrationID),
		Alias:          alias,
		Mock:           in.Mock,
	})
}

// SendInput 调试/业务下发推送。
type SendInput struct {
	UserID          string
	Alias           string
	RegistrationIDs []string
	Title           string
	Body            string
	Deeplink        string
	Extras          map[string]any
	Platform        string
}

// SendResult 下发结果。
type SendResult struct {
	Skipped bool   `json:"skipped,omitempty"`
	Reason  string `json:"reason,omitempty"`
	MsgID   string `json:"msg_id,omitempty"`
	SendNo  string `json:"sendno,omitempty"`
}

func (u *JPushUsecase) Send(ctx context.Context, in SendInput) (SendResult, error) {
	if u == nil || u.client == nil {
		return SendResult{Skipped: true, Reason: "jpush client nil"}, nil
	}

	alias := strings.TrimSpace(in.Alias)
	if alias == "" {
		alias = strings.TrimSpace(in.UserID)
	}
	rids := in.RegistrationIDs

	n := jpush.Notification{
		Title:                   strings.TrimSpace(in.Title),
		Alert:                   strings.TrimSpace(in.Body),
		Deeplink:                strings.TrimSpace(in.Deeplink),
		Extras:                  in.Extras,
		Platform:                in.Platform,
		AudienceAlias:           nil,
		AudienceRegistrationIDs: rids,
	}
	if len(rids) == 0 {
		if alias == "" {
			return SendResult{}, fmt.Errorf("alias or registration_id required")
		}
		n.AudienceAlias = []string{alias}
	}

	res, err := u.client.Send(ctx, n)
	if errors.Is(err, jpush.ErrNotConfigured) {
		return SendResult{Skipped: true, Reason: "jpush credentials not configured"}, nil
	}
	if err != nil {
		return SendResult{}, err
	}
	return SendResult{MsgID: res.MsgID, SendNo: res.SendNo}, nil
}

// NotifyUser 按 user_id 别名发一条带 deeplink 的通知（WS 离线兜底）。
func (u *JPushUsecase) NotifyUser(ctx context.Context, userID, title, body, deeplink string, extras map[string]any) (SendResult, error) {
	return u.Send(ctx, SendInput{
		UserID:   userID,
		Alias:    userID,
		Title:    title,
		Body:     body,
		Deeplink: deeplink,
		Extras:   extras,
	})
}
