// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 config
package config

// import 多行导入块开始
import (
	// 语句或表达式
	"fmt"
	// 语句或表达式
	"io"
	// 语句或表达式
	"log"
	// 语句或表达式
	"os"
	// 语句或表达式
	"time"

	// 语句或表达式
	"github.com/fsnotify/fsnotify"
	// 语句或表达式
	"github.com/spf13/pflag"
	// 语句或表达式
	"github.com/spf13/viper"
	// 闭合括号（import/var/参数列表等）
)

// 变量声明
var conf = new(Config)

// 定义结构体类型 Config
type Config struct {
	// 语句或表达式
	Pay *Pay `mapstructure:"pay"`
	// 语句或表达式
	Server *Server `mapstructure:"server"`
	// 语句或表达式
	Cache *Cache `mapstructure:"cache"`
	// 语句或表达式
	Upload *Upload `mapstructure:"upload"`
	// 语句或表达式
	Qiniu *Qiniu `mapstructure:"qiniu"`
	// 语句或表达式
	Aliyun *Aliyun `mapstructure:"aliyun"`
	// 语句或表达式
	DB *DB `mapstructure:"db"`
	// 语句或表达式
	Auth *Auth `mapstructure:"auth"`
	// 语句或表达式
	Wx *Wx `mapstructure:"wx"`
	// 语句或表达式
	Jwt *Jwt `mapstructure:"jwt"`
	// 语句或表达式
	Email *Email `mapstructure:"email"`
	// 语句或表达式
	Log *LogConfig `mapstructure:"log"`
	// 语句或表达式
	Elasticsearch *Elasticsearch `mapstructure:"elasticsearch"`
	// 语句或表达式
	Milvus *Milvus `mapstructure:"milvus"`
	// 语句或表达式
}

// 定义结构体类型 Email
type Email struct {
	// 语句或表达式
	Host *string `mapstructure:"host"`
	// 语句或表达式
	Port *int `mapstructure:"port"`
	// 语句或表达式
	Username *string `mapstructure:"username"`
	// 语句或表达式
	Password *string `mapstructure:"password"`
	// 语句或表达式
	Identity *string `mapstructure:"identity"`
	// 语句或表达式
	From *string `mapstructure:"from"`
	// 语句或表达式
	BaseURL *string `mapstructure:"baseUrl"`
	// 语句或表达式
}

// 定义函数 GetHost（签名开始）
func (e *Email) GetHost() string {
	// 条件判断 if
	if e == nil || e.Host == nil {
		// 返回语句
		return "localhost"
		// 语句或表达式
	}
	// 返回语句
	return *e.Host
	// 语句或表达式
}

// 定义函数 GetPort（签名开始）
func (e *Email) GetPort() int {
	// 条件判断 if
	if e == nil || e.Port == nil {
		// 返回语句
		return 587
		// 语句或表达式
	}
	// 返回语句
	return *e.Port
	// 语句或表达式
}

// 定义函数 GetUsername（签名开始）
func (e *Email) GetUsername() string {
	// 条件判断 if
	if e == nil || e.Username == nil {
		// 返回语句
		return ""
		// 语句或表达式
	}
	// 返回语句
	return *e.Username
	// 语句或表达式
}

// 定义函数 GetPassword（签名开始）
func (e *Email) GetPassword() string {
	// 条件判断 if
	if e == nil || e.Password == nil {
		// 返回语句
		return ""
		// 语句或表达式
	}
	// 返回语句
	return *e.Password
	// 语句或表达式
}

// 定义函数 GetIdentity（签名开始）
func (e *Email) GetIdentity() string {
	// 条件判断 if
	if e == nil || e.Identity == nil {
		// 返回语句
		return ""
		// 语句或表达式
	}
	// 返回语句
	return *e.Identity
	// 语句或表达式
}

// 定义函数 GetFrom（签名开始）
func (e *Email) GetFrom() string {
	// 条件判断 if
	if e == nil || e.From == nil {
		// 返回语句
		return ""
		// 语句或表达式
	}
	// 返回语句
	return *e.From
	// 语句或表达式
}

// 定义函数 GetBaseURL（签名开始）
func (e *Email) GetBaseURL() string {
	// 条件判断 if
	if e == nil || e.BaseURL == nil {
		// 返回语句
		return ""
		// 语句或表达式
	}
	// 返回语句
	return *e.BaseURL
	// 语句或表达式
}

// 定义结构体类型 Jwt
type Jwt struct {
	// 语句或表达式
	Secret *string `mapstructure:"secret"`
	// 语句或表达式
	Expire *time.Duration `mapstructure:"expire"`
	// 语句或表达式
	Refresh *time.Duration `mapstructure:"refresh"`
	// 语句或表达式
}

