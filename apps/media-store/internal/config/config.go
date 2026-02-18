// config 包定义 media-store 的全部配置项。
// 所有配置通过环境变量注入，保持零文件依赖。
package config

import (
	"os"
	"strconv"
	"strings"
)

// Config media-store 服务配置。
type Config struct {
	Addr       string // 监听地址
	StorageDir string // 文件存储根目录（即 CDN 共享 volume）
	Secret     string // 静态鉴权 secret
	CDNOrigin  string // CDN 外部访问地址前缀（用于拼接返回的 URL）
	MaxSize    int64  // 单文件最大字节数
}

// Load 从环境变量加载配置。
func Load() Config {
	return Config{
		Addr:       envOr("MEDIA_STORE_ADDR", "0.0.0.0:9200"),
		StorageDir: envOr("MEDIA_STORE_DIR", "/srv/cdn/assets"),
		Secret:     envOr("MEDIA_STORE_SECRET", "flashsale-media-dev"),
		CDNOrigin:  envOr("MEDIA_STORE_CDN_ORIGIN", "http://localhost:19000"),
		MaxSize:    envOrInt64("MEDIA_STORE_MAX_SIZE", 10<<20), // 默认 10 MB
	}
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envOrInt64(key string, def int64) int64 {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return def
}
