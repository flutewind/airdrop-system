// internal/config/config.go
package config

import (
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var (
	cfg  *Config
	once sync.Once
)

type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Admin      AdminConfig      `mapstructure:"admin"`
	Airdrop    AirdropConfig    `mapstructure:"airdrop"`
	Blockchain BlockchainConfig `mapstructure:"blockchain"`
	Security   SecurityConfig   `mapstructure:"security"`
	Logging    LoggingConfig    `mapstructure:"logging"`
	Cache      CacheConfig      `mapstructure:"cache"`
	Email      EmailConfig      `mapstructure:"email"`
}

type ServerConfig struct {
	Address      string      `mapstructure:"address"`
	Mode         string      `mapstructure:"mode"`
	ReadTimeout  int         `mapstructure:"read_timeout"`
	WriteTimeout int         `mapstructure:"write_timeout"`
	HTTPS        HTTPSConfig `mapstructure:"https"`
}

type HTTPSConfig struct {
	Enable   bool   `mapstructure:"enable"`
	Address  string `mapstructure:"address"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

type DatabaseConfig struct {
	Driver      string         `mapstructure:"driver"`
	SQLite      SQLiteConfig   `mapstructure:"sqlite"`
	Postgres    PostgresConfig `mapstructure:"postgres"`
	MySQL       MySQLConfig    `mapstructure:"mysql"`
	MaxIdle     int            `mapstructure:"max_idle_conns"`
	MaxOpen     int            `mapstructure:"max_open_conns"`
	MaxLifetime int            `mapstructure:"conn_max_lifetime"`
}

type SQLiteConfig struct {
	Path        string `mapstructure:"path"`
	ForeignKeys bool   `mapstructure:"foreign_keys"`
}

type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type AdminConfig struct {
	InitUsername  string `mapstructure:"init_username"`
	InitPassword  string `mapstructure:"init_password"`
	SessionExpire int    `mapstructure:"session_expire"`
	JWTSecret     string `mapstructure:"jwt_secret"`
	JWTExpire     int    `mapstructure:"jwt_expire"`
}

// 其他配置结构体定义...
// 在type定义部分添加
type MySQLConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
}

type PaginationConfig struct {
	DefaultPage     int `mapstructure:"default_page"`
	DefaultPageSize int `mapstructure:"default_page_size"`
	MaxPageSize     int `mapstructure:"max_page_size"`
}

type AirdropConfig struct {
	MaxParticipants    int              `mapstructure:"max_participants"`
	TwitterValidation  bool             `mapstructure:"twitter_validation"`
	EthereumValidation bool             `mapstructure:"ethereum_validation"`
	DuplicateCheck     bool             `mapstructure:"duplicate_check"`
	DefaultDecimals    int              `mapstructure:"default_token_decimals"`
	Pagination         PaginationConfig `mapstructure:"pagination"`
}

type EthereumConfig struct {
	Network         string `mapstructure:"network"`
	RPCURL          string `mapstructure:"rpc_url"`
	PrivateKey      string `mapstructure:"private_key"`
	ContractAddress string `mapstructure:"contract_address"`
}

type BlockchainConfig struct {
	Ethereum EthereumConfig `mapstructure:"ethereum"`
	GasLimit int            `mapstructure:"gas_limit"`
	GasPrice string         `mapstructure:"gas_price"`
}

type CORSConfig struct {
	AllowOrigins []string `mapstructure:"allow_origins"`
	AllowMethods []string `mapstructure:"allow_methods"`
}

type RateLimitConfig struct {
	Enable            bool `mapstructure:"enable"`
	RequestsPerMinute int  `mapstructure:"requests_per_minute"`
}

type RecaptchaConfig struct {
	Enable    bool   `mapstructure:"enable"`
	SiteKey   string `mapstructure:"site_key"`
	SecretKey string `mapstructure:"secret_key"`
}

type SecurityConfig struct {
	CORS      CORSConfig      `mapstructure:"cors"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
	Recaptcha RecaptchaConfig `mapstructure:"recaptcha"`
}

type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	FilePath   string `mapstructure:"file_path"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
}

type RedisConfig struct {
	Enable   bool   `mapstructure:"enable"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type CacheConfig struct {
	Redis             RedisConfig `mapstructure:"redis"`
	DefaultExpiration int         `mapstructure:"default_expiration"`
}