// 定义函数 GetSecret（签名开始）
func (j *Jwt) GetSecret() string {
	// 条件判断 if
	if j == nil || j.Secret == nil {
		// 返回语句
		return ""
		// 语句或表达式
	}
	// 返回语句
	return *j.Secret
	// 语句或表达式
}

// 定义函数 GetExpire（签名开始）
func (j *Jwt) GetExpire() time.Duration {
	// 条件判断 if
	if j == nil || j.Expire == nil {
		// 返回语句
		return 24 * time.Hour
		// 语句或表达式
	}
	// 返回语句
	return *j.Expire
	// 语句或表达式
}

// 定义函数 GetRefresh（签名开始）
func (j *Jwt) GetRefresh() time.Duration {
	// 条件判断 if
	if j == nil || j.Refresh == nil {
		// 返回语句
		return 7 * 24 * time.Hour
		// 语句或表达式
	}
	// 返回语句
	return *j.Refresh
	// 语句或表达式
}

// 定义结构体类型 Pay
type Pay struct {
	// 语句或表达式
	WxPay *WxPay `mapstructure:"wxPay"`
	// 语句或表达式
}

// 定义结构体类型 WxPay
type WxPay struct {
	// 语句或表达式
	AppId *string `mapstructure:"appId"`
	// 语句或表达式
	MchId *string `mapstructure:"mchId"` //商户证书的证书序列号
	// 语句或表达式
	MchSerialNo *string `mapstructure:"mchSerialNo"` //商户证书的证书序列号
	// 语句或表达式
	ApiV3Key *string `mapstructure:"apiV3Key"` //apiV3Key，商户平台获取
	// 语句或表达式
	PrivateKey *string `mapstructure:"privateKey"` //私钥 apiclient_key.pem 读取后的内容
	// 语句或表达式
	AppSecret *string `mapstructure:"appSecret"`
	// 语句或表达式
	NotifyUrl *string `mapstructure:"notifyUrl"`
	// 语句或表达式
	MchCertPath *string `mapstructure:"mchCertPath"`
	// 语句或表达式
	MchKeyPath *string `mapstructure:"mchKeyPath"`
	// 语句或表达式
}

// 定义结构体类型 DB
type DB struct {
	// 语句或表达式
	Redis *Redis `mapstructure:"redis"`
	// 语句或表达式
	Mysql *Mysql `mapstructure:"mysql"`
	// 语句或表达式
	Postgres *Postgres `mapstructure:"postgres"`
	// 语句或表达式
}

// 定义结构体类型 Server
type Server struct {
	// 语句或表达式
	Port *int `mapstructure:"port"`
	// 语句或表达式
	Cros []string `mapstructure:"cros"`
	// 语句或表达式
	AllowOrigins []string `mapstructure:"allowOrigins"`
	// 语句或表达式
	Mode *string `mapstructure:"mode"`
	// 语句或表达式
	Name *string `mapstructure:"name"`
	// 语句或表达式
	Version *string `mapstructure:"version"`
	// 语句或表达式
	Host *string `mapstructure:"host"`
	// 语句或表达式
	ReadTimeout *time.Duration `mapstructure:"readTimeout"`
	// 语句或表达式
	WriteTimeout *time.Duration `mapstructure:"writeTimeout"`
	// 语句或表达式
}

// 定义结构体类型 LogConfig
type LogConfig struct {
	// 语句或表达式
	Level *string `mapstructure:"level"`
	// 语句或表达式
	Format *string `mapstructure:"format"`
	// 语句或表达式
	AddSource *bool `mapstructure:"addSource"`
	// 语句或表达式
	Filename *string `mapstructure:"filename"`
	// 语句或表达式
	MaxSize *int `mapstructure:"maxSize"`
	// 语句或表达式
	MaxAge *int `mapstructure:"maxAge"`
	// 语句或表达式
	MaxBackups *int `mapstructure:"maxBackups"`
	// 语句或表达式
	Output io.Writer `mapstructure:"output"`
	// 语句或表达式
}

// 定义函数 GetLevel（签名开始）
func (l *LogConfig) GetLevel() string {
	// 条件判断 if
	if l == nil || l.Level == nil {
		// 返回语句
		return "info"
		// 语句或表达式
	}
	// 返回语句
	return *l.Level
	// 语句或表达式
}

// 定义函数 GetFormat（签名开始）
func (l *LogConfig) GetFormat() string {
	// 条件判断 if
	if l == nil || l.Format == nil {
		// 返回语句
		return "json"
		// 语句或表达式
	}
	// 返回语句
	return *l.Format
	// 语句或表达式
}

