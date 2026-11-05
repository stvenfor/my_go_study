// Package schemaboot 是进程启动期唯一允许碰 DDL 的入口（parity-p0-1）。
//
// 之前有四套建表机制并存（GORM AutoMigrate、内嵌 SQL、8 处域 EnsureSchema、
// migrations/ 目录），每个实例启动都跑一遍 DDL。这里把前三类收进一个开关，
// 并让 migrations/ 成为正式环境的单一真相：启动只读版本、不建表，版本落后就拒绝启动。
package schemaboot

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Step 是一条启动期 DDL 动作（由调用方注入，本包不依赖任何业务域）。
type Step struct {
	Name string
	Fn   func() error
}

// Options 启动闸门参数。
type Options struct {
	AppEnv      string
	AutoMigrate bool
	// VersionCheck 为 true 时比对 schema_migrations 与 migrations/ 目录版本。
	VersionCheck  bool
	MigrationsDir string
	// Required：失败即拒绝启动（核心建表链）。
	Required []Step
	// BestEffort：域内兜底建表，失败仅 Warn，保持原有可用性。
	BestEffort []Step
}

// Boot 执行「允许的 DDL + 版本校验」。返回 error 时调用方必须退出。
func Boot(db *gorm.DB, opt Options, log *zap.Logger) error {
	if opt.AutoMigrate {
		log.Info("启动期 DDL 已启用（database.auto_migrate=true）", zap.String("app_env", opt.AppEnv))
		for _, s := range opt.Required {
			if err := runStep(s); err != nil {
				return fmt.Errorf("启动建表失败（%s）: %w", s.Name, err)
			}
		}
		for _, s := range opt.BestEffort {
			if err := runStep(s); err != nil {
				log.Warn("域内建表兜底失败，功能可能受限", zap.String("step", s.Name), zap.Error(err))
			}
		}
	} else {
		log.Info("启动期 DDL 已关闭：schema 由发布流程 migrations/ 管理",
			zap.String("app_env", opt.AppEnv),
			zap.String("how_to_apply", "make migrate-up"),
		)
	}

	if !opt.VersionCheck {
		log.Warn("schema 版本校验已关闭（database.schema_version_check=false）",
			zap.String("app_env", opt.AppEnv))
		return nil
	}

	expected, missingDown, err := LoadMigrations(opt.MigrationsDir)
	if err != nil {
		return err
	}
	if len(missingDown) > 0 {
		log.Warn("以下迁移缺 .down.sql，回滚会失败（不影响启动）",
			zap.Strings("migrations", missingDown))
	}

	state, err := readState(db)
	if err != nil {
		return err
	}

	switch {
	case state.tableMissing:
		return handleNoVersionTable(db, opt, expected, log)
	case state.dirty:
		return fmt.Errorf("上一次迁移中断（schema_migrations.dirty=true, version=%d）："+
			"人工修好库或补齐该迁移后执行 make migrate-force 重置版本再启动", state.version)
	case state.version < expected[len(expected)-1].Version:
		missing := namesAfter(expected, state.version)
		return fmt.Errorf("schema 版本落后：库内 %d，迁移目录最高 %d；缺失 %d 个迁移：%s；执行 make migrate-up",
			state.version, expected[len(expected)-1].Version, len(missing), strings.Join(missing, ", "))
	}

	log.Info("schema 版本校验通过",
		zap.Int64("version", state.version),
		zap.Int("migrations", len(expected)),
	)
	return nil
}

// handleNoVersionTable 处理「库里没有 schema_migrations 表」：
// 本地 AutoMigrate 路径跳过；已有业务表但没版本记录 → 提示基线化后放行；空库 → 拒绝启动。
func handleNoVersionTable(db *gorm.DB, opt Options, expected []Migration, log *zap.Logger) error {
	if opt.AutoMigrate {
		log.Info("本地 AutoMigrate 路径：schema 未纳入迁移版本，跳过版本校验")
		return nil
	}
	baseline, err := hasBaselineTable(db)
	if err != nil {
		return err
	}
	if baseline {
		log.Warn("库由历史 AutoMigrate 建立，尚无 schema_migrations 版本记录",
			zap.String("baseline_now", "make migrate-baseline"),
			zap.String("then", "make migrate-up"),
		)
		return nil
	}
	return fmt.Errorf("数据库未初始化：缺少 schema_migrations 且无业务表；先执行 make migrate-up（compose 环境由一次性 migrate 服务负责）")
}

func runStep(s Step) error {
	if s.Fn == nil {
		return nil
	}
	return s.Fn()
}

type state struct {
	tableMissing bool
	dirty        bool
	version      int64
}

func readState(db *gorm.DB) (state, error) {
	ok, err := tableExists(db, schemaMigrationsTable)
	if err != nil {
		return state{}, err
	}
	if !ok {
		return state{tableMissing: true}, nil
	}
	var row struct {
		Version int64
		Dirty   bool
	}
	if err := db.Table(schemaMigrationsTable).Select("version", "dirty").Take(&row).Error; err != nil {
		return state{}, fmt.Errorf("读取 %s 失败: %w", schemaMigrationsTable, err)
	}
	return state{version: row.Version, dirty: row.Dirty}, nil
}

// hasBaselineTable 用一张必然存在的业务表判断「这不是空库」。
func hasBaselineTable(db *gorm.DB) (bool, error) {
	return tableExists(db, baselineTable)
}

// HasTable 供装配处判断「表是否已存在」，从而决定要不要构造会自建表的第三方组件
// （如 Casbin gorm-adapter）。闸门关着的进程不该借别人的手做 DDL。
func HasTable(db *gorm.DB, table string) (bool, error) {
	return tableExists(db, table)
}

// tableExists 走 to_regclass：按 search_path 解析，表不存在时返回 NULL 而不是报错。
func tableExists(db *gorm.DB, table string) (bool, error) {
	var out sql.NullString
	if err := db.Raw("SELECT to_regclass(?)", table).Row().Scan(&out); err != nil {
		return false, fmt.Errorf("检查表 %s 失败: %w", table, err)
	}
	return out.Valid && out.String != "", nil
}

func namesAfter(all []Migration, version int64) []string {
	var out []string
	for _, m := range all {
		if m.Version > version {
			out = append(out, m.Name)
		}
	}
	sort.Strings(out)
	return out
}
