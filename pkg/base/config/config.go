// config 包包含相关应用代码。
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

// AppConfig 是基础层总配置模型。
type AppConfig struct {
	Service    string           `mapstructure:"service"`
	Log        LogConfig        `mapstructure:"log"`
	MySQL      MySQLConfig      `mapstructure:"mysql"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Kafka      KafkaConfig      `mapstructure:"kafka"`
	JWT        JWTConfig        `mapstructure:"jwt"`
	OTEL       OTELConfig       `mapstructure:"otel"`
	Prometheus PrometheusConfig `mapstructure:"prometheus"`
}

// LogConfig 是日志配置。
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// MySQLConfig 是 MySQL 配置。
type MySQLConfig struct {
	DSN             string        `mapstructure:"dsn"`
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	Params          string        `mapstructure:"params"`
	MaxOpenConns    int           `mapstructure:"maxOpenConns"`
	MaxIdleConns    int           `mapstructure:"maxIdleConns"`
	ConnMaxLifetime time.Duration `mapstructure:"connMaxLifetime"`
	Databases       []string      `mapstructure:"databases"`
}

// RedisConfig 是 Redis 配置。
type RedisConfig struct {
	Addr         string        `mapstructure:"addr"`
	Password     string        `mapstructure:"password"`
	DB           int           `mapstructure:"db"`
	DialTimeout  time.Duration `mapstructure:"dialTimeout"`
	ReadTimeout  time.Duration `mapstructure:"readTimeout"`
	WriteTimeout time.Duration `mapstructure:"writeTimeout"`
}

// KafkaConfig 是 Kafka 配置。
type KafkaConfig struct {
	Brokers  []string         `mapstructure:"brokers"`
	ClientID string           `mapstructure:"clientID"`
	GroupID  string           `mapstructure:"groupID"`
	Topics   KafkaTopicConfig `mapstructure:"topics"`
}

// KafkaTopicConfig 是业务 Topic 配置。
type KafkaTopicConfig struct {
	OrderCreate       string `mapstructure:"orderCreate"`
	OrderCreateDLQ    string `mapstructure:"orderCreateDLQ"`
	StockCompensate   string `mapstructure:"stockCompensate"`
	SeckillTrafficRaw string `mapstructure:"seckillTrafficRaw"`
	SeckillTrafficDLQ string `mapstructure:"seckillTrafficDLQ"`
	SeckillOrderState string `mapstructure:"seckillOrderState"`
}

// JWTConfig 是双域 JWT 配置。
type JWTConfig struct {
	User  JWTDomainConfig `mapstructure:"user"`
	Admin JWTDomainConfig `mapstructure:"admin"`
}

// JWTDomainConfig 是单域 JWT 配置。
type JWTDomainConfig struct {
	Secret   string        `mapstructure:"secret"`
	Issuer   string        `mapstructure:"issuer"`
	Audience string        `mapstructure:"audience"`
	TTL      time.Duration `mapstructure:"ttl"`
}

// OTELConfig 是链路追踪配置。
type OTELConfig struct {
	Endpoint    string `mapstructure:"endpoint"`
	Insecure    bool   `mapstructure:"insecure"`
	ServiceName string `mapstructure:"serviceName"`
}

// PrometheusConfig 是指标暴露配置。
type PrometheusConfig struct {
	Addr      string `mapstructure:"addr"`
	Path      string `mapstructure:"path"`
	Namespace string `mapstructure:"namespace"`
}

// Load 从配置文件和环境变量加载 AppConfig。
//
// 关键流程说明：
// 1. 初始化 viper 并注入默认值，保证缺省配置可运行。
// 2. 绑定配置文件路径与环境变量前缀 FLASHSALE。
// 3. 读取配置文件，区分“文件不存在”和“读取失败”。
// 4. 使用 decode hook 反序列化 duration/slice 等字段。
// 5. 执行显式环境变量覆盖，处理切片与端口等细粒度场景。
func Load(path string) (*AppConfig, error) {
	v := viper.New()
	setDefaults(v)

	v.SetConfigFile(path)
	v.SetEnvPrefix("FLASHSALE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if errors.As(err, &notFound) {
			return nil, fmt.Errorf("config file not found: %s", path)
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := &AppConfig{}
	err := v.Unmarshal(cfg,
		viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	applyExplicitEnvOverrides(cfg)
	return cfg, nil
}

// setDefaults 设置基础默认值，避免配置缺失导致启动失败。
func setDefaults(v *viper.Viper) {
	v.SetDefault("service", "flashsale-base")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")

	v.SetDefault("mysql.host", "localhost")
	v.SetDefault("mysql.port", 3306)
	v.SetDefault("mysql.user", "flash")
	v.SetDefault("mysql.password", "flash123")
	v.SetDefault("mysql.database", "flash_user")
	v.SetDefault("mysql.params", "parseTime=true&loc=Local&charset=utf8mb4")
	v.SetDefault("mysql.maxOpenConns", 50)
	v.SetDefault("mysql.maxIdleConns", 10)
	v.SetDefault("mysql.connMaxLifetime", "30m")
	v.SetDefault("mysql.databases", []string{"flash_user", "flash_admin", "flash_product", "flash_order", "flash_seckill"})

	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.dialTimeout", "5s")
	v.SetDefault("redis.readTimeout", "3s")
	v.SetDefault("redis.writeTimeout", "3s")

	v.SetDefault("kafka.brokers", []string{"localhost:29092"})
	v.SetDefault("kafka.clientID", "flashsale-local")
	v.SetDefault("kafka.groupID", "flashsale-smoke")
	v.SetDefault("kafka.topics.orderCreate", "order.create")
	v.SetDefault("kafka.topics.orderCreateDLQ", "order.create.dlq")
	v.SetDefault("kafka.topics.stockCompensate", "stock.compensate")
	v.SetDefault("kafka.topics.seckillTrafficRaw", "seckill.traffic.raw")
	v.SetDefault("kafka.topics.seckillTrafficDLQ", "seckill.traffic.dlq")
	v.SetDefault("kafka.topics.seckillOrderState", "seckill.order.state")

	v.SetDefault("jwt.user.secret", "user-secret-change-me")
	v.SetDefault("jwt.user.issuer", "flashsale-user")
	v.SetDefault("jwt.user.audience", "flashsale-user-client")
	v.SetDefault("jwt.user.ttl", "24h")
	v.SetDefault("jwt.admin.secret", "admin-secret-change-me")
	v.SetDefault("jwt.admin.issuer", "flashsale-admin")
	v.SetDefault("jwt.admin.audience", "flashsale-admin-client")
	v.SetDefault("jwt.admin.ttl", "12h")

	v.SetDefault("otel.endpoint", "localhost:4317")
	v.SetDefault("otel.insecure", true)
	v.SetDefault("otel.serviceName", "flashsale-base")

	v.SetDefault("prometheus.addr", ":9100")
	v.SetDefault("prometheus.path", "/metrics")
	v.SetDefault("prometheus.namespace", "flashsale")
}

// applyExplicitEnvOverrides 对部分关键字段执行显式覆盖。
func applyExplicitEnvOverrides(cfg *AppConfig) {
	if cfg == nil {
		return
	}
	if v := strings.TrimSpace(os.Getenv("FLASHSALE_MYSQL_HOST")); v != "" {
		cfg.MySQL.Host = v
	}
	if v := strings.TrimSpace(os.Getenv("FLASHSALE_MYSQL_PORT")); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.MySQL.Port = p
		}
	}
	if v := strings.TrimSpace(os.Getenv("FLASHSALE_MYSQL_USER")); v != "" {
		cfg.MySQL.User = v
	}
	if v := os.Getenv("FLASHSALE_MYSQL_PASSWORD"); v != "" {
		cfg.MySQL.Password = v
	}
	if v := strings.TrimSpace(os.Getenv("FLASHSALE_REDIS_ADDR")); v != "" {
		cfg.Redis.Addr = v
	}
	if v := strings.TrimSpace(os.Getenv("FLASHSALE_KAFKA_BROKERS")); v != "" {
		parts := strings.Split(v, ",")
		cfg.Kafka.Brokers = make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				cfg.Kafka.Brokers = append(cfg.Kafka.Brokers, part)
			}
		}
	}
}
