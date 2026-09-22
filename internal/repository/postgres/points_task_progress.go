package postgres

import (
	"context"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"gorm.io/gorm"
)

// PointsTaskProgress 成长任务完成事实（社区发帖 / 商城已付订单）。
type PointsTaskProgress struct {
	db  *gorm.DB
	loc *time.Location
}

// NewPointsTaskProgress 创建。
func NewPointsTaskProgress(db *gorm.DB) *PointsTaskProgress {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	return &PointsTaskProgress{db: db, loc: loc}
}

func (p *PointsTaskProgress) dayRange(day string) (time.Time, time.Time, error) {
	start, err := time.ParseInLocation("2006-01-02", day, p.loc)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return start, start.Add(24 * time.Hour), nil
}

// HasLoggedInToday 已鉴权调用方即视为当日已登录。
func (p *PointsTaskProgress) HasLoggedInToday(context.Context, string, string) (bool, error) {
	return true, nil
}

func (p *PointsTaskProgress) HasPostedToday(ctx context.Context, userID, day string) (bool, error) {
	start, end, err := p.dayRange(day)
	if err != nil {
		return false, err
	}
	var n int64
	err = p.db.WithContext(ctx).Model(&entity.WysPost{}).
		Where("user_id = ? AND deleted_at IS NULL AND created_at >= ? AND created_at < ?", userID, start, end).
		Count(&n).Error
	return n > 0, err
}

func (p *PointsTaskProgress) HasPaidOrderToday(ctx context.Context, userID, day string) (bool, error) {
	start, end, err := p.dayRange(day)
	if err != nil {
		return false, err
	}
	var n int64
	err = p.db.WithContext(ctx).Model(&entity.WysMallOrder{}).
		Where("buyer_user_id = ? AND status = ? AND paid_at IS NOT NULL AND paid_at >= ? AND paid_at < ?",
			userID, entity.MallOrderPaid, start, end).
		Count(&n).Error
	return n > 0, err
}
