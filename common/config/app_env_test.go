// app_env_test.go — harden-h7-prod-bootability：APP_ENV 归一化与「缺失覆盖文件拒绝回退」的可跑证明。
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const envTestStrongSecret = "x3Kd9mQra7Vb2nYe8Tg1Ls4Pw6Zh0Jc5Rq8Nf2Bd7Km1Vx"

func setEnvTestProdSecrets(t *testing.T) {
	t.Helper()
	t.Setenv("JWT_SECRET", envTestStrongSecret)
	t.Setenv("DATABASE_PASSWORD", "str0ng-db-pass-8+")
	t.Setenv("SUPABASE_SERVICE_ROLE_KEY", "")
	t.Setenv("DATABASE_SSLMODE", "")
	t.Setenv("DATABASE_AUTO_MIGRATE", "")
}

func TestNormalizeAppEnv(t *testing.T) {
	cases := map[string]string{
		"prod": "prod", "production": "prod", "Production": "prod", " production ": "prod",
		"dev": "dev", "lan": "lan", "staging": "staging", "prodution": "prodution", "": "",
	}
	for in, want := range cases {
		if got := normalizeAppEnv(in); got != want {
			t.Errorf("normalizeAppEnv(%q)=%q want %q", in, got, want)
		}
	}
}

// APP_ENV=production 与 prod 必须都落到 config.prod.yaml，不得静默回退基础 config.yaml
// （config.yaml 是 debug + sslmode=disable，正是 h7 要堵的坑）。
func TestLoadProductionNormalizesToProdOverlay(t *testing.T) {
	setEnvTestProdSecrets(t)
	const dir = "../../configs"

	for _, env := range []string{"prod", "production", "Production", " production "} {
		cfg, err := Load(dir, env)
		if err != nil {
			if strings.Contains(err.Error(), "SERVICE_ROLE") {
				t.Skipf("本机 supabase.env 影响 prod 校验：%v", err)
			}
			t.Fatalf("Load(%q): %v", env, err)
		}
		if cfg.AppEnv != "prod" {
			t.Errorf("APP_ENV=%q 归一化后 AppEnv=%q，期望 prod", env, cfg.AppEnv)
		}
		if cfg.Server.Mode != "release" {
			t.Errorf("APP_ENV=%q server.mode=%q，期望 release（不能是基础配置 debug）", env, cfg.Server.Mode)
		}
		if cfg.Database.SSLMode != "require" {
			t.Errorf("APP_ENV=%q database.sslmode=%q，期望 require", env, cfg.Database.SSLMode)
		}
		if cfg.Database.AutoMigrate {
			t.Errorf("APP_ENV=%q database.auto_migrate 必须为 false", env)
		}
		t.Logf("APP_ENV=%q -> AppEnv=%q server.mode=%q database.sslmode=%q database.auto_migrate=%v",
			env, cfg.AppEnv, cfg.Server.Mode, cfg.Database.SSLMode, cfg.Database.AutoMigrate)
	}
}

// 正式环境缺 config.prod.yaml 必须明确失败并点名缺失文件，绝不 Warn 后按 debug 继续跑。
func TestLoadProdRefusesMissingOverlay(t *testing.T) {
	dir := t.TempDir()
	base := "server:\n  port: 8080\n  mode: debug\ndatabase:\n  sslmode: disable\njwt:\n  secret: \"\"\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(base), 0o600); err != nil {
		t.Fatal(err)
	}
	setEnvTestProdSecrets(t)

	for _, env := range []string{"prod", "production"} {
		_, err := Load(dir, env)
		if err == nil {
			t.Fatalf("Load(%q) 缺 config.prod.yaml 时应拒绝启动", env)
		}
		if !strings.Contains(err.Error(), "config.prod.yaml") {
			t.Fatalf("Load(%q) 错误应点名缺失文件，实得 %v", env, err)
		}
	}
}

// 未知 / 拼错的环境名同样拒绝启动：避免把 production 打错就静默以 debug 基础配置运行。
func TestLoadUnknownEnvRefusesMissingOverlay(t *testing.T) {
	dir := t.TempDir()
	base := "server:\n  port: 8080\n  mode: debug\ndatabase:\n  sslmode: disable\njwt:\n  secret: \"\"\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(base), 0o600); err != nil {
		t.Fatal(err)
	}
	setEnvTestProdSecrets(t)

	_, err := Load(dir, "prodution")
	if err == nil {
		t.Fatal("未知环境名缺覆盖文件时应拒绝启动")
	}
	if !strings.Contains(err.Error(), "config.prodution.yaml") {
		t.Fatalf("错误应点名缺失文件，实得 %v", err)
	}
}

// 既有 P0-1 行为不回归：已知联调环境（含 staging）缺覆盖文件仍回退基础配置。
func TestLoadKnownLabEnvStillFallsBack(t *testing.T) {
	dir := t.TempDir()
	base := "server:\n  port: 8080\n  mode: debug\ndatabase:\n  sslmode: disable\njwt:\n  secret: \"\"\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(base), 0o600); err != nil {
		t.Fatal(err)
	}
	setEnvTestProdSecrets(t)

	if _, err := Load(dir, "staging"); err != nil {
		t.Fatalf("staging 缺覆盖文件仍应能加载（P0-1 逃生门）: %v", err)
	}
}