// 定义函数 GetAddSource（签名开始）
func (l *LogConfig) GetAddSource() bool {
	// 条件判断 if
	if l == nil || l.AddSource == nil {
		// 返回语句
		return false
		// 语句或表达式
	}
	// 返回语句
	return *l.AddSource
	// 语句或表达式
}

// 定义函数 GetFilename（签名开始）
func (l *LogConfig) GetFilename() string {
	// 条件判断 if
	if l == nil || l.Filename == nil {
		// 返回语句
		return ""
		// 语句或表达式
	}
	// 返回语句
	return *l.Filename
	// 语句或表达式
}

// 定义函数 GetMaxSize（签名开始）
func (l *LogConfig) GetMaxSize() int {
	// 条件判断 if
	if l == nil || l.MaxSize == nil {
		// 返回语句
		return 100
		// 语句或表达式
	}
	// 返回语句
	return *l.MaxSize
	// 语句或表达式
}

// 定义函数 GetMaxAge（签名开始）
func (l *LogConfig) GetMaxAge() int {
	// 条件判断 if
	if l == nil || l.MaxAge == nil {
		// 返回语句
		return 30
		// 语句或表达式
	}
	// 返回语句
	return *l.MaxAge
	// 语句或表达式
}

// 定义函数 GetMaxBackups（签名开始）
func (l *LogConfig) GetMaxBackups() int {
	// 条件判断 if
	if l == nil || l.MaxBackups == nil {
		// 返回语句
		return 3
		// 语句或表达式
	}
	// 返回语句
	return *l.MaxBackups
	// 语句或表达式
}

// 定义结构体类型 Redis
type Redis struct {
	// 语句或表达式
	Addr *string `mapstructure:"addr"`
	// 语句或表达式
	Password *string `mapstructure:"password"`
	// 语句或表达式
	DB *int `mapstructure:"db"`
	// 语句或表达式
	PoolSize *int `mapstructure:"poolSize"`
	// 语句或表达式
	IdleTimeout *int `mapstructure:"idleTimeout"`
	// 语句或表达式
	MaxOpenConns *int `mapstructure:"maxOpenConns"`
	// 语句或表达式
	MaxIdleConns *int `mapstructure:"maxIdleConns"`
	// 语句或表达式
}

// 定义结构体类型 Mysql
type Mysql struct {
	// 语句或表达式
	Host *string `mapstructure:"host"`
	// 语句或表达式
	Port *int `mapstructure:"port"`
	// 语句或表达式
	User *string `mapstructure:"user"`
	// 语句或表达式
	Password *string `mapstructure:"password"`
	// 语句或表达式
	Database *string `mapstructure:"database"`
	// 语句或表达式
	MaxIdleConns *int `mapstructure:"maxIdleConns"`
	// 语句或表达式
	PingTimeout *time.Duration `mapstructure:"pingTimeout"`
	// 语句或表达式
	MaxOpenConns *int `mapstructure:"maxOpenConns"`
	// 语句或表达式
}

// 定义结构体类型 Cache
type Cache struct {
	// 语句或表达式
	NeedCache []string `mapstructure:"needCache"`
	// 语句或表达式
	Expire *int64 `mapstructure:"expire"` //单位秒
	// 语句或表达式
}

// 定义函数 GetExpire（签名开始）
func (c *Cache) GetExpire() int64 {
	// 条件判断 if
	if c == nil || c.Expire == nil {
		// 返回语句
		return 3600 // 默认1小时
		// 语句或表达式
	}
	// 返回语句
	return *c.Expire
	// 语句或表达式
}

// 定义函数 GetNeedCache（签名开始）
func (c *Cache) GetNeedCache() []string {
	// 条件判断 if
	if c == nil || c.NeedCache == nil {
		// 返回语句
		return []string{}
		// 语句或表达式
	}
	// 返回语句
	return c.NeedCache
	// 语句或表达式
}

// 定义结构体类型 Upload
type Upload struct {
	// 语句或表达式
	Prefix *string `mapstructure:"prefix"`
	// 语句或表达式
}

// 定义结构体类型 Qiniu
type Qiniu struct {
	// 语句或表达式
	Bucket *string `mapstructure:"bucket"`
	// 语句或表达式
	AccessKey *string `mapstructure:"accessKey"`
	// 语句或表达式
	SecretKey *string `mapstructure:"secretKey"`
	// 语句或表达式
	Region *string `mapstructure:"region"`
	// 语句或表达式
}

