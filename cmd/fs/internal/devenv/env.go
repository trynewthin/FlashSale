// devenv 负责加载本地环境变量文件并提供连接参数解析。
package devenv

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Load 读取 .env 格式文件并注入环境变量。
func Load(path string) error {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		example := path + ".example"
		if _, statErr := os.Stat(example); statErr == nil {
			return fmt.Errorf("dev env file not found: %s, copy from %s", path, example)
		}
		return fmt.Errorf("dev env file not found: %s", path)
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		if len(value) >= 2 {
			if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
				(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
				value = value[1 : len(value)-1]
			}
		}
		_ = os.Setenv(key, value)
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	applyDefaults()
	return nil
}

func applyDefaults() {
	setIfEmpty("FLASHSALE_MYSQL_HOST", "127.0.0.1")
	normalizeLocalhostHost("FLASHSALE_MYSQL_HOST")
	if os.Getenv("FLASHSALE_MYSQL_PORT") == "" && os.Getenv("FLASH_MYSQL_PORT") != "" {
		_ = os.Setenv("FLASHSALE_MYSQL_PORT", os.Getenv("FLASH_MYSQL_PORT"))
	}
	if os.Getenv("FLASHSALE_MYSQL_USER") == "" && os.Getenv("FLASH_MYSQL_APP_USER") != "" {
		_ = os.Setenv("FLASHSALE_MYSQL_USER", os.Getenv("FLASH_MYSQL_APP_USER"))
	}
	if os.Getenv("FLASHSALE_MYSQL_PASSWORD") == "" && os.Getenv("FLASH_MYSQL_APP_PASSWORD") != "" {
		_ = os.Setenv("FLASHSALE_MYSQL_PASSWORD", os.Getenv("FLASH_MYSQL_APP_PASSWORD"))
	}
	if os.Getenv("FLASHSALE_REDIS_ADDR") == "" && os.Getenv("FLASH_REDIS_PORT") != "" {
		_ = os.Setenv("FLASHSALE_REDIS_ADDR", "127.0.0.1:"+os.Getenv("FLASH_REDIS_PORT"))
	}
	normalizeLocalhostAddr("FLASHSALE_REDIS_ADDR")
	if os.Getenv("FLASHSALE_KAFKA_BROKERS") == "" && os.Getenv("FLASH_KAFKA_PORT") != "" {
		_ = os.Setenv("FLASHSALE_KAFKA_BROKERS", "127.0.0.1:"+os.Getenv("FLASH_KAFKA_PORT"))
	}
	normalizeLocalhostBrokers("FLASHSALE_KAFKA_BROKERS")
}

func setIfEmpty(key, value string) {
	if strings.TrimSpace(os.Getenv(key)) == "" {
		_ = os.Setenv(key, value)
	}
}

// normalizeLocalhostHost 将 localhost 规范为 127.0.0.1，避免 IPv6 回环差异。
func normalizeLocalhostHost(key string) {
	v := strings.TrimSpace(os.Getenv(key))
	if strings.EqualFold(v, "localhost") {
		_ = os.Setenv(key, "127.0.0.1")
	}
}

// normalizeLocalhostAddr 规范 host:port 形式的 localhost 地址。
func normalizeLocalhostAddr(key string) {
	v := strings.TrimSpace(os.Getenv(key))
	if strings.EqualFold(v, "localhost") {
		_ = os.Setenv(key, "127.0.0.1")
		return
	}
	if strings.HasPrefix(strings.ToLower(v), "localhost:") {
		_ = os.Setenv(key, "127.0.0.1:"+strings.TrimPrefix(v, "localhost:"))
	}
}

// normalizeLocalhostBrokers 规范逗号分隔的 broker 列表。
func normalizeLocalhostBrokers(key string) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return
	}
	parts := strings.Split(raw, ",")
	for i := range parts {
		part := strings.TrimSpace(parts[i])
		if strings.EqualFold(part, "localhost") {
			parts[i] = "127.0.0.1"
			continue
		}
		lower := strings.ToLower(part)
		if strings.HasPrefix(lower, "localhost:") {
			parts[i] = "127.0.0.1:" + strings.TrimPrefix(part, "localhost:")
		}
	}
	_ = os.Setenv(key, strings.Join(parts, ","))
}

// ResolvePath 计算相对仓库根目录的路径。
func ResolvePath(repoRoot, rel string) string {
	if filepath.IsAbs(rel) {
		return rel
	}
	return filepath.Join(repoRoot, rel)
}

// MySQLConfig 返回 mysql 连接配置。
func MySQLConfig() (host string, port int, user string, pass string, err error) {
	host = strings.TrimSpace(os.Getenv("FLASHSALE_MYSQL_HOST"))
	if host == "" {
		host = "127.0.0.1"
	}
	port = 3306
	if raw := strings.TrimSpace(os.Getenv("FLASHSALE_MYSQL_PORT")); raw != "" {
		v, convErr := strconv.Atoi(raw)
		if convErr != nil || v <= 0 {
			return "", 0, "", "", fmt.Errorf("invalid FLASHSALE_MYSQL_PORT: %s", raw)
		}
		port = v
	}

	user = strings.TrimSpace(os.Getenv("FLASHSALE_MYSQL_USER"))
	if user == "" {
		user = strings.TrimSpace(os.Getenv("FLASH_MYSQL_APP_USER"))
	}
	pass = os.Getenv("FLASHSALE_MYSQL_PASSWORD")
	if pass == "" {
		pass = os.Getenv("FLASH_MYSQL_APP_PASSWORD")
	}
	if user == "" || pass == "" {
		return "", 0, "", "", fmt.Errorf("mysql user/password not found from environment")
	}
	return host, port, user, pass, nil
}
