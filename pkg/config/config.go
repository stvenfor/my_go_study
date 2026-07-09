// config.go 负责加载应用配置，支持 YAML 文件与环境变量覆盖。
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 聚合所有运行时配置项。
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Log      LogConfig      `mapstructure:"log"`
	Supabase SupabaseConfig `mapstructure:"supabase"`
}

// ServerConfig HTTP 服务配置。
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

// DatabaseConfig PostgreSQL 连接配置。
type DatabaseConfig struct {
	Host                   string `mapstructure:"host"`
	Port                   int    `mapstructure:"port"`
	User                   string `mapstructure:"user"`
	Password               string `mapstructure:"password"`
	DBName                 string `mapstructure:"dbname"`
	SSLMode                string `mapstructure:"sslmode"`
	MaxOpenConns           int    `mapstructure:"max_open_conns"`
	MaxIdleConns           int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetimeMinutes int    `mapstructure:"conn_max_lifetime_minutes"`
}

// RedisConfig Redis 连接配置。
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// JWTConfig JWT 签发配置。
type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

// SupabaseConfig Supabase 连接配置（与 Flutter my_ai_project 共用同一 Project）。
type SupabaseConfig struct {
	URL            string `mapstructure:"url"`
	AnonKey        string `mapstructure:"anon_key"`
	ServiceRoleKey string `mapstructure:"service_role_key"`
}

// Enabled 是否已配置 Supabase（用于启用 profile / transactions 路由）。
func (s SupabaseConfig) Enabled() bool {
	return s.URL != "" && s.AnonKey != ""
}

// DBKey 返回访问 PostgREST 的 API Key（优先 service_role）。
func (s SupabaseConfig) DBKey() string {
	if s.ServiceRoleKey != "" {
		return s.ServiceRoleKey
	}
	return s.AnonKey
}

// LogConfig 日志输出配置。
type LogConfig struct {
	Level      string `mapstructure:"level"`
	File       string `mapstructure:"file"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

// Load 读取配置文件并解析为 Config。
// configPath 传 configs 目录路径，env 传 dev/prod 等环境名。
func Load(configPath, env string) (*Config, error) {
	v := viper.New()
	v.AddConfigPath(configPath)
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取基础配置失败: %w", err)
	}

	if env != "" {
		v.SetConfigName("config." + env)
		if err := v.MergeInConfig(); err != nil {
			return nil, fmt.Errorf("读取环境配置 config.%s.yaml 失败: %w", env, err)
		}
	}

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 常用环境变量映射
	_ = v.BindEnv("server.port", "SERVER_PORT")
	_ = v.BindEnv("database.host", "DATABASE_HOST")
	_ = v.BindEnv("database.port", "DATABASE_PORT")
	_ = v.BindEnv("database.user", "DATABASE_USER")
	_ = v.BindEnv("database.password", "DATABASE_PASSWORD")
	_ = v.BindEnv("database.dbname", "DATABASE_DBNAME")
	_ = v.BindEnv("redis.addr", "REDIS_ADDR")
	_ = v.BindEnv("redis.password", "REDIS_PASSWORD")
	_ = v.BindEnv("jwt.secret", "JWT_SECRET")
	_ = v.BindEnv("supabase.url", "SUPABASE_URL")
	_ = v.BindEnv("supabase.anon_key", "SUPABASE_ANON_KEY", "SUPABASE_KEY")
	_ = v.BindEnv("supabase.service_role_key", "SUPABASE_SERVICE_ROLE_KEY")

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	// supabase.env / .env / .env.local 作为团队常量与本地覆盖默认值
	if err := applySupabaseDefaults(configPath, &cfg); err != nil {
		return nil, err
	}
	if err := applyLocalEnvDefaults(configPath, &cfg); err != nil {
		return nil, err
	}

	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("jwt.secret 不能为空")
	}
	if cfg.Database.ConnMaxLifetimeMinutes <= 0 {
		cfg.Database.ConnMaxLifetimeMinutes = 30
	}

	return &cfg, nil
}

// ResolveConfigDir 定位 configs 目录（支持从子目录或 IDE 非根目录启动）。
func ResolveConfigDir() string {
	const name = "configs"
	if wd, err := os.Getwd(); err == nil {
		dir := wd
		for {
			candidate := filepath.Join(dir, name)
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				return candidate
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return name
}

// applySupabaseDefaults 从 configs/supabase.env 填充仍为空的 Supabase 配置。
func applySupabaseDefaults(configPath string, cfg *Config) error {
	path := filepath.Join(configPath, "supabase.env")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	sv := viper.New()
	sv.SetConfigFile(path)
	sv.SetConfigType("env")
	if err := sv.ReadInConfig(); err != nil {
		return fmt.Errorf("读取 supabase.env 失败: %w", err)
	}
	if cfg.Supabase.URL == "" {
		cfg.Supabase.URL = sv.GetString("SUPABASE_URL")
	}
	if cfg.Supabase.AnonKey == "" {
		cfg.Supabase.AnonKey = sv.GetString("SUPABASE_ANON_KEY")
	}
	return nil
}

// applyLocalEnvDefaults 从项目根 .env / .env.local 填充仍为空的 Supabase service_role 等。
func applyLocalEnvDefaults(configPath string, cfg *Config) error {
	root := filepath.Dir(configPath)
	for _, name := range []string{".env", ".env.local"} {
		path := filepath.Join(root, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}
		sv := viper.New()
		sv.SetConfigFile(path)
		sv.SetConfigType("env")
		if err := sv.ReadInConfig(); err != nil {
			return fmt.Errorf("读取 %s 失败: %w", name, err)
		}
		if cfg.Supabase.ServiceRoleKey == "" {
			cfg.Supabase.ServiceRoleKey = sv.GetString("SUPABASE_SERVICE_ROLE_KEY")
		}
	}
	return nil
}

// DSN 生成 PostgreSQL 连接字符串。
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=Asia/Shanghai",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}

// ConnMaxLifetime 返回连接最大生命周期。
func (d DatabaseConfig) ConnMaxLifetime() time.Duration {
	return time.Duration(d.ConnMaxLifetimeMinutes) * time.Minute
}

// JWTExpire 返回 token 过期时长。
func (j JWTConfig) JWTExpire() time.Duration {
	if j.ExpireHours <= 0 {
		return 72 * time.Hour
	}
	return time.Duration(j.ExpireHours) * time.Hour
}
