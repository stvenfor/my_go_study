// =============================================================================
// import-supabase — 从 Supabase Cloud 导入用户 / profiles / transactions 到本地 Postgres
//
// 用法（在仓库根目录）：
//
//	./scripts/load-env.sh go run ./cmd/import-supabase --default-password='ChangeMe123!'
//	make import-supabase DEFAULT_PASSWORD='ChangeMe123!'
//
// 说明：
// - Cloud 密码哈希无法经 Admin API 导出；导入用户使用 --default-password（bcrypt）
// - 需 SUPABASE_URL + SUPABASE_SERVICE_ROLE_KEY + 本地 DATABASE_*
// - 幂等：按 UUID / transaction id upsert
// =============================================================================
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/repository/postgres"
	"github.com/stvenfor/my_go_study/pkg/config"
	"github.com/stvenfor/my_go_study/pkg/database"
	pkgsb "github.com/stvenfor/my_go_study/pkg/supabase"
	"github.com/supabase-community/gotrue-go/types"
	postgrest "github.com/supabase-community/postgrest-go"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func main() {
	defaultPassword := flag.String("default-password", "", "导入用户的临时登录密码（必填，Cloud 密码无法导出）")
	dryRun := flag.Bool("dry-run", false, "只拉取并统计，不写本地库")
	skipUsers := flag.Bool("skip-users", false, "跳过 users")
	skipProfiles := flag.Bool("skip-profiles", false, "跳过 profiles")
	skipTransactions := flag.Bool("skip-transactions", false, "跳过 transactions")
	flag.Parse()

	if !*dryRun && strings.TrimSpace(*defaultPassword) == "" && !*skipUsers {
		fmt.Fprintln(os.Stderr, "错误: 请提供 --default-password（或 --skip-users / --dry-run）")
		os.Exit(2)
	}
	if len(strings.TrimSpace(*defaultPassword)) > 0 && len(*defaultPassword) < 6 {
		fmt.Fprintln(os.Stderr, "错误: --default-password 至少 6 位")
		os.Exit(2)
	}

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}
	cfg, err := config.Load(config.ResolveConfigDir(), env)
	if err != nil {
		fatalf("加载配置失败: %v", err)
	}
	if !cfg.Supabase.Enabled() {
		fatalf("未配置 SUPABASE_URL / SUPABASE_ANON_KEY")
	}
	if strings.TrimSpace(cfg.Supabase.ServiceRoleKey) == "" {
		fatalf("未配置 SUPABASE_SERVICE_ROLE_KEY（.env.local）")
	}

	sb, err := pkgsb.New(cfg.Supabase)
	if err != nil {
		fatalf("初始化 Supabase: %v", err)
	}

	var db *gorm.DB
	if !*dryRun {
		db, err = database.NewPostgres(cfg.Database)
		if err != nil {
			fatalf("连接本地 Postgres: %v", err)
		}
		sqlDB, err := db.DB()
		if err != nil {
			fatalf("获取 sql.DB: %v", err)
		}
		defer sqlDB.Close()
		if err := prepareLocalSchema(db); err != nil {
			fatalf("准备本地表结构: %v", err)
		}
	}

	fmt.Printf("源: %s\n目标库: %s@%s:%d/%s  dry-run=%v\n",
		cfg.Supabase.URL, cfg.Database.User, cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName, *dryRun)

	if !*skipUsers {
		users, err := sb.ListAdminUsersAll()
		if err != nil {
			fatalf("拉取 Auth 用户: %v", err)
		}
		fmt.Printf("Auth 用户: %d\n", len(users))
		if !*dryRun {
			n, err := importUsers(db, users, *defaultPassword)
			if err != nil {
				fatalf("导入用户: %v", err)
			}
			fmt.Printf("  写入/更新 users: %d\n", n)
		}
	}

	if !*skipProfiles {
		profiles, err := fetchAllProfiles(sb)
		if err != nil {
			fatalf("拉取 profiles: %v", err)
		}
		fmt.Printf("profiles: %d\n", len(profiles))
		if !*dryRun {
			n, err := importProfiles(db, profiles)
			if err != nil {
				fatalf("导入 profiles: %v", err)
			}
			fmt.Printf("  写入/更新 users 资料: %d\n", n)
		}
	}

	if !*skipTransactions {
		txs, err := fetchAllTransactions(sb)
		if err != nil {
			fatalf("拉取 transactions: %v", err)
		}
		fmt.Printf("transactions: %d\n", len(txs))
		if !*dryRun {
			n, err := importTransactions(db, txs)
			if err != nil {
				fatalf("导入 transactions: %v", err)
			}
			fmt.Printf("  写入/更新 transactions: %d\n", n)
		}
	}

	fmt.Println("完成。导入用户请使用 --default-password 登录后尽快改密（本地暂无改密 API 时可重新 register 新号）。")
}

