package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

// RongCloudGroupClient 融云群操作。
type RongCloudGroupClient interface {
	CreateGroup(ctx context.Context, groupID, groupName string, memberIDs []string) error
	JoinGroup(ctx context.Context, groupID string, memberIDs []string) error
	QuitGroup(ctx context.Context, groupID string, memberIDs []string) error
	DismissGroup(ctx context.Context, groupID, operatorUserID string) error
}

// ImGroupUsecase 门店群 + 自由群。
type ImGroupUsecase struct {
	repo   repository.ImGroupRepository
	rong   RongCloudGroupClient
}

func NewImGroupUsecase(repo repository.ImGroupRepository, rong RongCloudGroupClient) *ImGroupUsecase {
	return &ImGroupUsecase{repo: repo, rong: rong}
}

// ImFreeGroupOut 自由群创建结果。
type ImFreeGroupOut struct {
	GroupID     string `json:"group_id"`
	Name        string `json:"name"`
	OwnerUserID string `json:"owner_user_id"`
}

// ImStoreGroupOut 门店群同步结果。
type ImStoreGroupOut struct {
	StoreID string `json:"store_id"`
	GroupID string `json:"group_id"`
}

// CreateFreeGroup 创建自由群；创建者为群主并入群。
func (u *ImGroupUsecase) CreateFreeGroup(ctx context.Context, ownerID, name string, memberIDs []string) (*ImFreeGroupOut, error) {
	if u == nil || u.repo == nil || u.rong == nil {
		return nil, fmt.Errorf("im group usecase not ready")
	}
	ownerID = strings.TrimSpace(ownerID)
	name = strings.TrimSpace(name)
	if ownerID == "" || name == "" {
		return nil, fmt.Errorf("owner and name required")
	}
	members := uniqueIDs(append(memberIDs, ownerID))
	groupID := "free_" + uuid.NewString()
	if err := u.rong.CreateGroup(ctx, groupID, name, members); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	g := &entity.WysImFreeGroup{
		GroupID: groupID, Name: name, OwnerUserID: ownerID,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := u.repo.CreateFreeGroup(ctx, g, members); err != nil {
		return nil, err
	}
	return &ImFreeGroupOut{GroupID: groupID, Name: name, OwnerUserID: ownerID}, nil
}

// InviteFreeGroup 仅群主可邀请。
func (u *ImGroupUsecase) InviteFreeGroup(ctx context.Context, actorID, groupID string, memberIDs []string) error {
	g, err := u.requireOwner(ctx, actorID, groupID)
	if err != nil {
		return err
	}
	ids := uniqueIDs(memberIDs)
	if len(ids) == 0 {
		return fmt.Errorf("member_ids required")
	}
	if err := u.rong.JoinGroup(ctx, g.GroupID, ids); err != nil {
		return err
	}
	return u.repo.AddFreeMembers(ctx, g.GroupID, ids)
}

// KickFreeGroup 仅群主可踢人（不可踢自己用 kick，应 dismiss）。
func (u *ImGroupUsecase) KickFreeGroup(ctx context.Context, actorID, groupID string, memberIDs []string) error {
	g, err := u.requireOwner(ctx, actorID, groupID)
	if err != nil {
		return err
	}
	ids := make([]string, 0, len(memberIDs))
	for _, id := range uniqueIDs(memberIDs) {
		if id == g.OwnerUserID {
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return fmt.Errorf("no members to kick")
	}
	if err := u.rong.QuitGroup(ctx, g.GroupID, ids); err != nil {
		return err
	}
	return u.repo.RemoveFreeMembers(ctx, g.GroupID, ids)
}

// QuitFreeGroup 成员退群；群主请用解散。
func (u *ImGroupUsecase) QuitFreeGroup(ctx context.Context, actorID, groupID string) error {
	if u == nil || u.repo == nil || u.rong == nil {
		return fmt.Errorf("im group usecase not ready")
	}
	g, err := u.repo.GetFreeGroup(ctx, strings.TrimSpace(groupID))
	if err != nil {
		return err
	}
	if g.DismissedAt != nil {
		return repository.ErrImFreeGroupDismissed
	}
	actorID = strings.TrimSpace(actorID)
	if g.OwnerUserID == actorID {
		return fmt.Errorf("owner must dismiss instead of quit")
	}
	ok, err := u.repo.IsFreeMember(ctx, g.GroupID, actorID)
	if err != nil {
		return err
	}
	if !ok {
		return repository.ErrImNotGroupMember
	}
	if err := u.rong.QuitGroup(ctx, g.GroupID, []string{actorID}); err != nil {
		return err
	}
	return u.repo.RemoveFreeMembers(ctx, g.GroupID, []string{actorID})
}

// DismissFreeGroup 仅群主可解散。
func (u *ImGroupUsecase) DismissFreeGroup(ctx context.Context, actorID, groupID string) error {
	g, err := u.requireOwner(ctx, actorID, groupID)
	if err != nil {
		return err
	}
	if err := u.rong.DismissGroup(ctx, g.GroupID, actorID); err != nil {
		return err
	}
	return u.repo.DismissFreeGroup(ctx, g.GroupID)
}

// EnsureStoreMembership 保证一店一群并把用户拉入（登录补拉 / 入店）。
func (u *ImGroupUsecase) EnsureStoreMembership(ctx context.Context, storeID, userID string) (*ImStoreGroupOut, error) {
	if u == nil || u.repo == nil || u.rong == nil {
		return nil, fmt.Errorf("im group usecase not ready")
	}
	storeID = strings.TrimSpace(storeID)
	userID = strings.TrimSpace(userID)
	if storeID == "" || userID == "" {
		return nil, fmt.Errorf("store_id and user_id required")
	}
	groupID := "store_" + storeID
	existing, err := u.repo.GetStoreGroup(ctx, storeID)
	if err == nil && existing != nil {
		groupID = existing.GroupID
	} else {
		name := "门店群-" + storeID
		if err := u.rong.CreateGroup(ctx, groupID, name, []string{userID}); err != nil {
			// 群可能已存在于融云；继续 join + upsert
			_ = err
		}
		now := time.Now().UTC()
		if err := u.repo.UpsertStoreGroup(ctx, &entity.WysImStoreGroup{
			StoreID: storeID, GroupID: groupID, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			return nil, err
		}
	}
	if err := u.rong.JoinGroup(ctx, groupID, []string{userID}); err != nil {
		return nil, err
	}
	return &ImStoreGroupOut{StoreID: storeID, GroupID: groupID}, nil
}

// RemoveStoreMembership 离店踢出。
func (u *ImGroupUsecase) RemoveStoreMembership(ctx context.Context, storeID, userID string) error {
	if u == nil || u.repo == nil || u.rong == nil {
		return fmt.Errorf("im group usecase not ready")
	}
	g, err := u.repo.GetStoreGroup(ctx, strings.TrimSpace(storeID))
	if err != nil {
		return nil // 无群则 noop
	}
	return u.rong.QuitGroup(ctx, g.GroupID, []string{strings.TrimSpace(userID)})
}

func (u *ImGroupUsecase) requireOwner(ctx context.Context, actorID, groupID string) (*entity.WysImFreeGroup, error) {
	if u == nil || u.repo == nil || u.rong == nil {
		return nil, fmt.Errorf("im group usecase not ready")
	}
	g, err := u.repo.GetFreeGroup(ctx, strings.TrimSpace(groupID))
	if err != nil {
		return nil, err
	}
	if g.DismissedAt != nil {
		return nil, repository.ErrImFreeGroupDismissed
	}
	if g.OwnerUserID != strings.TrimSpace(actorID) {
		return nil, repository.ErrImNotGroupOwner
	}
	return g, nil
}

func uniqueIDs(ids []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
