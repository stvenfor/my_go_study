package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const packingDemoPhone = "13400000000"

// EnsureHomeTodoPackingDemo 为测试号 13400000000 准备装箱演示：
// 店管权限 + 当前店 1，领域数据支撑首页「1 大 + 3 中 + 4 小」卡面。
func EnsureHomeTodoPackingDemo(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var user entity.User
		err := tx.Where("(phone = ? OR email = ?) AND deleted_at IS NULL",
			packingDemoPhone, packingDemoPhone+"@dev.test.local").
			First(&user).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil // 尚未登录创建账号，下次启动再种
			}
			return fmt.Errorf("查找测试手机号用户失败: %w", err)
		}

		storeID := 1
		if err := tx.Exec(`
			INSERT INTO wys_store (store_id, name) VALUES (?, ?)
			ON CONFLICT (store_id) DO NOTHING
		`, storeID, "[4S]北京沃德龙鼎吉利").Error; err != nil {
			return err
		}

		member := entity.WysStoreMember{UserID: user.UserID, StoreID: storeID, Position: 2}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "store_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"position", "updated_at"}),
		}).Create(&member).Error; err != nil {
			return err
		}

		if err := tx.Exec(`
			INSERT INTO user_role (user_id, role_code, store_id, granted_by, created_at)
			VALUES (?, 'store_admin', ?, NULL, now())
			ON CONFLICT (user_id, role_code, store_id) WHERE store_id IS NOT NULL DO NOTHING
		`, user.UserID, storeID).Error; err != nil {
			return err
		}

		if err := tx.Model(&entity.User{}).
			Where("user_id = ?", user.UserID).
			Update("current_store_id", storeID).Error; err != nil {
			return err
		}

		if err := tx.Exec(`
			CREATE TABLE IF NOT EXISTS wys_home_todo_packing_demo (
			  store_id integer PRIMARY KEY,
			  large_n integer NOT NULL DEFAULT 1,
			  medium_n integer NOT NULL DEFAULT 3,
			  small_n integer NOT NULL DEFAULT 4,
			  created_at timestamptz NOT NULL DEFAULT now()
			)
		`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			INSERT INTO wys_home_todo_packing_demo (store_id, large_n, medium_n, small_n)
			VALUES (?, 1, 3, 4)
			ON CONFLICT (store_id) DO UPDATE SET large_n=1, medium_n=3, small_n=4
		`, storeID).Error; err != nil {
			return err
		}

		applicantID := "todo-packing-applicant"
		if err := ensureApplicantUser(tx, applicantID); err != nil {
			return err
		}

		// 大卡：1 条 pending 入店申请
		if err := tx.Where("store_id = ? AND status = ?", storeID, entity.JoinStatusPending).
			Delete(&entity.WysStoreJoinApplication{}).Error; err != nil {
			return err
		}
		app := entity.WysStoreJoinApplication{
			StoreID:         storeID,
			ApplicantUserID: applicantID,
			Status:          entity.JoinStatusPending,
		}
		if err := tx.Create(&app).Error; err != nil {
			return err
		}

		// 中卡数据：3 待跟进 + 若干预约（聚合演示会拆成 3 张中卡）
		if err := tx.Where("store_id = ?", storeID).Delete(&entity.WysStoreCustomer{}).Error; err != nil {
			return err
		}
		past := time.Now().Add(-2 * time.Hour)
		for i := 1; i <= 3; i++ {
			row := entity.WysStoreCustomer{
				StoreID:        storeID,
				DisplayName:    fmt.Sprintf("待跟进客户-%d", i),
				Phone:          fmt.Sprintf("1380000000%d", i),
				NextFollowUpAt: &past,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}

		if err := tx.Where("store_id = ?", storeID).Delete(&entity.WysAfterSalesAppointment{}).Error; err != nil {
			return err
		}
		loc, err := time.LoadLocation("Asia/Shanghai")
		if err != nil {
			loc = time.FixedZone("CST", 8*3600)
		}
		local := time.Now().In(loc)
		day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
		for i := 1; i <= 3; i++ {
			row := entity.WysAfterSalesAppointment{
				StoreID:         storeID,
				CustomerName:    fmt.Sprintf("预约客户-%d", i),
				AppointmentDate: day,
				Status:          entity.AppointmentStatusPending,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}

		// 小卡：4 笔待审店务单
		if err := tx.Where("store_id = ?", storeID).Delete(&entity.WysStoreReviewOrder{}).Error; err != nil {
			return err
		}
		for i := 1; i <= 4; i++ {
			row := entity.WysStoreReviewOrder{
				StoreID: storeID,
				Title:   fmt.Sprintf("店务审核单-%d", i),
				Status:  entity.ReviewOrderStatusPending,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func ensureApplicantUser(tx *gorm.DB, userID string) error {
	var n int64
	if err := tx.Model(&entity.User{}).Where("user_id = ?", userID).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	now := time.Now()
	u := entity.User{
		UserID:       userID,
		UserName:     "待确认伙伴",
		Email:        userID + "@seed.local",
		Phone:        "",
		PasswordHash: "!", // 不可登录占位
		Status:       entity.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	return tx.Create(&u).Error
}

// GetPackingDemoSpec 若该店启用装箱演示则返回规格。
func (r *HomeTodoRepository) GetPackingDemoSpec(ctx context.Context, storeID int) (*repository.HomeTodoPackingDemoSpec, error) {
	var row struct {
		LargeN  int `gorm:"column:large_n"`
		MediumN int `gorm:"column:medium_n"`
		SmallN  int `gorm:"column:small_n"`
	}
	err := r.db.WithContext(ctx).
		Table("wys_home_todo_packing_demo").
		Select("large_n, medium_n, small_n").
		Where("store_id = ?", storeID).
		Take(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "does not exist") {
			return nil, nil
		}
		return nil, err
	}
	return &repository.HomeTodoPackingDemoSpec{
		LargeN:  row.LargeN,
		MediumN: row.MediumN,
		SmallN:  row.SmallN,
	}, nil
}