func importUsers(db *gorm.DB, users []types.User, defaultPassword string) (int, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	hashStr := string(hash)
	n := 0
	for _, u := range users {
		id := u.ID.String()
		if id == "" || id == "00000000-0000-0000-0000-000000000000" {
			continue
		}
		email := strings.ToLower(strings.TrimSpace(u.Email))
		if email == "" {
			// 无邮箱用户跳过（本地登录以邮箱为主）
			fmt.Printf("  skip user %s (无 email)\n", id)
			continue
		}
		display := displayNameFromUser(u)
		phone := strings.TrimSpace(u.Phone)
		row := entity.User{
			UserID:       id,
			UserName:     display,
			Email:        email,
			Phone:        phone,
			PasswordHash: hashStr,
			Status:       entity.UserStatusActive,
		}
		if err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"user_name", "email", "phone", "password_hash", "updated_at"}),
		}).Create(&row).Error; err != nil {
			return n, fmt.Errorf("user %s: %w", id, err)
		}
		n++
	}
	return n, nil
}

func displayNameFromUser(u types.User) string {
	if u.UserMetadata != nil {
		if v, ok := u.UserMetadata["display_name"]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	email := strings.TrimSpace(u.Email)
	if email != "" {
		return strings.Split(email, "@")[0]
	}
	return u.ID.String()
}

func fetchAllProfiles(sb *pkgsb.Client) ([]entity.Profile, error) {
	const pageSize = 500
	var all []entity.Profile
	for offset := 0; ; offset += pageSize {
		data, _, err := sb.Admin.From(entity.ProfilesTable).
			Select("*", "", false).
			Range(offset, offset+pageSize-1, "").
			Execute()
		if err != nil {
			return nil, err
		}
		var batch []entity.Profile
		if err := json.Unmarshal(data, &batch); err != nil {
			return nil, fmt.Errorf("解析 profiles: %w", err)
		}
		all = append(all, batch...)
		if len(batch) < pageSize {
			break
		}
	}
	return all, nil
}

func importProfiles(db *gorm.DB, profiles []entity.Profile) (int, error) {
	n := 0
	for _, p := range profiles {
		id := strings.TrimSpace(p.ID)
		if id == "" {
			continue
		}
		updates := map[string]any{"updated_at": time.Now().UTC()}
		if p.DisplayName != nil {
			updates["user_name"] = *p.DisplayName
		}
		if p.AvatarURL != nil {
			updates["avatar_url"] = *p.AvatarURL
		}
		if p.Phone != nil {
			updates["phone"] = *p.Phone
		}
		if p.CurrentStoreID != nil {
			updates["current_store_id"] = *p.CurrentStoreID
		}
		res := db.Model(&entity.User{}).Where("user_id = ?", id).Updates(updates)
		if res.Error != nil {
			return n, fmt.Errorf("profile %s: %w", id, res.Error)
		}
		if res.RowsAffected > 0 {
			n++
		}
	}
	return n, nil
}

func fetchAllTransactions(sb *pkgsb.Client) ([]entity.Transaction, error) {
	const pageSize = 500
	var all []entity.Transaction
	for offset := 0; ; offset += pageSize {
		data, _, err := sb.Admin.From(entity.TransactionsTable).
			Select("*", "", false).
			Order("id", &postgrest.OrderOpts{Ascending: true}).
			Range(offset, offset+pageSize-1, "").
			Execute()
		if err != nil {
			return nil, err
		}
		var batch []entity.Transaction
		if err := json.Unmarshal(data, &batch); err != nil {
			return nil, fmt.Errorf("解析 transactions: %w", err)
		}
		all = append(all, batch...)
		if len(batch) < pageSize {
			break
		}
	}
	return all, nil
}

func importTransactions(db *gorm.DB, items []entity.Transaction) (int, error) {
	n := 0
	for _, tx := range items {
		if tx.ID == 0 {
			continue
		}
		row := tx
		now := time.Now().UTC()
		if row.CreatedAt == nil {
			row.CreatedAt = &now
		}
		if row.UpdatedAt == nil {
			row.UpdatedAt = &now
		}
		if err := db.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"user_id", "type", "category", "amount", "date", "note", "updated_at",
			}),
		}).Create(&row).Error; err != nil {
			return n, fmt.Errorf("transaction %d: %w", tx.ID, err)
		}
		n++
	}
	return n, nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

// prepareLocalSchema 确保 UUID 版 transactions 可用；旧 bigint user_id 表改名为 legacy。
func prepareLocalSchema(db *gorm.DB) error {
	var dataType string
	_ = db.Raw(`
		SELECT data_type FROM information_schema.columns
		WHERE table_schema = CURRENT_SCHEMA()
		  AND table_name = 'transactions'
		  AND column_name = 'user_id'
	`).Scan(&dataType).Error
	if dataType == "bigint" || dataType == "integer" {
		fmt.Printf("检测到旧 transactions.user_id=%s，重命名为 transactions_legacy_uint\n", dataType)
		if err := db.Exec(`ALTER TABLE transactions RENAME TO transactions_legacy_uint`).Error; err != nil {
			return fmt.Errorf("重命名旧 transactions: %w", err)
		}
	}
	if err := postgres.ConsolidateLocalUsers(db); err != nil {
		return err
	}
	return db.AutoMigrate(
		&entity.User{},
		&entity.AuthRefreshToken{},
		&entity.Transaction{},
	)
}
