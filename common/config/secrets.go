// secrets.go 启动期密钥强度校验。
// 只报「命中占位默认值 / 长度不足 / 来源不合规」，任何日志与错误信息都不得带出密钥值本身。
package config

import (
	"fmt"
	"strings"
)

// knownWeakSecrets 教程/脚手架默认值：本地联调能用，部署必改。
var knownWeakSecrets = map[string]struct{}{
	"change-me":               {},
	"change-me-in-dev":        {},
	"change-me-in-lan":        {},
	"change-me-in-production": {},
	"secret":                  {},
	"jwt-secret":              {},
	"your-secret":             {},
	"postgres":                {},
	"password":                {},
	"admin":                   {},
	"root":                    {},
	"123456":                  {},
	"dev-test-phone-secret":   {},
}

// MinProdSecretLen prod 下 JWT 密钥最短长度。
const MinProdSecretLen = 32

func isKnownWeakSecret(value string) bool {
	_, ok := knownWeakSecrets[strings.ToLower(strings.TrimSpace(value))]
	return ok
}

// secretShapeProblem 只凭长度/字符构成判断的问题（不做内容比对，返回值可安全入日志）。
func secretShapeProblem(value string) string {
	switch {
	case value == "":
		return "未设置"
	case strings.ContainsAny(value, " \t\n\"'"):
		return "含空白或引号（多为复制粘贴带进来的）"
	case allSameByte(value):
		return "全部字符相同"
	case len(value) < MinProdSecretLen:
		return fmt.Sprintf("长度 %d，不足 %d", len(value), MinProdSecretLen)
	}
	return ""
}

func allSameByte(value string) bool {
	if len(value) < 2 {
		return true
	}
	for i := 1; i < len(value); i++ {
		if value[i] != value[0] {
			return false
		}
	}
	return true
}

// SecretFindings 现有配置里「能跑但不该上生产」的项，供 lab 环境启动提示；描述只含变量名与原因。
func (c *Config) SecretFindings() []string {
	var out []string
	if c.JWT.Secret != "" && isKnownWeakSecret(c.JWT.Secret) {
		out = append(out, "JWT_SECRET: 命中已知占位默认值")
	} else if p := secretShapeProblem(c.JWT.Secret); p != "" && p != "未设置" {
		out = append(out, "JWT_SECRET: "+p)
	}
	if isKnownWeakSecret(c.Database.Password) {
		out = append(out, "DATABASE_PASSWORD: 命中已知默认值")
	}
	if c.Supabase.ServiceRoleKey != "" && c.serviceRoleSource == ".env" {
		out = append(out, "SUPABASE_SERVICE_ROLE_KEY: 取自入库文件 .env，应改放 .env.local")
	}
	return out
}

// validateSecrets prod（IsLabAppEnv=false）硬校验；lab 环境放行（联调优先，弱值仅由 SecretFindings 提示）。
func validateSecrets(c *Config) error {
	if IsLabAppEnv(c.AppEnv) {
		return nil
	}
	if isKnownWeakSecret(c.JWT.Secret) {
		return fmt.Errorf("prod 启动校验失败：JWT_SECRET 仍是占位默认值，请换成 ≥%d 位随机串", MinProdSecretLen)
	}
	if p := secretShapeProblem(c.JWT.Secret); p != "" {
		return fmt.Errorf("prod 启动校验失败：JWT_SECRET %s", p)
	}
	if isKnownWeakSecret(c.Database.Password) || len(strings.TrimSpace(c.Database.Password)) < 8 {
		return fmt.Errorf("prod 启动校验失败：DATABASE_PASSWORD 为已知默认值或长度不足 8")
	}
	if c.Supabase.ServiceRoleKey != "" && c.serviceRoleSource == ".env" {
		return fmt.Errorf("prod 启动校验失败：SUPABASE_SERVICE_ROLE_KEY 取自 .env（该文件入库），请只经 OS env 或 .env.local 注入")
	}
	return nil
}