// Aliyun 阿里云配置
// 定义结构体类型 Aliyun
type Aliyun struct {
	// 语句或表达式
	AccessKeyID *string `mapstructure:"accessKeyId"`
	// 语句或表达式
	AccessKeySecret *string `mapstructure:"accessKeySecret"`
	// 语句或表达式
	Endpoint *string `mapstructure:"endpoint"`
	// 语句或表达式
	Bucket *string `mapstructure:"bucket"`
	// 语句或表达式
}

// 定义结构体类型 Auth
type Auth struct {
	// 语句或表达式
	IsAuth *bool `mapstructure:"isAuth"`
	// 语句或表达式
	Ignores []string `mapstructure:"ignores"`
	// 语句或表达式
	NeedLogins []string `mapstructure:"needLogins"`
	// 语句或表达式
}

// 定义结构体类型 Wx
type Wx struct {
	// 语句或表达式
	AppId *string `mapstructure:"appId"`
	// 语句或表达式
	Secret *string `mapstructure:"secret"`
	// 语句或表达式
	Token *string `mapstructure:"token"`
	// 语句或表达式
	AesKey *string `mapstructure:"aesKey"`
	// 语句或表达式
}

// 定义结构体类型 Postgres
type Postgres struct {
	// 语句或表达式
	Host *string `mapstructure:"host"`
	// 语句或表达式
	Port *int `mapstructure:"port"`
	// 语句或表达式
	User *string `mapstructure:"user"`
	// 语句或表达式
	Password *string `mapstructure:"password"`
	// 语句或表达式
	Database *string `mapstructure:"database"`
	// 语句或表达式
	SSLMode *string `mapstructure:"sslmode"`
	// 语句或表达式
	MaxIdleConns *int `mapstructure:"maxIdleConns"`
	// 语句或表达式
	PingTimeout *time.Duration `mapstructure:"pingTimeout"`
	// 语句或表达式
	MaxOpenConns *int `mapstructure:"maxOpenConns"`
	// 语句或表达式
}

// Elasticsearch 配置结构体
// 定义结构体类型 Elasticsearch
type Elasticsearch struct {
	// 语句或表达式
	Addresses []string `mapstructure:"addresses"`
	// 语句或表达式
	Username *string `mapstructure:"username"`
	// 语句或表达式
	Password *string `mapstructure:"password"`
	// 语句或表达式
	APIKey *string `mapstructure:"apiKey"`
	// 语句或表达式
}

// GetAddresses 获取 ES 地址列表
// 定义函数 GetAddresses（签名开始）
func (e *Elasticsearch) GetAddresses() []string {
	// 条件判断 if
	if e == nil || len(e.Addresses) == 0 {
		// 返回语句
		return []string{"http://localhost:9200"}
		// 语句或表达式
	}
	// 返回语句
	return e.Addresses
	// 语句或表达式
}

// GetUsername 获取 ES 用户名
// 定义函数 GetUsername（签名开始）
func (e *Elasticsearch) GetUsername() string {
	// 条件判断 if
	if e == nil || e.Username == nil {
		// 返回语句
		return ""
		// 语句或表达式
	}
	// 返回语句
	return *e.Username
	// 语句或表达式
}

// GetPassword 获取 ES 密码
// 定义函数 GetPassword（签名开始）
func (e *Elasticsearch) GetPassword() string {
	// 条件判断 if
	if e == nil || e.Password == nil {
		// 返回语句
		return ""
		// 语句或表达式
	}
	// 返回语句
	return *e.Password
	// 语句或表达式
}

// GetAPIKey 获取 ES API Key
// 定义函数 GetAPIKey（签名开始）
func (e *Elasticsearch) GetAPIKey() string {
	// 条件判断 if
	if e == nil || e.APIKey == nil {
		// 返回语句
		return ""
		// 语句或表达式
	}
	// 返回语句
	return *e.APIKey
	// 语句或表达式
}

// Milvus 配置结构体
// 定义结构体类型 Milvus
type Milvus struct {
	// 语句或表达式
	Address *string `mapstructure:"address"`
	// 语句或表达式
	DBName *string `mapstructure:"dbName"`
	// 语句或表达式
	Username *string `mapstructure:"username"`
	// 语句或表达式
	Password *string `mapstructure:"password"`
	// 语句或表达式
}

// GetAddress 获取 Milvus 地址
// 定义函数 GetAddress（签名开始）
func (m *Milvus) GetAddress() string {
	// 条件判断 if
	if m == nil || m.Address == nil {
		// 返回语句
		return "localhost:19530"
		// 语句或表达式
	}
	// 返回语句
	return *m.Address
	// 语句或表达式
}