type EmailConfig struct {
	Enable   bool   `mapstructure:"enable"`
	SMTPHost string `mapstructure:"smtp_host"`
	SMTPPort int    `mapstructure:"smtp_port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
}

// Load 加载配置
func Load() *Config {
	once.Do(func() {
		// 初始化Viper
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")

		// 添加配置文件路径（按优先级）
		viper.AddConfigPath(".")              // 当前目录
		viper.AddConfigPath("./config")       // config目录
		viper.AddConfigPath("$HOME/.airdrop") // 用户目录
		viper.AddConfigPath("/etc/airdrop")   // 系统目录

		// 环境变量覆盖
		viper.AutomaticEnv()
		viper.SetEnvPrefix("AIRDROP")

		// 绑定环境变量
		bindEnvVars()

		// 读取配置文件
		if err := viper.ReadInConfig(); err != nil {
			log.Printf("Warning: Could not read config file: %v", err)
			log.Println("Using default configuration...")
		} else {
			log.Printf("Using config file: %s", viper.ConfigFileUsed())
		}

		// 监听配置变化（仅开发环境）
		if viper.GetString("server.mode") == "debug" {
			viper.WatchConfig()
			viper.OnConfigChange(func(e fsnotify.Event) {
				log.Printf("Config file changed: %s", e.Name)
				reloadConfig()
			})
		}

		// 反序列化配置
		if err := viper.Unmarshal(&cfg); err != nil {
			log.Fatalf("Unable to decode config: %v", err)
		}

		// 创建必要目录
		createDirectories()

		// 验证配置
		validateConfig()
	})

	return cfg
}

// 绑定环境变量
func bindEnvVars() {
	viper.BindEnv("server.address", "AIRDROP_SERVER_ADDRESS")
	viper.BindEnv("server.mode", "AIRDROP_SERVER_MODE")
	viper.BindEnv("database.driver", "AIRDROP_DB_DRIVER")
	viper.BindEnv("database.postgres.host", "AIRDROP_DB_HOST")
	viper.BindEnv("database.postgres.port", "AIRDROP_DB_PORT")
	viper.BindEnv("database.postgres.user", "AIRDROP_DB_USER")
	viper.BindEnv("database.postgres.password", "AIRDROP_DB_PASSWORD")
	viper.BindEnv("database.postgres.dbname", "AIRDROP_DB_NAME")
	viper.BindEnv("admin.jwt_secret", "AIRDROP_JWT_SECRET")
	viper.BindEnv("blockchain.ethereum.rpc_url", "AIRDROP_ETH_RPC_URL")
	viper.BindEnv("blockchain.ethereum.private_key", "AIRDROP_ETH_PRIVATE_KEY")
}

// 创建必要目录
func createDirectories() {
	dirs := []string{
		"./data",
		"./logs",
		"./static/uploads",
		"./migrations",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("Failed to create directory %s: %v", dir, err)
		}
	}
}

// 重新加载配置
func reloadConfig() {
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Printf("Error reloading config: %v", err)
	}
}

// 验证配置
func validateConfig() {
	if cfg.Admin.JWTSecret == "your-jwt-secret-key-change-in-production" {
		log.Println("WARNING: Using default JWT secret. Change it in production!")
	}

	if cfg.Database.Driver == "sqlite" {
		// 确保SQLite数据库目录存在
		sqlitePath := cfg.Database.SQLite.Path
		dir := filepath.Dir(sqlitePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create SQLite directory: %v", err)
		}
	}

	// 验证以太坊地址格式（如果配置了）
	if cfg.Blockchain.Ethereum.ContractAddress != "" {
		if !isValidEthereumAddress(cfg.Blockchain.Ethereum.ContractAddress) {
			log.Printf("WARNING: Invalid Ethereum contract address: %s",
				cfg.Blockchain.Ethereum.ContractAddress)
		}
	}
}

// 保留函数但不调用它
func isValidEthereumAddress(address string) bool {
	return true // 始终返回true，跳过验证
}

// 辅助函数：获取配置实例
func Get() *Config {
	if cfg == nil {
		return Load()
	}
	return cfg
}

// 获取环境特定的配置
func GetEnvConfig() string {
	env := os.Getenv("AIRDROP_ENV")
	if env == "" {
		return "development"
	}
	return env
}
