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
	Server    ServerConfig    `mapstructure:"server"`
	GRPC      GRPCConfig      `mapstructure:"grpc"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Redis     RedisConfig     `mapstructure:"redis"`
	JWT       JWTConfig       `mapstructure:"jwt"`
	Auth      AuthConfig      `mapstructure:"auth"`
	Log       LogConfig       `mapstructure:"log"`
	Supabase  SupabaseConfig  `mapstructure:"supabase"`
	Realtime  RealtimeConfig  `mapstructure:"realtime"`
	Queue     QueueConfig     `mapstructure:"queue"`
	Scheduler SchedulerConfig `mapstructure:"scheduler"`
	SSE       SSEConfig       `mapstructure:"sse"`
	Community CommunityConfig `mapstructure:"community"`
}

// CommunityConfig 社区动态。
type CommunityConfig struct {
	AskEveryoneInviteUserIDs []string `mapstructure:"ask_everyone_invite_user_ids"`
}

// ServerConfig HTTP 服务配置。
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

// GRPCConfig gRPC 服务配置（与 HTTP 同进程双端口）。
type GRPCConfig struct {
	Enabled bool `mapstructure:"enabled"`
	Port    int  `mapstructure:"port"`
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

// AuthProviderLocal / AuthProviderSupabase 认证后端切换（见 docs/local-auth-postgres.md）。
const (
	AuthProviderLocal    = "local"
	AuthProviderSupabase = "supabase"
)

// AuthConfig 单设备登录等认证策略配置。
type AuthConfig struct {
	// Provider: local（本机 Postgres Auth）| supabase（Cloud GoTrue）；空则按 supabase 兼容旧行为。
	Provider                string   `mapstructure:"provider"`
	SessionTTLHours         int      `mapstructure:"session_ttl_hours"`
	SessionWhitelistUserIDs []string `mapstructure:"session_whitelist_user_ids"`
	SessionWhitelistEmails  []string `mapstructure:"session_whitelist_emails"`
	DevTestPhone            string   `mapstructure:"dev_test_phone"`
	DevTestOTP              string   `mapstructure:"dev_test_otp"`
	DevTestPassword         string   `mapstructure:"dev_test_password"`
}

// ProviderName 归一化认证后端名称。
func (a AuthConfig) ProviderName() string {
	switch strings.ToLower(strings.TrimSpace(a.Provider)) {
	case AuthProviderLocal:
		return AuthProviderLocal
	default:
		return AuthProviderSupabase
	}
}

// IsLocalProvider 是否使用本机 Auth + Postgres 业务库。
func (a AuthConfig) IsLocalProvider() bool {
	return a.ProviderName() == AuthProviderLocal
}

// SessionTTL 返回 Redis device session 有效期；0 表示永不过期（仅 logout / 互踢删除）。
func (a AuthConfig) SessionTTL() time.Duration {
	if a.SessionTTLHours <= 0 {
		return 0
	}
	return time.Duration(a.SessionTTLHours) * time.Hour
}

// IsSessionExempt 账号是否在单设备 session 白名单（可多设备同时在线）。
func (a AuthConfig) IsSessionExempt(userID, email string) bool {
	userID = strings.TrimSpace(userID)
	if userID != "" {
		for _, id := range a.SessionWhitelistUserIDs {
			if strings.TrimSpace(id) == userID {
				return true
			}
		}
	}
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	if normalizedEmail != "" {
		for _, item := range a.SessionWhitelistEmails {
			if strings.ToLower(strings.TrimSpace(item)) == normalizedEmail {
				return true
			}
		}
	}
	return false
}

// IsDevMode 是否为非生产运行模式（debug 等），用于启用测试手机号 bypass。
func IsDevMode(serverMode string) bool {
	return strings.ToLower(strings.TrimSpace(serverMode)) != "release"
}

// IsDevTestPhone 是否为配置的测试手机号（仅比较数字部分）。
func (a AuthConfig) IsDevTestPhone(phone string) bool {
	configured := NormalizePhoneDigits(a.DevTestPhone)
	if configured == "" {
		return false
	}
	return NormalizePhoneDigits(phone) == configured
}

// DevBypassEnabled 是否启用测试手机号 OTP bypass。
func (a AuthConfig) DevBypassEnabled(serverMode string) bool {
	return IsDevMode(serverMode) &&
		strings.TrimSpace(a.DevTestPhone) != "" &&
		strings.TrimSpace(a.DevTestOTP) != ""
}

// DevTestPasswordOrDefault 返回 Admin 创建测试用户用的内部密码。
func (a AuthConfig) DevTestPasswordOrDefault() string {
	if pwd := strings.TrimSpace(a.DevTestPassword); pwd != "" {
		return pwd
	}
	return "dev-test-phone-secret"
}

// NormalizePhoneDigits 规范化手机号为大陆 11 位数字。
func NormalizePhoneDigits(phone string) string {
	digits := strings.ReplaceAll(strings.TrimSpace(phone), " ", "")
	if strings.HasPrefix(digits, "+86") {
		digits = digits[3:]
	} else if strings.HasPrefix(digits, "86") && len(digits) == 13 {
		digits = digits[2:]
	}
	return digits
}

// ToE164China 格式化为 Supabase 要求的 E.164（+86 前缀）。
func ToE164China(phone string) string {
	return "+86" + NormalizePhoneDigits(phone)
}

// DevPhoneEmail 测试手机号映射邮箱（仅 dev bypass 使用）。
func DevPhoneEmail(digits string) string {
	return NormalizePhoneDigits(digits) + "@dev.test.local"
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

// QueueConfig 异步任务队列与多实例 WS 广播配置。
type QueueConfig struct {
	Enabled   bool              `mapstructure:"enabled"`
	PushAsync *bool             `mapstructure:"push_async"`
	Asynq     QueueAsynqConfig  `mapstructure:"asynq"`
	PubSub    QueuePubSubConfig `mapstructure:"pubsub"`
}

// QueueAsynqConfig Asynq 任务队列参数（后端复用 Redis）。
type QueueAsynqConfig struct {
	Concurrency int `mapstructure:"concurrency"`
}

// QueuePubSubConfig Redis Pub/Sub 多实例 WS 广播参数。
type QueuePubSubConfig struct {
	Channel string `mapstructure:"channel"`
}

// AsynqConcurrency 返回 Worker 并发数。
func (q QueueConfig) AsynqConcurrency() int {
	if q.Asynq.Concurrency <= 0 {
		return 10
	}
	return q.Asynq.Concurrency
}

// PubSubChannel 返回 Pub/Sub 广播频道名。
func (q QueueConfig) PubSubChannel() string {
	if ch := strings.TrimSpace(q.PubSub.Channel); ch != "" {
		return ch
	}
	return "realtime:fanout"
}

// UseAsyncPush 是否将 /realtime/push 异步入队（默认真；开发可设 push_async=false 同步直投 WS）。
func (q QueueConfig) UseAsyncPush() bool {
	if !q.Enabled {
		return false
	}
	if q.PushAsync == nil {
		return true
	}
	return *q.PushAsync
}

// SchedulerConfig 定时任务调度配置。
type SchedulerConfig struct {
	Enabled      bool               `mapstructure:"enabled"`
	Timezone     string             `mapstructure:"timezone"`
	HourlyNotify HourlyNotifyConfig `mapstructure:"hourly_notify"`
}

// HourlyNotifyConfig 每天 10:00–19:00 每小时系统通知。
type HourlyNotifyConfig struct {
	Enabled        bool                     `mapstructure:"enabled"`
	Cron           string                   `mapstructure:"cron"`
	TitleTemplate  string                   `mapstructure:"title_template"`
	BodyTemplate   string                   `mapstructure:"body_template"`
	DefaultMessage string                   `mapstructure:"default_message"`
	ExpiresMinutes int                      `mapstructure:"expires_minutes"`
	Action         HourlyNotifyActionConfig `mapstructure:"action"`
}

// HourlyNotifyActionConfig 通知点击行为配置。
type HourlyNotifyActionConfig struct {
	Type   string         `mapstructure:"type"`
	Route  string         `mapstructure:"route"`
	Params map[string]any `mapstructure:"params"`
	URL    string         `mapstructure:"url"`
}

// Location 返回调度时区。
func (s SchedulerConfig) Location() *time.Location {
	tz := strings.TrimSpace(s.Timezone)
	if tz == "" {
		tz = "Asia/Shanghai"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc, _ = time.LoadLocation("Asia/Shanghai")
	}
	return loc
}

// ExpiresMinutesOrDefault 返回通知过期分钟数。
func (h HourlyNotifyConfig) ExpiresMinutesOrDefault() int {
	if h.ExpiresMinutes <= 0 {
		return 120
	}
	return h.ExpiresMinutes
}

// CronSpec 返回 Cron 表达式。
func (h HourlyNotifyConfig) CronSpec() string {
	if spec := strings.TrimSpace(h.Cron); spec != "" {
		return spec
	}
	return "0 10-19 * * *"
}

// RealtimeConfig WebSocket Realtime 网关配置。
type RealtimeConfig struct {
	WsPath                string `mapstructure:"ws_path"`
	TicketTTLSeconds      int    `mapstructure:"ticket_ttl_seconds"`
	HeartbeatIntervalSec  int    `mapstructure:"heartbeat_interval_seconds"`
	MaxConnectionsPerUser int    `mapstructure:"max_connections_per_user"`
	EventRetention        int    `mapstructure:"event_retention"`
	PublicWSHost          string `mapstructure:"public_ws_host"`
}

// TicketTTL 返回 ticket 有效期。
func (r RealtimeConfig) TicketTTL() time.Duration {
	if r.TicketTTLSeconds <= 0 {
		return 120 * time.Second
	}
	return time.Duration(r.TicketTTLSeconds) * time.Second
}

// WSURL 构建 WebSocket 连接地址（供 ticket 接口返回）。
func (r RealtimeConfig) WSURL(httpPort int) string {
	path := r.WsPath
	if path == "" {
		path = "/realtime/v1/connect"
	}
	host := r.PublicWSHost
	if host == "" {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("ws://%s:%d%s", host, httpPort, path)
}

// SSEProviderMock / SSEProviderOpenAICompatible Provider 开关。
const (
	SSEProviderMock             = "mock"
	SSEProviderOpenAICompatible = "openai_compatible"
)

// SSEConfig AI 小石头 HTTP SSE 生成流配置。
type SSEConfig struct {
	Enabled                   bool            `mapstructure:"enabled"`
	Provider                  string          `mapstructure:"provider"` // mock | openai_compatible
	MaxPromptBytes            int             `mapstructure:"max_prompt_bytes"`
	MaxTokens                 int             `mapstructure:"max_tokens"`
	KeepaliveSeconds          int             `mapstructure:"keepalive_seconds"`
	RequestTimeoutSeconds     int             `mapstructure:"request_timeout_seconds"`
	RateLimitPerUserPerMinute int             `mapstructure:"rate_limit_per_user_per_minute"`
	ConversationTTLSeconds    int             `mapstructure:"conversation_ttl_seconds"`
	MaxTurns                  int             `mapstructure:"max_turns"`
	OpenAI                    SSEOpenAIConfig `mapstructure:"openai"`
}

// SSEOpenAIConfig OpenAI 兼容上游（密钥优先环境变量）。
type SSEOpenAIConfig struct {
	BaseURL string `mapstructure:"base_url"`
	Model   string `mapstructure:"model"`
	APIKey  string `mapstructure:"api_key"` // 建议仅 env：SSE_OPENAI_API_KEY
}

// ProviderName 归一化 Provider。
func (s SSEConfig) ProviderName() string {
	switch strings.ToLower(strings.TrimSpace(s.Provider)) {
	case SSEProviderOpenAICompatible:
		return SSEProviderOpenAICompatible
	default:
		return SSEProviderMock
	}
}

// ConversationTTL 停留会话滑动 TTL。
func (s SSEConfig) ConversationTTL() time.Duration {
	if s.ConversationTTLSeconds <= 0 {
		return 30 * time.Minute
	}
	return time.Duration(s.ConversationTTLSeconds) * time.Second
}

// MaxTurnsOrDefault 上下文最大轮数。
func (s SSEConfig) MaxTurnsOrDefault() int {
	if s.MaxTurns <= 0 {
		return 10
	}
	return s.MaxTurns
}

// MaxPromptBytesOrDefault prompt 字节上限。
func (s SSEConfig) MaxPromptBytesOrDefault() int {
	if s.MaxPromptBytes <= 0 {
		return 8192
	}
	return s.MaxPromptBytes
}

// MaxTokensOrDefault 生成 token 上限。
func (s SSEConfig) MaxTokensOrDefault() int {
	if s.MaxTokens <= 0 {
		return 2048
	}
	return s.MaxTokens
}

// KeepaliveOrDefault keepalive 注释帧间隔。
func (s SSEConfig) KeepaliveOrDefault() time.Duration {
	if s.KeepaliveSeconds <= 0 {
		return 15 * time.Second
	}
	return time.Duration(s.KeepaliveSeconds) * time.Second
}

// RequestTimeout 单次生成超时。
func (s SSEConfig) RequestTimeout() time.Duration {
	if s.RequestTimeoutSeconds <= 0 {
		return 120 * time.Second
	}
	return time.Duration(s.RequestTimeoutSeconds) * time.Second
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
	_ = v.BindEnv("grpc.enabled", "GRPC_ENABLED")
	_ = v.BindEnv("grpc.port", "GRPC_PORT")
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
	_ = v.BindEnv("auth.provider", "AUTH_PROVIDER")
	_ = v.BindEnv("auth.session_ttl_hours", "AUTH_SESSION_TTL_HOURS")
	_ = v.BindEnv("auth.session_whitelist_user_ids", "AUTH_SESSION_WHITELIST_USER_IDS")
	_ = v.BindEnv("auth.session_whitelist_emails", "AUTH_SESSION_WHITELIST_EMAILS")
	_ = v.BindEnv("auth.dev_test_phone", "AUTH_DEV_TEST_PHONE")
	_ = v.BindEnv("auth.dev_test_otp", "AUTH_DEV_TEST_OTP")
	_ = v.BindEnv("auth.dev_test_password", "AUTH_DEV_TEST_PASSWORD")
	_ = v.BindEnv("queue.enabled", "QUEUE_ENABLED")
	_ = v.BindEnv("queue.push_async", "QUEUE_PUSH_ASYNC")
	_ = v.BindEnv("queue.asynq.concurrency", "QUEUE_ASYNQ_CONCURRENCY")
	_ = v.BindEnv("queue.pubsub.channel", "QUEUE_PUBSUB_CHANNEL")
	_ = v.BindEnv("scheduler.enabled", "SCHEDULER_ENABLED")
	_ = v.BindEnv("scheduler.hourly_notify.enabled", "SCHEDULER_HOURLY_NOTIFY_ENABLED")
	_ = v.BindEnv("realtime.public_ws_host", "REALTIME_PUBLIC_WS_HOST")
	_ = v.BindEnv("sse.enabled", "SSE_ENABLED")
	_ = v.BindEnv("sse.provider", "SSE_PROVIDER")
	_ = v.BindEnv("sse.max_prompt_bytes", "SSE_MAX_PROMPT_BYTES")
	_ = v.BindEnv("sse.max_tokens", "SSE_MAX_TOKENS")
	_ = v.BindEnv("sse.keepalive_seconds", "SSE_KEEPALIVE_SECONDS")
	_ = v.BindEnv("sse.request_timeout_seconds", "SSE_REQUEST_TIMEOUT_SECONDS")
	_ = v.BindEnv("sse.rate_limit_per_user_per_minute", "SSE_RATE_LIMIT_PER_USER_PER_MINUTE")
	_ = v.BindEnv("sse.conversation_ttl_seconds", "SSE_CONVERSATION_TTL_SECONDS")
	_ = v.BindEnv("sse.max_turns", "SSE_MAX_TURNS")
	_ = v.BindEnv("sse.openai.base_url", "SSE_OPENAI_BASE_URL")
	_ = v.BindEnv("sse.openai.model", "SSE_OPENAI_MODEL")
	_ = v.BindEnv("sse.openai.api_key", "SSE_OPENAI_API_KEY")
	_ = v.BindEnv("community.ask_everyone_invite_user_ids", "COMMUNITY_ASK_EVERYONE_INVITE_USER_IDS")

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
	if cfg.GRPC.Port <= 0 {
		cfg.GRPC.Port = 9090
	}
	// 未显式配置时默认启用（dev/lan 试验）；可用 GRPC_ENABLED=false 关闭。
	if !v.IsSet("grpc.enabled") && os.Getenv("GRPC_ENABLED") == "" {
		cfg.GRPC.Enabled = true
	}
	if cfg.Realtime.WsPath == "" {
		cfg.Realtime.WsPath = "/realtime/v1/connect"
	}
	if cfg.Realtime.TicketTTLSeconds <= 0 {
		cfg.Realtime.TicketTTLSeconds = 120
	}
	if cfg.Realtime.MaxConnectionsPerUser <= 0 {
		cfg.Realtime.MaxConnectionsPerUser = 3
	}
	if cfg.Realtime.EventRetention <= 0 {
		cfg.Realtime.EventRetention = 200
	}
	if cfg.Queue.Asynq.Concurrency <= 0 {
		cfg.Queue.Asynq.Concurrency = 10
	}
	if strings.TrimSpace(cfg.Queue.PubSub.Channel) == "" {
		cfg.Queue.PubSub.Channel = "realtime:fanout"
	}
	if strings.TrimSpace(cfg.Scheduler.Timezone) == "" {
		cfg.Scheduler.Timezone = "Asia/Shanghai"
	}
	if strings.TrimSpace(cfg.Scheduler.HourlyNotify.Cron) == "" {
		cfg.Scheduler.HourlyNotify.Cron = "0 10-19 * * *"
	}
	if cfg.Scheduler.HourlyNotify.ExpiresMinutes <= 0 {
		cfg.Scheduler.HourlyNotify.ExpiresMinutes = 120
	}
	if cfg.Scheduler.HourlyNotify.TitleTemplate == "" {
		cfg.Scheduler.HourlyNotify.TitleTemplate = "整点提醒"
	}
	if cfg.Scheduler.HourlyNotify.BodyTemplate == "" {
		cfg.Scheduler.HourlyNotify.BodyTemplate = "现在是 {{hour}}:00，{{message}}"
	}
	if cfg.Scheduler.HourlyNotify.DefaultMessage == "" {
		cfg.Scheduler.HourlyNotify.DefaultMessage = "别错过重要消息"
	}
	if cfg.Scheduler.HourlyNotify.Action.Type == "" {
		cfg.Scheduler.HourlyNotify.Action.Type = "deeplink"
	}
	if cfg.Scheduler.HourlyNotify.Action.Route == "" {
		cfg.Scheduler.HourlyNotify.Action.Route = "/home"
	}
	applySSEDefaults(&cfg)
	applyAuthWhitelistEnv(&cfg.Auth)
	applyCommunityEnv(&cfg.Community)

	return &cfg, nil
}

func applySSEDefaults(cfg *Config) {
	// 未配置时默认启用 mock（与 Spec 一期一致）
	if !cfg.SSE.Enabled && cfg.SSE.Provider == "" && cfg.SSE.MaxPromptBytes == 0 {
		cfg.SSE.Enabled = true
	}
	if strings.TrimSpace(cfg.SSE.Provider) == "" {
		cfg.SSE.Provider = SSEProviderMock
	}
	if cfg.SSE.MaxPromptBytes <= 0 {
		cfg.SSE.MaxPromptBytes = 8192
	}
	if cfg.SSE.MaxTokens <= 0 {
		cfg.SSE.MaxTokens = 2048
	}
	if cfg.SSE.KeepaliveSeconds <= 0 {
		cfg.SSE.KeepaliveSeconds = 15
	}
	if cfg.SSE.RequestTimeoutSeconds <= 0 {
		cfg.SSE.RequestTimeoutSeconds = 120
	}
	if cfg.SSE.RateLimitPerUserPerMinute <= 0 {
		cfg.SSE.RateLimitPerUserPerMinute = 20
	}
	if cfg.SSE.ConversationTTLSeconds <= 0 {
		cfg.SSE.ConversationTTLSeconds = 1800
	}
	if cfg.SSE.MaxTurns <= 0 {
		cfg.SSE.MaxTurns = 10
	}
	if strings.TrimSpace(cfg.SSE.OpenAI.BaseURL) == "" {
		cfg.SSE.OpenAI.BaseURL = "https://api.openai.com/v1"
	}
	if strings.TrimSpace(cfg.SSE.OpenAI.Model) == "" {
		cfg.SSE.OpenAI.Model = "gpt-4o-mini"
	}
	if key := strings.TrimSpace(os.Getenv("SSE_OPENAI_API_KEY")); key != "" {
		cfg.SSE.OpenAI.APIKey = key
	}
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

// applyAuthWhitelistEnv 用逗号分隔的环境变量覆盖 session 白名单（便于生产注入）。
func applyAuthWhitelistEnv(auth *AuthConfig) {
	if raw := strings.TrimSpace(os.Getenv("AUTH_SESSION_WHITELIST_USER_IDS")); raw != "" {
		auth.SessionWhitelistUserIDs = splitCommaTrimmed(raw)
	}
	if raw := strings.TrimSpace(os.Getenv("AUTH_SESSION_WHITELIST_EMAILS")); raw != "" {
		auth.SessionWhitelistEmails = splitCommaTrimmed(raw)
	}
}

func applyCommunityEnv(c *CommunityConfig) {
	if raw := strings.TrimSpace(os.Getenv("COMMUNITY_ASK_EVERYONE_INVITE_USER_IDS")); raw != "" {
		c.AskEveryoneInviteUserIDs = splitCommaTrimmed(raw)
	}
}

func splitCommaTrimmed(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
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
