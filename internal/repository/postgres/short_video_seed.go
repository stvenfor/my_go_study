package postgres

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"gorm.io/gorm"
)

const shortVideoSeedPhone = "13400000000"
const shortVideoSeedTarget = 16

var shortVideoSeedTitles = []string{
	"周末试驾精彩片段",
	"展厅新车速览",
	"客户交车仪式",
	"售后保养小贴士",
	"雨天行车注意",
	"内饰清洁演示",
	"智驾辅助体验",
	"门店开放日花絮",
	"销售顾问日常",
	"新车到店开箱",
	"试乘路线打卡",
	"车主故事短访",
	"活动现场回顾",
	"配件安装说明",
	"保养误区三则",
	"晚间展厅氛围",
}

// EnsureShortVideoSeed 为测试号 13400000000 补齐 16 条已过审小视频（不足则补）。
func EnsureShortVideoSeed(db *gorm.DB) error {
	var user entity.User
	err := db.Where("(phone = ? OR email = ?) AND deleted_at IS NULL",
		shortVideoSeedPhone, shortVideoSeedPhone+"@dev.test.local").
		First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return fmt.Errorf("查找小视频种子用户失败: %w", err)
	}

	var n int64
	if err := db.Model(&entity.WysShortVideo{}).
		Where("user_id = ? AND deleted_at IS NULL", user.UserID).
		Count(&n).Error; err != nil {
		return err
	}
	need := shortVideoSeedTarget - int(n)
	if need <= 0 {
		return nil
	}

	now := time.Now().UTC()
	urls := []string{
		"https://flutter.github.io/assets-for-api-docs/assets/videos/bee.mp4",
		"https://vjs.zencdn.net/v/oceans.mp4",
		"https://www.w3school.com.cn/example/html5/mov_bbb.mp4",
	}
	covers := []string{
		"https://picsum.photos/seed/sv_play_1/400/640",
		"https://picsum.photos/seed/sv_play_2/400/500",
		"https://picsum.photos/seed/sv_play_3/400/700",
	}
	aspects := []float64{1.25, 0.75, 1.0, 1.4}

	rows := make([]entity.WysShortVideo, 0, need)
	for i := 0; i < need; i++ {
		idx := int(n) + i
		title := shortVideoSeedTitles[idx%len(shortVideoSeedTitles)]
		if idx >= len(shortVideoSeedTitles) {
			title = fmt.Sprintf("%s #%d", title, idx+1)
		}
		created := now.Add(-time.Duration(need-i) * time.Minute)
		approved := created.Add(30 * time.Second)
		rows = append(rows, entity.WysShortVideo{
			ID:          uuid.New(),
			UserID:      user.UserID,
			Title:       title,
			VideoURL:    urls[idx%len(urls)],
			CoverURL:    covers[idx%len(covers)],
			Duration:    "0:15",
			AspectRatio: aspects[idx%len(aspects)],
			Status:      entity.ShortVideoStatusNormal,
			ViewCount:   int64((idx*7)%50 + 1),
			LikeCount:   (idx * 3) % 20,
			ApprovedAt:  &approved,
			CreatedAt:   created,
			UpdatedAt:   created,
		})
	}
	return db.CreateInBatches(rows, 16).Error
}