// GetDBName 获取 Milvus 数据库名称
// 定义函数 GetDBName（签名开始）
func (m *Milvus) GetDBName() string {
	// 条件判断 if
	if m == nil || m.DBName == nil {
		// 返回语句
		return ""
		// 语句或表达式
	}
	// 返回语句
	return *m.DBName
	// 语句或表达式
}

// GetUsername 获取 Milvus 用户名
// 定义函数 GetUsername（签名开始）
func (m *Milvus) GetUsername() string {
	// 条件判断 if
	if m == nil || m.Username == nil {
		// 返回语句
		return ""
		// 语句或表达式
	}
	// 返回语句
	return *m.Username
	// 语句或表达式
}

// GetPassword 获取 Milvus 密码
// 定义函数 GetPassword（签名开始）
func (m *Milvus) GetPassword() string {
	// 条件判断 if
	if m == nil || m.Password == nil {
		// 返回语句
		return ""
		// 语句或表达式
	}
	// 返回语句
	return *m.Password
	// 语句或表达式
}

// 定义函数 GetHost（签名开始）
func (s *Server) GetHost() string {
	// 条件判断 if
	if s == nil || s.Host == nil {
		// 返回语句
		return "127.0.0.1"
		// 语句或表达式
	}
	// 返回语句
	return *s.Host
	// 语句或表达式
}

// 定义函数 GetPort（签名开始）
func (s *Server) GetPort() int {
	// 条件判断 if
	if s == nil || s.Port == nil {
		// 返回语句
		return 8080
		// 语句或表达式
	}
	// 返回语句
	return *s.Port
	// 语句或表达式
}

// 定义函数 GetMode（签名开始）
func (s *Server) GetMode() string {
	// 条件判断 if
	if s == nil || s.Mode == nil {
		// 返回语句
		return "release"
		// 语句或表达式
	}
	// 返回语句
	return *s.Mode
	// 语句或表达式
}

// 定义函数 GetReadTimeout（签名开始）
func (s *Server) GetReadTimeout() time.Duration {
	// 条件判断 if
	if s == nil || s.ReadTimeout == nil {
		// 返回语句
		return 5 * time.Second
		// 语句或表达式
	}
	// 返回语句
	return *s.ReadTimeout
	// 语句或表达式
}

// 定义函数 GetWriteTimeout（签名开始）
func (s *Server) GetWriteTimeout() time.Duration {
	// 条件判断 if
	if s == nil || s.WriteTimeout == nil {
		// 返回语句
		return 5 * time.Second
		// 语句或表达式
	}
	// 返回语句
	return *s.WriteTimeout
	// 语句或表达式
}

// 定义函数 GetCros（签名开始）
func (s *Server) GetCros() []string {
	// 条件判断 if
	if s == nil || s.Cros == nil {
		// 返回语句
		return []string{}
		// 语句或表达式
	}
	// 返回语句
	return s.Cros
	// 语句或表达式
}

// 定义函数 GetAppId（签名开始）
func (w *WxPay) GetAppId() string {
	// 条件判断 if
	if w == nil || w.AppId == nil {
		// 返回语句
		return ""
		// 语句或表达式
	}
	// 返回语句
	return *w.AppId
	// 语句或表达式
}

// 定义函数 GetMchId（签名开始）
func (w *WxPay) GetMchId() string {
	// 条件判断 if
	if w == nil || w.MchId == nil {
		// 返回语句
		return ""
		// 语句或表达式
	}
	// 返回语句
	return *w.MchId
	// 语句或表达式
}

// 定义函数 GetNotifyUrl（签名开始）
func (w *WxPay) GetNotifyUrl() string {
	// 条件判断 if
	if w == nil || w.NotifyUrl == nil {
		// 返回语句
		return ""
		// 语句或表达式
	}
	// 返回语句
	return *w.NotifyUrl
	// 语句或表达式
}

// 定义函数 GetHost（签名开始）
func (p *Postgres) GetHost() string {
	// 条件判断 if
	if p == nil || p.Host == nil {
		// 返回语句
		return "127.0.0.1"
		// 语句或表达式
	}
	// 返回语句
	return *p.Host
	// 语句或表达式
}

// 定义函数 GetPort（签名开始）
func (p *Postgres) GetPort() int {
	// 条件判断 if
	if p == nil || p.Port == nil {
		// 返回语句
		return 5432
		// 语句或表达式
	}
	// 返回语句
	return *p.Port
	// 语句或表达式
}

// 定义函数 GetDatabase（签名开始）
func (p *Postgres) GetDatabase() string {
	// 条件判断 if
	if p == nil || p.Database == nil {
		// 返回语句
		return ""
		// 语句或表达式
	}
	// 返回语句
	return *p.Database
	// 语句或表达式
}

