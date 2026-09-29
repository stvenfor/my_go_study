package postgres

import (
	_ "embed"
	"fmt"

	"gorm.io/gorm"
)

//go:embed sql/consolidate_users.sql
var consolidateUsersSQL string

//go:embed sql/redesign_users.sql
var redesignUsersSQL string

//go:embed sql/access_control.sql
var accessControlSQL string

//go:embed sql/mall_schema.sql
var mallSchemaSQL string

//go:embed sql/user_address_schema.sql
var userAddressSchemaSQL string

//go:embed sql/points_schema.sql
var pointsSchemaSQL string

//go:embed sql/home_todo_schema.sql
var homeTodoSchemaSQL string

//go:embed sql/new_car_follow_schema.sql
var consolidateNewCarFollowSchemaSQL string

//go:embed sql/cash_wallet_schema.sql
var cashWalletSchemaSQL string

//go:embed sql/membership_schema.sql
var membershipSchemaSQLConsolidate string

// ConsolidateLocalUsers 把遗留用户表收成 user_id 主键的 users，并补上账号状态列。可重复执行。
func ConsolidateLocalUsers(db *gorm.DB) error {
	if err := db.Exec(consolidateUsersSQL).Error; err != nil {
		return fmt.Errorf("合并 users 表失败: %w", err)
	}
	if err := db.Exec(redesignUsersSQL).Error; err != nil {
		return fmt.Errorf("升级 users 表失败: %w", err)
	}
	if err := db.Exec(accessControlSQL).Error; err != nil {
		return fmt.Errorf("建立门店与权限表失败: %w", err)
	}
	if err := db.Exec(mallSchemaSQL).Error; err != nil {
		return fmt.Errorf("建立商城表失败: %w", err)
	}
	if err := db.Exec(userAddressSchemaSQL).Error; err != nil {
		return fmt.Errorf("建立用户地址表失败: %w", err)
	}
	if err := db.Exec(pointsSchemaSQL).Error; err != nil {
		return fmt.Errorf("建立积分表失败: %w", err)
	}
	if err := db.Exec(cashWalletSchemaSQL).Error; err != nil {
		return fmt.Errorf("建立现金钱包表失败: %w", err)
	}
	if err := db.Exec(membershipSchemaSQLConsolidate).Error; err != nil {
		return fmt.Errorf("建立会员订阅表失败: %w", err)
	}
	if err := db.Exec(homeTodoSchemaSQL).Error; err != nil {
		return fmt.Errorf("建立首页待办表失败: %w", err)
	}
	if err := db.Exec(consolidateNewCarFollowSchemaSQL).Error; err != nil {
		return fmt.Errorf("建立新车跟进档案表失败: %w", err)
	}
	return nil
}
