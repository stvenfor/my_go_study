package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSecretsProdRejectsWeakAndShort(t *testing.T) {
	cases := []struct {
		name   string
		env    string
		secret string
		wantOK bool
	}{
		{"prod 占位值", "prod", "change-me-in-production", false},
		{"production 占位值", "production", "change-me-in-lan", false},
		{"prod 过短", "prod", "abc123", false},
		{"prod 含空格", "prod", "aaaa bbbb cccc dddd eeee ffff", false},
		{"prod 全同字符", "prod", strings.Repeat("a", 40), false},
		{"prod 强值", "prod", strings.Repeat("a", 16) + strings.Repeat("B", 8) + "12345678", true},
		{"dev 放行弱值", "dev", "change-me-in-production", true},
		{"lan 放行弱值", "lan", "change-me-in-lan", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &Config{AppEnv: tc.env}
			c.JWT.Secret = tc.secret
			c.Database.Password = strings.Repeat("x", 12)
			err := validateSecrets(c)
			if tc.wantOK && err != nil {
				t.Fatalf("应放行，实得 %v", err)
			}
			if !tc.wantOK && err == nil {
				t.Fatal("应拒绝，实得 nil")
			}
			// 错误信息不得回显密钥值本身
			if err != nil && strings.Contains(err.Error(), tc.secret) {
				t.Fatalf("错误信息泄露密钥值: %v", err)
			}
		})
	}
}

func TestValidateSecretsProdRejectsDefaultDBPassword(t *testing.T) {
	c := &Config{AppEnv: "prod"}
	c.JWT.Secret = strings.Repeat("k", 32) + "1A"
	c.Database.Password = "postgres"
	if err := validateSecrets(c); err == nil || !strings.Contains(err.Error(), "DATABASE_PASSWORD") {
		t.Fatalf("默认 DB 密码应被拒绝，实得 %v", err)
	}
}

// service_role 来自入库 .env 时 prod 必须拒绝；来自 OS env（source=""）放行。
func TestValidateSecretsServiceRoleSource(t *testing.T) {
	c := &Config{AppEnv: "prod"}
	c.JWT.Secret = strings.Repeat("k", 32) + "1A"
	c.Database.Password = strings.Repeat("x", 12)
	c.Supabase.ServiceRoleKey = "any-value"

	c.serviceRoleSource = ".env"
	if err := validateSecrets(c); err == nil || !strings.Contains(err.Error(), ".env") {
		t.Fatalf("来自 .env 的 service_role 应被拒绝，实得 %v", err)
	}
	c.serviceRoleSource = ".env.local"
	if err := validateSecrets(c); err != nil {
		t.Fatalf("来自 .env.local 应放行，实得 %v", err)
	}
	c.serviceRoleSource = ""
	if err := validateSecrets(c); err != nil {
		t.Fatalf("来自 OS env 应放行，实得 %v", err)
	}
}

// lab 环境不拦，但要能在启动日志里看到清单（且同样不泄露值）。
func TestSecretFindingsLabListsWeakValues(t *testing.T) {
	c := &Config{AppEnv: "dev"}
	c.JWT.Secret = "change-me-in-production"
	c.Database.Password = "postgres"
	c.Supabase.ServiceRoleKey = "leaked"
	c.serviceRoleSource = ".env"

	findings := c.SecretFindings()
	if len(findings) != 3 {
		t.Fatalf("期望 3 条提示，实得 %d: %v", len(findings), findings)
	}
	for _, f := range findings {
		if strings.Contains(f, "change-me-in-production") || strings.Contains(f, "postgres") || strings.Contains(f, "leaked") {
			t.Fatalf("提示泄露密钥值: %s", f)
		}
	}
}

func TestLoadProdRejectsPlaceholderJWTSecret(t *testing.T) {
	dir := t.TempDir()
	base := "server:\n  port: 8080\njwt:\n  secret: \"\"\n  expire_hours: 72\ndatabase:\n  password: \"\"\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(base), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.prod.yaml"), []byte("server:\n  mode: release\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.dev.yaml"), []byte("server:\n  mode: debug\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("JWT_SECRET", "change-me-in-production")
	t.Setenv("DATABASE_PASSWORD", "a-real-looking-db-password")

	_, err := Load(dir, "prod")
	if err == nil {
		t.Fatal("prod 下占位 JWT_SECRET 应拒绝加载")
	}
	if !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("错误应点名 JWT_SECRET，实得 %v", err)
	}

	// 同一个占位值在 dev 下仍可加载（联调不被打断）
	t.Setenv("APP_ENV", "dev")
	if _, err := Load(dir, "dev"); err != nil {
		t.Fatalf("dev 应放行占位值，实得 %v", err)
	}
}