// 定义函数 GetUser（签名开始）
func (p *Postgres) GetUser() string {
	// 条件判断 if
	if p == nil || p.User == nil {
		// 返回语句
		return ""
		// 语句或表达式
	}
	// 返回语句
	return *p.User
	// 语句或表达式
}

// 定义函数 GetPassword（签名开始）
func (p *Postgres) GetPassword() string {
	// 条件判断 if
	if p == nil || p.Password == nil {
		// 返回语句
		return ""
		// 语句或表达式
	}
	// 返回语句
	return *p.Password
	// 语句或表达式
}

// 定义函数 GetSSLMode（签名开始）
func (p *Postgres) GetSSLMode() string {
	// 条件判断 if
	if p == nil || p.SSLMode == nil {
		// 返回语句
		return "disable"
		// 语句或表达式
	}
	// 返回语句
	return *p.SSLMode
	// 语句或表达式
}

// 定义函数 GetMaxIdleConns（签名开始）
func (p *Postgres) GetMaxIdleConns() int {
	// 条件判断 if
	if p == nil || p.MaxIdleConns == nil {
		// 返回语句
		return 10
		// 语句或表达式
	}
	// 返回语句
	return *p.MaxIdleConns
	// 语句或表达式
}

// 定义函数 GetPingTimeout（签名开始）
func (p *Postgres) GetPingTimeout() time.Duration {
	// 条件判断 if
	if p == nil || p.PingTimeout == nil {
		// 返回语句
		return 5 * time.Second
		// 语句或表达式
	}
	// 返回语句
	return *p.PingTimeout
	// 语句或表达式
}

// 定义函数 GetMaxOpenConns（签名开始）
func (p *Postgres) GetMaxOpenConns() int {
	// 条件判断 if
	if p == nil || p.MaxOpenConns == nil {
		// 返回语句
		return 100
		// 语句或表达式
	}
	// 返回语句
	return *p.MaxOpenConns
	// 语句或表达式
}

// 定义函数 GetAddr（签名开始）
func (r *Redis) GetAddr() string {
	// 条件判断 if
	if r == nil || r.Addr == nil {
		// 返回语句
		return "127.0.0.1:6379"
		// 语句或表达式
	}
	// 返回语句
	return *r.Addr
	// 语句或表达式
}

// 定义函数 GetDB（签名开始）
func (r *Redis) GetDB() int {
	// 条件判断 if
	if r == nil || r.DB == nil {
		// 返回语句
		return 0
		// 语句或表达式
	}
	// 返回语句
	return *r.DB
	// 语句或表达式
}

// 定义函数 GetPassword（签名开始）
func (r *Redis) GetPassword() string {
	// 条件判断 if
	if r == nil || r.Password == nil {
		// 返回语句
		return ""
		// 语句或表达式
	}
	// 返回语句
	return *r.Password
	// 语句或表达式
}

// 定义函数 GetPoolSize（签名开始）
func (r *Redis) GetPoolSize() int {
	// 条件判断 if
	if r == nil || r.PoolSize == nil {
		// 返回语句
		return 100
		// 语句或表达式
	}
	// 返回语句
	return *r.PoolSize
	// 语句或表达式
}

// 定义函数 GetMaxIdleConns（签名开始）
func (r *Redis) GetMaxIdleConns() int {
	// 条件判断 if
	if r == nil || r.MaxIdleConns == nil {
		// 返回语句
		return 10
		// 语句或表达式
	}
	// 返回语句
	return *r.MaxIdleConns
	// 语句或表达式
}

// 定义函数 GetMaxOpenConns（签名开始）
func (r *Redis) GetMaxOpenConns() int {
	// 条件判断 if
	if r == nil || r.MaxOpenConns == nil {
		// 返回语句
		return 100
		// 语句或表达式
	}
	// 返回语句
	return *r.MaxOpenConns
	// 语句或表达式
}

// 定义函数 GetJwt（签名开始）
func (c *Config) GetJwt() *Jwt {
	// 条件判断 if
	if c == nil {
		// 返回语句
		return nil
		// 语句或表达式
	}
	// 返回语句
	return c.Jwt
	// 语句或表达式
}

// 定义函数 GetAliyun（签名开始）
func (c *Config) GetAliyun() *Aliyun {
	// 条件判断 if
	if c == nil {
		// 返回语句
		return nil
		// 语句或表达式
	}
	// 返回语句
	return c.Aliyun
	// 语句或表达式
}

