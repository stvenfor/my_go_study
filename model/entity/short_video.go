// short_video.go 小视频实体。
package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	WysShortVideoTable     = "wys_short_videos"
	WysShortVideoLikeTable = "wys_short_video_likes"

	ShortVideoStatusReviewing int16 = 0
	ShortVideoStatusNormal    int16 = 1
)

// WysShortVideo 小视频（独立于社区动态）。
type WysShortVideo struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID       string     `gorm:"column:user_id;size:64;not null;index"`
	Title        string     `gorm:"type:text;not null"`
	VideoURL     string     `gorm:"type:text;not null"`
	CoverURL     string     `gorm:"type:text;not null"`
	Duration     string     `gorm:"size:32;not null;default:'0:15'"`
	AspectRatio  float64    `gorm:"not null;default:1.25"`
	TopicID      *uuid.UUID `gorm:"type:uuid;index"`
	Status       int16      `gorm:"not null;default:0;index"`
	ViewCount    int64      `gorm:"not null;default:0"`
	LikeCount    int        `gorm:"not null;default:0"`
	ApprovedAt   *time.Time
	DeletedAt    *time.Time `gorm:"index"`
	CreatedAt    time.Time  `gorm:"autoCreateTime;index"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime"`
}

func (WysShortVideo) TableName() string { return WysShortVideoTable }

// WysShortVideoLike 小视频点赞。
type WysShortVideoLike struct {
	ShortVideoID uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID       string    `gorm:"column:user_id;size:64;primaryKey"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
}

func (WysShortVideoLike) TableName() string { return WysShortVideoLikeTable }

// ShortVideoStats 作者维度统计。
type ShortVideoStats struct {
	VideoCount int64
	ViewCount  int64
	LikeCount  int64
}
