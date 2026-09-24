package usecase

import (
	"context"
	"errors"
	"strings"

	"github.com/stvenfor/my_go_study/internal/domain/repository"
	"github.com/stvenfor/my_go_study/pkg/rongcloud"
)

// RongCloudTokenClient 签发 IM Token（可替换为假实现）。
type RongCloudTokenClient interface {
	GetToken(ctx context.Context, userID, name, portraitURI string) (rongcloud.TokenResult, error)
}

// ImSessionUsecase 融云 IM 连接凭证。
type ImSessionUsecase struct {
	client RongCloudTokenClient
	users  repository.ImUserLookup
}

func NewImSessionUsecase(client RongCloudTokenClient, users repository.ImUserLookup) *ImSessionUsecase {
	return &ImSessionUsecase{client: client, users: users}
}

// ImSessionOutput POST /api/v1/im/session 响应。
type ImSessionOutput struct {
	UserID           string `json:"user_id"`
	Token            string `json:"token"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

// ErrImNotConfigured 服务端未配置融云。
var ErrImNotConfigured = errors.New("im not configured")

// ErrImUserIDRequired 业务用户 ID 为空。
var ErrImUserIDRequired = errors.New("im user id required")

// IssueSession 为已登录用户签发 IM Token；融云 userId = 业务 UUID。
// 昵称/头像优先取本地 users 表，再回退客户端传入的 displayName。
func (u *ImSessionUsecase) IssueSession(ctx context.Context, userID, displayName string) (*ImSessionOutput, error) {
	if u == nil || u.client == nil {
		return nil, ErrImNotConfigured
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrImUserIDRequired
	}
	name := strings.TrimSpace(displayName)
	portrait := ""
	if u.users != nil {
		if user, err := u.users.FindByUserID(ctx, userID); err == nil && user != nil {
			if name == "" {
				name = strings.TrimSpace(user.UserName)
			}
			portrait = strings.TrimSpace(user.AvatarURL)
		}
	}
	out, err := u.client.GetToken(ctx, userID, name, portrait)
	if err != nil {
		if errors.Is(err, rongcloud.ErrNotConfigured) {
			return nil, ErrImNotConfigured
		}
		return nil, err
	}
	return &ImSessionOutput{
		UserID:           out.UserID,
		Token:            out.Token,
		ExpiresInSeconds: 0, // 融云 Token 默认永久；0=未声明 TTL
	}, nil
}