// 定义函数 GetAccessKeyID（签名开始）
func (a *Aliyun) GetAccessKeyID() string {
	// 条件判断 if
	if a == nil || a.AccessKeyID == nil {
		// 返回语句
		return ""
		// 语句或表达式
	}
	// 返回语句
	return *a.AccessKeyID
	// 语句或表达式
}

// 定义函数 GetAccessKeySecret（签名开始）
func (a *Aliyun) GetAccessKeySecret() string {
	// 条件判断 if
	if a == nil || a.AccessKeySecret == nil {
		// 返回语句
		return ""
		// 语句或表达式
	}
	// 返回语句
	return *a.AccessKeySecret
	// 语句或表达式
}

// 定义函数 GetEndpoint（签名开始）
func (a *Aliyun) GetEndpoint() string {
	// 条件判断 if
	if a == nil || a.Endpoint == nil {
		// 返回语句
		return ""
		// 语句或表达式
	}
	// 返回语句
	return *a.Endpoint
	// 语句或表达式
}

// 定义函数 GetBucket（签名开始）
func (a *Aliyun) GetBucket() string {
	// 条件判断 if
	if a == nil || a.Bucket == nil {
		// 返回语句
		return ""
		// 语句或表达式
	}
	// 返回语句
	return *a.Bucket
	// 语句或表达式
}

// 定义函数 GetIsAuth（签名开始）
func (a *Auth) GetIsAuth() bool {
	// 条件判断 if
	if a == nil || a.IsAuth == nil {
		// 返回语句
		return false
		// 语句或表达式
	}
	// 返回语句
	return *a.IsAuth
	// 语句或表达式
}

// 定义函数 GetIgnores（签名开始）
func (a *Auth) GetIgnores() []string {
	// 条件判断 if
	if a == nil || a.Ignores == nil {
		// 返回语句
		return []string{}
		// 语句或表达式
	}
	// 返回语句
	return a.Ignores
	// 语句或表达式
}

// 定义函数 InitConfig（签名开始）
func InitConfig() {
	// 短变量声明并赋值
	workDir, _ := os.Getwd()
	// 语句或表达式
	viper.SetConfigName("config")
	// 语句或表达式
	viper.SetConfigType("yml")
	// 语句或表达式
	viper.AddConfigPath(workDir + "/etc")
	// 短变量声明并赋值
	err := viper.ReadInConfig()
	// 条件判断 if
	if err != nil {
		// 语句或表达式
		panic(err)
		// 语句或表达式
	}
	// 赋值语句
	err = viper.Unmarshal(conf)
	// 条件判断 if
	if err != nil {
		// 语句或表达式
		log.Fatalf("config unmarshal failed, err:%v", err)
		// 语句或表达式
	}
	// 语句或表达式
	viper.WatchConfig()
	// 语句或表达式
	viper.OnConfigChange(func(in fsnotify.Event) {
		// 赋值语句
		err = viper.Unmarshal(conf)
		// 条件判断 if
		if err != nil {
			// 语句或表达式
			log.Fatalf("config unmarshal failed, err:%v", err)
			// 语句或表达式
		}
		// 语句或表达式
	})
	// 语句或表达式
}

