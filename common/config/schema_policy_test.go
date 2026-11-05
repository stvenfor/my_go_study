package config

import (
	"strings"
	"testing"
)

func TestNoDDLAppEnv(t *testing.T) {
	for _, env := range []string{"prod", "production", "lan", " LAN ", "Production"} {
		if !NoDDLAppEnv(env) {
			t.Fatalf("%q 应禁止启动期 DDL", env)
		}
	}
	for _, env := range []string{"", "dev", "local", "test"} {
		if NoDDLAppEnv(env) {
			t.Fatalf("%q 应允许本机逃生门", env)
		}
	}
}

func TestApplySchemaPolicy(t *testing.T) {
	// lan/prod 无视 yaml：这是「改个 APP_ENV 就漏建表路径」的最后一道闸。
	for _, env := range []string{"lan", "prod"} {
		cfg := &Config{AppEnv: env}
		cfg.Database.AutoMigrate = true
		applySchemaPolicy(cfg)
		if cfg.Database.AutoMigrate {
			t.Fatalf("APP_ENV=%s 仍保留 auto_migrate", env)
		}
	}

	cfg := &Config{AppEnv: "dev", Database: DatabaseConfig{AutoMigrate: true, MigrationsDir: "  "}}
	applySchemaPolicy(cfg)
	if !cfg.Database.AutoMigrate {
		t.Fatal("dev 逃生门被误关")
	}
	if cfg.Database.MigrationsDir != "migrations" {
		t.Fatalf("空 migrations_dir 应回落默认值，得到 %q", cfg.Database.MigrationsDir)
	}
}

// 走真实 configs/ 目录：验证 yaml 三件套 + 「缺 config.<env>.yaml 退回基础配置」。
func TestLoadSchemaGateDefaults(t *testing.T) {
	// configs/config.yaml 的 jwt.secret 已置空（P0-7）：密钥只能由 env 注入。
	t.Setenv("JWT_SECRET", "test-secret-32-bytes-long-aaaaaaaaaa")
	repoConfigs := "../../configs"

	// staging 没有对应文件：以前会「读取环境配置失败」直接起不来（P0-1 已知坑）。
	cfg, err := Load(repoConfigs, "staging")
	if err != nil {
		t.Fatalf("缺环境覆盖文件时应退回 config.yaml: %v", err)
	}
	if cfg.Database.AutoMigrate {
		t.Fatal("基础配置不该开启动期 DDL")
	}
	if !cfg.Database.SchemaVersionCheck {
		t.Fatal("schema_version_check 默认必须为 true（viper 布尔零值会读成 false）")
	}
	if cfg.Database.MigrationsDir != "migrations" {
		t.Fatalf("migrations_dir=%q", cfg.Database.MigrationsDir)
	}

	if cfg, err := Load(repoConfigs, "dev"); err != nil {
		t.Fatalf("Load(dev): %v", err)
	} else if !cfg.Database.AutoMigrate {
		t.Fatal("config.dev.yaml 应保留本机逃生门，否则空库 make run 起不来")
	}

	// compose/lan：即使 env 显式给 true，也要被 APP_ENV=lan 强制关掉。
	t.Setenv("DATABASE_AUTO_MIGRATE", "true")
	if cfg, err := Load(repoConfigs, "lan"); err != nil {
		t.Fatalf("Load(lan): %v", err)
	} else if cfg.Database.AutoMigrate {
		t.Fatal("lan 必须零启动期 DDL（建表交一次性 migrate 服务）")
	}
}

func TestLoadProdKeepsSchemaGateClosed(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("JWT_SECRET", "x3Kd9mQra7Vb2nYe8Tg1Ls4Pw6Zh0Jc5Rq8Nf2Bd7Km1Vx")
	t.Setenv("DATABASE_PASSWORD", "str0ng-db-pass-8+")
	t.Setenv("DATABASE_AUTO_MIGRATE", "true")
	t.Setenv("SUPABASE_SERVICE_ROLE_KEY", "")

	cfg, err := Load("../../configs", "prod")
	if err != nil {
		// 若 configs/supabase.env 里带了 service_role，会走 .env 之外的来源；这里只关心 schema 闸门。
		if strings.Contains(err.Error(), "SERVICE_ROLE") {
			t.Skipf("本机 supabase.env 影响了 prod 校验： %v", err)
		}
		t.Fatalf("Load(prod): %v", err)
	}
	if cfg.Database.AutoMigrate || !cfg.Database.SchemaVersionCheck {
		t.Fatalf("prod schema 闸门异常: %+v", cfg.Database)
	}
}
