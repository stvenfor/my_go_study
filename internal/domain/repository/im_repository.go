package repository

import (
	"context"
	"errors"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

var (
	ErrImFriendRequestNotFound = errors.New("im friend request not found")
	ErrImFreeGroupNotFound     = errors.New("im free group not found")
	ErrImFreeGroupDismissed    = errors.New("im free group dismissed")
	ErrImNotGroupOwner         = errors.New("im not group owner")
	ErrImNotGroupMember        = errors.New("im not group member")
)

// ImFriendRepository 好友申请与关系。
type ImFriendRepository interface {
	CreateRequest(ctx context.Context, req *entity.WysImFriendRequest) error
	GetRequest(ctx context.Context, id string) (*entity.WysImFriendRequest, error)
	UpdateRequestStatus(ctx context.Context, id, status string) error
	FindPending(ctx context.Context, fromUserID, toUserID string) (*entity.WysImFriendRequest, error)
	ListPendingTo(ctx context.Context, toUserID string) ([]*entity.WysImFriendRequest, error)
	AreFriends(ctx context.Context, userA, userB string) (bool, error)
	UpsertFriendship(ctx context.Context, userA, userB string) error
	ListFriendIDs(ctx context.Context, userID string) ([]string, error)
}

// ImGroupRepository 自由群 + 门店群映射。
type ImGroupRepository interface {
	CreateFreeGroup(ctx context.Context, g *entity.WysImFreeGroup, memberIDs []string) error
	GetFreeGroup(ctx context.Context, groupID string) (*entity.WysImFreeGroup, error)
	AddFreeMembers(ctx context.Context, groupID string, memberIDs []string) error
	RemoveFreeMembers(ctx context.Context, groupID string, memberIDs []string) error
	IsFreeMember(ctx context.Context, groupID, userID string) (bool, error)
	DismissFreeGroup(ctx context.Context, groupID string) error
	GetStoreGroup(ctx context.Context, storeID string) (*entity.WysImStoreGroup, error)
	UpsertStoreGroup(ctx context.Context, g *entity.WysImStoreGroup) error
}

// ImBackupRepository 消息业务备份。
type ImBackupRepository interface {
	InsertIgnore(ctx context.Context, row *entity.WysImMessageBackup) (inserted bool, err error)
}

// ImUserLookup 按 UUID / 手机号查本地用户（users 表）。
type ImUserLookup interface {
	FindByUserID(ctx context.Context, userID string) (*entity.User, error)
	FindByUserIDs(ctx context.Context, userIDs []string) ([]*entity.User, error)
	FindByPhone(ctx context.Context, phoneDigits string) (*entity.User, error)
}
