// Package config 提供应用程序配置加载和管理功能。
//
// 本文件定义了应用程序的全部配置结构，包括服务器、数据库、缓存、
// 安全、AI 模型等模块的配置。使用 viper 库支持 YAML 文件和环境变量。
package config

import (
	"log"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 是应用程序的顶层配置结构。
//
// 包含所有模块的配置，通过 mapstructure 标签映射配置文件。
type Config struct {
	App          App          `mapstructure:"app"`           // 应用基本配置
	Server       Server       `mapstructure:"server"`        // HTTP 服务器配置
	Log          Log          `mapstructure:"log"`           // 日志配置
	MySQL        MySQL        `mapstructure:"mysql"`         // MySQL 数据库配置
	Postgres     Postgres     `mapstructure:"postgres"`      // PostgreSQL 数据库配置
	SQLite       SQLite       `mapstructure:"sqlite"`        // SQLite 数据库配置
	Redis        Redis        `mapstructure:"redis"`         // Redis 缓存配置
	JWT          JWT          `mapstructure:"jwt"`           // JWT 认证配置
	Cors         Cors         `mapstructure:"cors"`          // CORS 跨域配置
	RateLimit    RateLimit    `mapstructure:"rate_limit"`    // 限流配置
	Pprof        Pprof        `mapstructure:"pprof"`         // pprof 性能分析配置
	Metrics      Metrics      `mapstructure:"metrics"`       // 指标暴露配置
	Security     Security     `mapstructure:"security"`      // 安全配置
	LLM          LLM          `mapstructure:"llm"`           // 大语言模型配置
	AI           AI           `mapstructure:"ai"`            // AI 功能配置
	Embedder     Embedder     `mapstructure:"embedder"`      // 向量嵌入模型配置
	Milvus       Milvus       `mapstructure:"milvus"`        // Milvus 向量数据库配置
	Prometheus   Prometheus   `mapstructure:"prometheus"`    // Prometheus 监控配置
	Swagger      Swagger      `mapstructure:"swagger"`       // Swagger API 文档配置
}

// App 包含应用程序基本配置。
type App struct {
	Name        string `mapstructure:"name"`         // 应用名称
	Env         string `mapstructure:"env"`          // 运行环境 (development/staging/production)
	Debug       bool   `mapstructure:"debug"`        // 调试模式开关
	Timezone    string `mapstructure:"timezone"`     // 时区设置
	AutoMigrate bool   `mapstructure:"auto_migrate"` // 自动迁移数据库表结构
}

// Server 包含 HTTP 服务器配置。
type Server struct {
	Host           string        `mapstructure:"host"`             // 监听主机
	Port           int           `mapstructure:"port"`             // 监听端口
	ReadTimeout    time.Duration `mapstructure:"read_timeout"`     // 读取超时
	WriteTimeout   time.Duration `mapstructure:"write_timeout"`    // 写入超时
	IdleTimeout    time.Duration `mapstructure:"idle_timeout"`     // 空闲超时
	MaxHeaderBytes string        `mapstructure:"max_header_bytes"` // 最大请求头字节数
	Salt           string        `mapstructure:"salt"`             // 密码加密盐值
}

// Log 包含日志配置。
type Log struct {
	Level  string  `mapstructure:"level"`  // 日志级别 (debug/info/warn/error)
	Format string  `mapstructure:"format"` // 输出格式 (json/text)
	Output string  `mapstructure:"output"` // 输出目标 (stdout/stderr/file)
	File   LogFile `mapstructure:"file"`   // 文件输出配置
}

// LogFile 包含日志文件配置。
type LogFile struct {
	Path       string `mapstructure:"path"`        // 文件路径
	MaxSize    int    `mapstructure:"max_size"`    // 单文件最大尺寸 (MB)
	MaxBackups int    `mapstructure:"max_backups"` // 最大备份数量
	MaxAge     int    `mapstructure:"max_age"`     // 最大保留天数
	Compress   bool   `mapstructure:"compress"`    // 是否压缩备份
}

// MySQL 包含 MySQL 数据库连接配置。
type MySQL struct {
	Enable          bool          `mapstructure:"enable"`            // 是否启用
	Host            string        `mapstructure:"host"`              // 数据库主机
	Port            string        `mapstructure:"port"`              // 数据库端口
	User            string        `mapstructure:"user"`              // 用户名
	Password        string        `mapstructure:"password"`          // 密码
	Database        string        `mapstructure:"database"`          // 数据库名
	Charset         string        `mapstructure:"charset"`           // 字符集
	ParseTime       bool          `mapstructure:"parse_time"`        // 是否解析时间
	Loc             string        `mapstructure:"loc"`               // 时区
	MaxOpenConns    int           `mapstructure:"max_open_conns"`    // 最大打开连接数
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`    // 最大空闲连接数
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"` // 连接最大生命周期
}

// Redis 包含 Redis 缓存连接配置。
type Redis struct {
	Enable       bool          `mapstructure:"enable"`         // 是否启用
	Addr         string        `mapstructure:"addr"`           // Redis 地址 (host:port)
	Password     string        `mapstructure:"password"`       // 密码
	DB           int           `mapstructure:"db"`             // 数据库索引
	PoolSize     int           `mapstructure:"pool_size"`      // 连接池大小
	MinIdleConns int           `mapstructure:"min_idle_conns"` // 最小空闲连接数
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`   // 拨号超时
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`   // 读取超时
	WriteTimeout time.Duration `mapstructure:"write_timeout"`  // 写入超时
}

// JWT 包含 JWT 认证配置。
type JWT struct {
	Secret        string        `mapstructure:"secret"`         // 签名密钥
	Issuer        string        `mapstructure:"issuer"`         // 签发者
	Expire        time.Duration `mapstructure:"expire"`         // 访问令牌过期时间
	RefreshExpire time.Duration `mapstructure:"refresh_expire"` // 刷新令牌过期时间
}

// Cors 包含 CORS 跨域配置。
type Cors struct {
	Enable           bool          `mapstructure:"enable"`            // 是否启用
	AllowOrigins     []string      `mapstructure:"allow_origins"`     // 允许的源
	AllowMethods     []string      `mapstructure:"allow_methods"`     // 允许的方法
	AllowHeaders     []string      `mapstructure:"allow_headers"`     // 允许的请求头
	ExposeHeaders    []string      `mapstructure:"expose_headers"`    // 暴露的响应头
	AllowCredentials bool          `mapstructure:"allow_credentials"` // 是否允许凭证
	MaxAge           time.Duration `mapstructure:"max_age"`           // 预检请求缓存时间
}

// RateLimit 包含限流配置。
type RateLimit struct {
	Enable bool `mapstructure:"enable"` // 是否启用
	Rps    int  `mapstructure:"rps"`    // 每秒请求数
	Burst  int  `mapstructure:"burst"`  // 突发容量
}

// Pprof 包含 pprof 性能分析配置。
type Pprof struct {
	Enable bool   `mapstructure:"enable"` // 是否启用
	Addr   string `mapstructure:"addr"`   // 监听地址
}

// Metrics 包含指标暴露配置。
type Metrics struct {
	Enable bool   `mapstructure:"enable"` // 是否启用
	Path   string `mapstructure:"path"`   // 指标路径
}

// Security 包含安全相关配置。
type Security struct {
	TrustedProxies []string `mapstructure:"trusted_proxies"` // 信任的代理 IP
	HideBanner     bool     `mapstructure:"hide_banner"`     // 是否隐藏启动横幅
	EncryptionKey  string   `mapstructure:"encryption_key"`  // 数据加密密钥
}

// LLM 包含大语言模型配置。
type LLM struct {
	Enable      bool    `mapstructure:"enable"`      // 是否启用
	Provider    string  `mapstructure:"provider"`    // 提供商 (openai/anthropic等)
	BaseURL     string  `mapstructure:"base_url"`    // API 基础 URL
	APIKey      string  `mapstructure:"api_key"`     // API 密钥
	Model       string  `mapstructure:"model"`       // 模型名称
	Temperature float64 `mapstructure:"temperature"` // 生成温度
}

// AI 包含 AI 功能开关配置。
type AI struct {
	UseMultiDomainArch    bool `mapstructure:"use_multi_domain_arch"`    // 是否使用多领域架构
	UseTurnBlockStreaming bool `mapstructure:"use_turn_block_streaming"` // 是否启用 turn/block 流式体验
}

// Postgres 包含 PostgreSQL 数据库连接配置。
type Postgres struct {
	Enable          bool          `mapstructure:"enable"`            // 是否启用
	Host            string        `mapstructure:"host"`              // 数据库主机
	Port            string        `mapstructure:"port"`              // 数据库端口
	User            string        `mapstructure:"user"`              // 用户名
	Password        string        `mapstructure:"password"`          // 密码
	Database        string        `mapstructure:"database"`          // 数据库名
	SSLMode         string        `mapstructure:"ssl_mode"`          // SSL 模式
	MaxOpenConns    int           `mapstructure:"max_open_conns"`    // 最大打开连接数
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`    // 最大空闲连接数
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"` // 连接最大生命周期
}

// SQLite 包含 SQLite 数据库配置。
type SQLite struct {
	Enable          bool          `mapstructure:"enable"`            // 是否启用
	File            string        `mapstructure:"file"`              // 数据库文件路径
	MaxOpenConns    int           `mapstructure:"max_open_conns"`    // 最大打开连接数
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`    // 最大空闲连接数
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"` // 连接最大生命周期
}

// Milvus 包含 Milvus 向量数据库配置。
type Milvus struct {
	Enable              bool          `mapstructure:"enable"`                // 是否启用
	Host                string        `mapstructure:"host"`                  // Milvus 主机
	Port                string        `mapstructure:"port"`                  // Milvus 端口
	Username            string        `mapstructure:"username"`              // 用户名
	Password            string        `mapstructure:"password"`              // 密码
	ApiKey              string        `mapstructure:"api_key"`               // API 密钥
	Database            string        `mapstructure:"database"`              // 数据库名
	Collection          string        `mapstructure:"collection"`            // 集合名
	UseTLS              bool          `mapstructure:"use_tls"`               // 是否使用 TLS
	Timeout             time.Duration `mapstructure:"timeout"`               // 超时时间
	HealthCheckInterval time.Duration `mapstructure:"health_check_interval"` // 健康检查间隔
	Dimension           int           `mapstructure:"dimension"`             // 向量维度
	IndexType           string        `mapstructure:"index_type"`            // 索引类型
}

// Embedder 包含向量嵌入模型配置。
type Embedder struct {
	Enable     bool          `mapstructure:"enable"`      // 是否启用
	Provider   string        `mapstructure:"provider"`    // 提供商
	Model      string        `mapstructure:"model"`       // 模型名称
	BaseURL    string        `mapstructure:"base_url"`    // API 基础 URL
	ApiKey     string        `mapstructure:"api_key"`     // API 密钥
	Timeout    time.Duration `mapstructure:"timeout"`     // 超时时间
	MaxRetries int           `mapstructure:"max_retries"` // 最大重试次数
}

// Prometheus 包含 Prometheus 监控配置。
type Prometheus struct {
	Enable         bool          `mapstructure:"enable"`          // 是否启用
	Address        string        `mapstructure:"address"`         // Prometheus 地址
	Host           string        `mapstructure:"host"`            // 主机
	Port           string        `mapstructure:"port"`            // 端口
	PushgatewayURL string        `mapstructure:"pushgateway_url"` // Pushgateway 地址
	Timeout        time.Duration `mapstructure:"timeout"`         // 超时时间
	MaxConcurrent  int           `mapstructure:"max_concurrent"`  // 最大并发数
	RetryCount     int           `mapstructure:"retry_count"`     // 重试次数
}

// Swagger 包含 Swagger API 文档配置。
type Swagger struct {
	Enable bool   `mapstructure:"enable"` // 是否启用 Swagger UI
	Path   string `mapstructure:"path"`   // Swagger UI 访问路径 (默认 /swagger/*)
}

// cfgFile 是配置文件路径，由命令行参数设置。
var cfgFile string

// CFG 是全局配置实例，在应用启动时初始化。
var CFG Config

// Debug 是全局调试模式标志。
var Debug bool

// SetConfigFile 设置配置文件路径。
func SetConfigFile(path string) {
	cfgFile = path
}

// MustNewConfig 加载并解析配置文件。
//
// 使用 viper 读取配置文件，支持环境变量覆盖。
// 如果加载失败则 panic。
func MustNewConfig() {
	// load config file
	viper.SetConfigFile(cfgFile)
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	replace := strings.NewReplacer(".", "_") // 替换点为下划线
	viper.SetEnvKeyReplacer(replace)         // 设置环境变量的替换器
	viper.AutomaticEnv()

	// unmarshal config
	if err := viper.Unmarshal(&CFG); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}
}

// AppEnv 返回应用环境名称（小写）。
func AppEnv() string {
	return strings.TrimSpace(strings.ToLower(CFG.App.Env))
}

// IsDevelopment 判断是否为开发环境。
func IsDevelopment() bool {
	switch AppEnv() {
	case "dev", "development", "local":
		return true
	default:
		return false
	}
}

// SwaggerEnabled 返回 Swagger UI 是否启用。
//
// 默认在开发环境启用，生产环境禁用。
// 可通过配置文件 swagger.enable 显式控制。
func SwaggerEnabled() bool {
	// 显式配置优先
	if CFG.Swagger.Enable {
		return true
	}
	// 开发环境默认启用
	return IsDevelopment()
}

// SwaggerPath 返回 Swagger UI 访问路径。
//
// 默认为 /swagger/*any (gin-swagger 标准格式)。
func SwaggerPath() string {
	if CFG.Swagger.Path != "" {
		return CFG.Swagger.Path
	}
	return "/swagger/*any"
}
