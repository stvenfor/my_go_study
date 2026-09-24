package entity

import "time"

const (
	WysImFriendRequestTable  = "wys_im_friend_request"
	WysImFriendshipTable     = "wys_im_friendship"
	WysImFreeGroupTable      = "wys_im_free_group"
	WysImFreeGroupMemberTable = "wys_im_free_group_member"
	WysImStoreGroupTable     = "wys_im_store_group"
	WysImMessageBackupTable  = "wys_im_message_backup"

	ImFriendRequestPending  = "pending"
	ImFriendRequestAccepted = "accepted"
	ImFriendRequestRejected = "rejected"

	ImBackupDirectionOut    = "out"
	ImBackupDirectionIn     = "in"
	ImBackupDirectionRecall = "recall"
)

// WysImFriendRequest 好友申请。
type WysImFriendRequest struct {
	ID         string    `json:"id" gorm:"column:id;primaryKey;size:64"`
	FromUserID string    `json:"from_user_id" gorm:"column:from_user_id;size:64;index;not null"`
	ToUserID   string    `json:"to_user_id" gorm:"column:to_user_id;size:64;index;not null"`
	Status     string    `json:"status" gorm:"column:status;size:16;not null;default:pending"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (WysImFriendRequest) TableName() string { return WysImFriendRequestTable }

// WysImFriendship 双向好友（user_a < user_b 字典序唯一）。
type WysImFriendship struct {
	UserA     string    `json:"user_a" gorm:"column:user_a;primaryKey;size:64"`
	UserB     string    `json:"user_b" gorm:"column:user_b;primaryKey;size:64"`
	CreatedAt time.Time `json:"created_at"`
}

func (WysImFriendship) TableName() string { return WysImFriendshipTable }

// WysImFreeGroup 自由群元数据（群主制）。
type WysImFreeGroup struct {
	GroupID     string     `json:"group_id" gorm:"column:group_id;primaryKey;size:128"`
	Name        string     `json:"name" gorm:"column:name;size:128;not null"`
	OwnerUserID string     `json:"owner_user_id" gorm:"column:owner_user_id;size:64;index;not null"`
	DismissedAt *time.Time `json:"dismissed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (WysImFreeGroup) TableName() string { return WysImFreeGroupTable }

// WysImFreeGroupMember 自由群成员。
type WysImFreeGroupMember struct {
	GroupID  string    `json:"group_id" gorm:"column:group_id;primaryKey;size:128"`
	UserID   string    `json:"user_id" gorm:"column:user_id;primaryKey;size:64"`
	JoinedAt time.Time `json:"joined_at"`
}

func (WysImFreeGroupMember) TableName() string { return WysImFreeGroupMemberTable }

// WysImStoreGroup 门店 ↔ 融云群（一店一群）。
type WysImStoreGroup struct {
	StoreID   string    `json:"store_id" gorm:"column:store_id;primaryKey;size:64"`
	GroupID   string    `json:"group_id" gorm:"column:group_id;size:128;uniqueIndex;not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (WysImStoreGroup) TableName() string { return WysImStoreGroupTable }

// WysImMessageBackup 消息业务备份（message_uid 幂等）。
type WysImMessageBackup struct {
	ID               int64      `json:"id" gorm:"primaryKey"`
	MessageUID       string     `json:"message_uid" gorm:"column:message_uid;size:128;uniqueIndex;not null"`
	UserID           string     `json:"user_id" gorm:"column:user_id;size:64;index;not null"`
	Direction        string     `json:"direction" gorm:"column:direction;size:16;not null"`
	ConversationType string     `json:"conversation_type" gorm:"column:conversation_type;size:32"`
	TargetID         string     `json:"target_id" gorm:"column:target_id;size:128;index"`
	MessageType      string     `json:"message_type" gorm:"column:message_type;size:64"`
	Payload          string     `json:"payload" gorm:"column:payload;type:text"`
	SentAt           *time.Time `json:"sent_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

func (WysImMessageBackup) TableName() string { return WysImMessageBackupTable }
