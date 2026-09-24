package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

var (
	ErrImFriendSelf        = errors.New("cannot friend yourself")
	ErrImFriendAlready     = errors.New("already friends")
	ErrImFriendPending     = errors.New("friend request already pending")
	ErrImFriendUserMissing = errors.New("user not found")
	ErrImFriendNotPeer     = errors.New("not the request recipient")
	ErrImSearchQueryEmpty  = errors.New("search query required")
)

// ImFriendUsecase 双向好友与单聊准入。
type ImFriendUsecase struct {
	friends repository.ImFriendRepository
	users   repository.ImUserLookup
}

func NewImFriendUsecase(friends repository.ImFriendRepository, users repository.ImUserLookup) *ImFriendUsecase {
	return &ImFriendUsecase{friends: friends, users: users}
}

// ImUserBrief 搜索/好友列表项。
type ImUserBrief struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	PhoneMasked string `json:"phone_masked,omitempty"`
}

// ImFriendRequestOut 申请回执。
type ImFriendRequestOut struct {
	ID         string `json:"id"`
	FromUserID string `json:"from_user_id"`
	ToUserID   string `json:"to_user_id"`
	Status     string `json:"status"`
}

// ImFriendRequestItem 待处理申请（收件箱）。
type ImFriendRequestItem struct {
	ID          string `json:"id"`
	FromUserID  string `json:"from_user_id"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	PhoneMasked string `json:"phone_masked,omitempty"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

// Search 按手机号或 UUID 精确查找（不含自己）。
func (u *ImFriendUsecase) Search(ctx context.Context, selfID, q string) ([]ImUserBrief, error) {
	if u == nil || u.users == nil {
		return nil, fmt.Errorf("im friend usecase not ready")
	}
	q = strings.TrimSpace(q)
	if q == "" {
		return nil, ErrImSearchQueryEmpty
	}
	selfID = strings.TrimSpace(selfID)
	var user *entity.User
	var err error
	if looksLikeUUID(q) {
		user, err = u.users.FindByUserID(ctx, q)
	} else {
		user, err = u.users.FindByPhone(ctx, digitsOnly(q))
	}
	if err != nil || user == nil {
		return []ImUserBrief{}, nil
	}
	if user.UserID == selfID {
		return []ImUserBrief{}, nil
	}
	return []ImUserBrief{{
		UserID:      user.UserID,
		DisplayName: user.UserName,
		AvatarURL:   user.AvatarURL,
		PhoneMasked: maskPhone(user.Phone),
	}}, nil
}

// ListFriends 好友 ID 列表（附简要资料若 lookup 可用）。
func (u *ImFriendUsecase) ListFriends(ctx context.Context, userID string) ([]ImUserBrief, error) {
	if u == nil || u.friends == nil {
		return nil, fmt.Errorf("im friend usecase not ready")
	}
	ids, err := u.friends.ListFriendIDs(ctx, strings.TrimSpace(userID))
	if err != nil {
		return nil, err
	}
	out := make([]ImUserBrief, 0, len(ids))
	for _, id := range ids {
		brief := ImUserBrief{UserID: id, DisplayName: id}
		if u.users != nil {
			if user, err := u.users.FindByUserID(ctx, id); err == nil && user != nil {
				brief.DisplayName = user.UserName
				brief.AvatarURL = user.AvatarURL
				brief.PhoneMasked = maskPhone(user.Phone)
			}
		}
		out = append(out, brief)
	}
	return out, nil
}

// ListProfiles 按业务 UUID（= 融云 userId）批量查昵称/头像。
func (u *ImFriendUsecase) ListProfiles(ctx context.Context, userIDs []string) ([]ImUserBrief, error) {
	if u == nil || u.users == nil {
		return nil, fmt.Errorf("im friend usecase not ready")
	}
	users, err := u.users.FindByUserIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	out := make([]ImUserBrief, 0, len(users))
	for _, user := range users {
		if user == nil {
			continue
		}
		name := strings.TrimSpace(user.UserName)
		if name == "" {
			name = user.UserID
		}
		out = append(out, ImUserBrief{
			UserID:      user.UserID,
			DisplayName: name,
			AvatarURL:   user.AvatarURL,
			PhoneMasked: maskPhone(user.Phone),
		})
	}
	return out, nil
}

// Request 发起好友申请。
func (u *ImFriendUsecase) Request(ctx context.Context, fromUserID, toUserID string) (*ImFriendRequestOut, error) {
	if u == nil || u.friends == nil || u.users == nil {
		return nil, fmt.Errorf("im friend usecase not ready")
	}
	fromUserID = strings.TrimSpace(fromUserID)
	toUserID = strings.TrimSpace(toUserID)
	if fromUserID == "" || toUserID == "" {
		return nil, ErrImFriendUserMissing
	}
	if fromUserID == toUserID {
		return nil, ErrImFriendSelf
	}
	if _, err := u.users.FindByUserID(ctx, toUserID); err != nil {
		return nil, ErrImFriendUserMissing
	}
	ok, err := u.friends.AreFriends(ctx, fromUserID, toUserID)
	if err != nil {
		return nil, err
	}
	if ok {
		return nil, ErrImFriendAlready
	}
	if pending, _ := u.friends.FindPending(ctx, fromUserID, toUserID); pending != nil {
		return nil, ErrImFriendPending
	}
	now := time.Now().UTC()
	req := &entity.WysImFriendRequest{
		ID:         uuid.NewString(),
		FromUserID: fromUserID,
		ToUserID:   toUserID,
		Status:     entity.ImFriendRequestPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := u.friends.CreateRequest(ctx, req); err != nil {
		return nil, err
	}
	return &ImFriendRequestOut{
		ID: req.ID, FromUserID: req.FromUserID, ToUserID: req.ToUserID, Status: req.Status,
	}, nil
}

// ListIncoming 当前用户待处理的好友申请。
func (u *ImFriendUsecase) ListIncoming(ctx context.Context, toUserID string) ([]ImFriendRequestItem, error) {
	if u == nil || u.friends == nil {
		return nil, fmt.Errorf("im friend usecase not ready")
	}
	reqs, err := u.friends.ListPendingTo(ctx, strings.TrimSpace(toUserID))
	if err != nil {
		return nil, err
	}
	out := make([]ImFriendRequestItem, 0, len(reqs))
	for _, req := range reqs {
		if req == nil {
			continue
		}
		item := ImFriendRequestItem{
			ID:         req.ID,
			FromUserID: req.FromUserID,
			DisplayName: req.FromUserID,
			Status:     req.Status,
			CreatedAt:  req.CreatedAt.UTC().Format(time.RFC3339),
		}
		if u.users != nil {
			if user, err := u.users.FindByUserID(ctx, req.FromUserID); err == nil && user != nil {
				name := strings.TrimSpace(user.UserName)
				if name != "" {
					item.DisplayName = name
				}
				item.AvatarURL = user.AvatarURL
				item.PhoneMasked = maskPhone(user.Phone)
			}
		}
		out = append(out, item)
	}
	return out, nil
}

// Respond 同意或拒绝申请。
func (u *ImFriendUsecase) Respond(ctx context.Context, actorID, requestID string, accept bool) (*ImFriendRequestOut, error) {
	if u == nil || u.friends == nil {
		return nil, fmt.Errorf("im friend usecase not ready")
	}
	req, err := u.friends.GetRequest(ctx, strings.TrimSpace(requestID))
	if err != nil {
		return nil, err
	}
	if req.ToUserID != strings.TrimSpace(actorID) {
		return nil, ErrImFriendNotPeer
	}
	if req.Status != entity.ImFriendRequestPending {
		return &ImFriendRequestOut{
			ID: req.ID, FromUserID: req.FromUserID, ToUserID: req.ToUserID, Status: req.Status,
		}, nil
	}
	status := entity.ImFriendRequestRejected
	if accept {
		status = entity.ImFriendRequestAccepted
		if err := u.friends.UpsertFriendship(ctx, req.FromUserID, req.ToUserID); err != nil {
			return nil, err
		}
	}
	if err := u.friends.UpdateRequestStatus(ctx, req.ID, status); err != nil {
		return nil, err
	}
	return &ImFriendRequestOut{
		ID: req.ID, FromUserID: req.FromUserID, ToUserID: req.ToUserID, Status: status,
	}, nil
}

// CanPrivateChat 单聊准入：须已是好友。
func (u *ImFriendUsecase) CanPrivateChat(ctx context.Context, userA, userB string) (bool, error) {
	if u == nil || u.friends == nil {
		return false, fmt.Errorf("im friend usecase not ready")
	}
	userA = strings.TrimSpace(userA)
	userB = strings.TrimSpace(userB)
	if userA == "" || userB == "" || userA == userB {
		return false, nil
	}
	return u.friends.AreFriends(ctx, userA, userB)
}

func looksLikeUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func maskPhone(phone string) string {
	d := digitsOnly(phone)
	if len(d) < 7 {
		return ""
	}
	return d[:3] + "****" + d[len(d)-4:]
}
