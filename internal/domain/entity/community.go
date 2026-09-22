package entity

import (
	"time"

	"github.com/google/uuid"
)

// 动态媒体类型（图文与视频互斥）。
const (
	MediaNone  int16 = 0
	MediaImage int16 = 1
	MediaVideo int16 = 2
)

// WysTopic 社区话题。
type WysTopic struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name          string    `gorm:"size:200;not null;uniqueIndex"`
	Heat          int64     `gorm:"not null;default:0"`
	IsAskEveryone bool      `gorm:"not null;default:false"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
}

func (WysTopic) TableName() string { return "wys_topics" }

// WysPost 社区动态。user_id 对齐 users.user_id（字符串 UUID）。
type WysPost struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID        string     `gorm:"column:user_id;size:64;not null;index"`
	Content       string     `gorm:"type:text;not null"`
	MediaType     int16      `gorm:"not null;default:0"`
	ImageURLs     []byte     `gorm:"type:jsonb;not null;default:'[]'"`
	VideoURL      *string    `gorm:"type:text"`
	VideoCoverURL *string    `gorm:"type:text"`
	TopicID       *uuid.UUID `gorm:"type:uuid;index"`
	IsAskEveryone bool       `gorm:"not null;default:false"`
	Source        string     `gorm:"size:64;not null;default:''"`
	LikeCount     int        `gorm:"not null;default:0"`
	CommentCount  int        `gorm:"not null;default:0"`
	DeletedAt     *time.Time `gorm:"index"`
	CreatedAt     time.Time  `gorm:"autoCreateTime;index"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime"`
}

func (WysPost) TableName() string { return "wys_posts" }

// WysPostLike 动态点赞。
type WysPostLike struct {
	PostID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID    string    `gorm:"column:user_id;size:64;primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (WysPostLike) TableName() string { return "wys_post_likes" }

// WysPostComment 动态评论。
type WysPostComment struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey"`
	PostID          uuid.UUID  `gorm:"type:uuid;not null;index"`
	UserID          string     `gorm:"column:user_id;size:64;not null"`
	Content         string     `gorm:"type:text;not null"`
	ReplyToNickname *string    `gorm:"size:100"`
	DeletedAt       *time.Time `gorm:"index"`
	CreatedAt       time.Time  `gorm:"autoCreateTime"`
}

func (WysPostComment) TableName() string { return "wys_post_comments" }