// Init 函数负责初始化配置
// 它会解析命令行参数，读取配置文件，并反序列化到 Config 结构体中
// 定义函数 Init（签名开始）
func Init() *viper.Viper {
	// 1. 设置命令行参数
	// 我们可以通过 -c 或 --config 来指定配置文件
	// 变量声明
	var configFile = pflag.StringP("config", "c", "etc/config.yml", "Path to the config file (e.g., etc/config.yml)")
	// 语句或表达式
	pflag.Parse()

	// 2. 初始化 Viper
	// 短变量声明并赋值
	v := viper.New()
	// 语句或表达式
	v.SetDefault("app.name", "MyDefaultAppName")

	// 语句或表达式
	v.SetDefault("server.mode", "release")
	// 语句或表达式
	v.SetDefault("server.host", "127.0.0.1")
	// 语句或表达式
	v.SetDefault("server.port", 8080)
	// 语句或表达式
	v.SetDefault("server.readTimeout", "5s")
	// 语句或表达式
	v.SetDefault("server.writeTimeout", "5s")

	// 语句或表达式
	v.SetDefault("log.level", "info")
	// 语句或表达式
	v.SetDefault("log.format", "json")
	// 语句或表达式
	v.SetDefault("log.addSource", false)

	// 如果命令行指定了配置文件，则使用它
	// 条件判断 if
	if *configFile != "" {
		// 语句或表达式
		v.SetConfigFile(*configFile)
		// 语句或表达式
		log.Printf("Using config file from command line: %s", *configFile)
		// 语句或表达式
	} else {
		// 否则，按默认规则查找
		// 语句或表达式
		v.AddConfigPath("etc") // 在etc目录查找
		// 语句或表达式
		v.SetConfigName("config") // 默认配置文件名（不带后缀）
		// 语句或表达式
		v.SetConfigType("yaml") // 配置文件类型
		// 语句或表达式
		log.Println("Searching for 'config.yml' in the current directory...")
		// 语句或表达式
	}

	// 3. 读取配置文件
	// 条件判断 if
	if err := v.ReadInConfig(); err != nil {
		// 语句或表达式
		panic(fmt.Errorf("fatal error config file: %w", err))
		// 语句或表达式
	}

	// 4. 将配置反序列化到全局的 conf 变量中
	// 条件判断 if
	if err := v.Unmarshal(&conf); err != nil {
		// 语句或表达式
		panic(fmt.Errorf("unable to decode into struct: %w", err))
		// 语句或表达式
	}

	// 5. 开启配置热加载
	// 语句或表达式
	v.WatchConfig()
	// 语句或表达式
	v.OnConfigChange(func(e fsnotify.Event) {
		// 语句或表达式
		log.Printf("Config file changed: %s. Reloading...", e.Name)
		// 条件判断 if
		if err := v.Unmarshal(&conf); err != nil {
			// 语句或表达式
			log.Printf("Error reloading config: %v", err)
			// 语句或表达式
		} else {
			// 语句或表达式
			log.Println("Config reloaded successfully.")
			// 这里可以加入回调函数，通知其他模块配置已更新
			// 比如，重新设置日志级别
			// 语句或表达式
		}
		// 语句或表达式
	})
	// 返回语句
	return v
	// 语句或表达式
}

// GetConfig 返回已加载的配置单例
// 在调用此函数前，必须先调用 Init()
// 定义函数 GetConfig（签名开始）
func GetConfig() *Config {
	// 条件判断 if
	if conf == nil {
		// 确保即使有人忘记调用 Init，程序也会以明确的方式失败
		// 语句或表达式
		panic("config not initialized, please call config.Init() first")
		// 语句或表达式
	}
	// 返回语句
	return conf
	// 语句或表达式
}

// 定义函数 GetHost（签名开始）
func (m *Mysql) GetHost() string {
	// 条件判断 if
	if m == nil || m.Host == nil {
		// 返回语句
		return "127.0.0.1"
		// 语句或表达式
	}
	// 返回语句
	return *m.Host
	// 语句或表达式
}

// 定义函数 GetPort（签名开始）
func (m *Mysql) GetPort() int {
	// 条件判断 if
	if m == nil || m.Port == nil {
		// 返回语句
		return 3306
		// 语句或表达式
	}
	// 返回语句
	return *m.Port
	// 语句或表达式
}

// 定义函数 GetUser（签名开始）
func (m *Mysql) GetUser() string {
	// 条件判断 if
	if m == nil || m.User == nil {
		// 返回语句
		return ""
		// 语句或表达式
	}
	// 返回语句
	return *m.User
	// 语句或表达式
}

// 定义函数 GetPassword（签名开始）
func (m *Mysql) GetPassword() string {
	// 条件判断 if
	if m == nil || m.Password == nil {
		// 返回语句
		return ""
		// 语句或表达式
	}
	// 返回语句
	return *m.Password
	// 语句或表达式
}

// 定义函数 GetDatabase（签名开始）
func (m *Mysql) GetDatabase() string {
	// 条件判断 if
	if m == nil || m.Database == nil {
		// 返回语句
		return ""
		// 语句或表达式
	}
	// 返回语句
	return *m.Database
	// 语句或表达式
}

// 定义函数 GetMaxIdleConns（签名开始）
func (m *Mysql) GetMaxIdleConns() int {
	// 条件判断 if
	if m == nil || m.MaxIdleConns == nil {
		// 返回语句
		return 10
		// 语句或表达式
	}
	// 返回语句
	return *m.MaxIdleConns
	// 语句或表达式
}

// 定义函数 GetPingTimeout（签名开始）
func (m *Mysql) GetPingTimeout() time.Duration {
	// 条件判断 if
	if m == nil || m.PingTimeout == nil {
		// 返回语句
		return 5 * time.Second
		// 语句或表达式
	}
	// 返回语句
	return *m.PingTimeout
	// 语句或表达式
}

// 定义函数 GetMaxOpenConns（签名开始）
func (m *Mysql) GetMaxOpenConns() int {
	// 条件判断 if
	if m == nil || m.MaxOpenConns == nil {
		// 返回语句
		return 100
		// 语句或表达式
	}
	// 返回语句
	return *m.MaxOpenConns
	// 语句或表达式
}
