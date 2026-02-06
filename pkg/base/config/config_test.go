// config 包测试：验证配置加载与环境变量覆盖。
package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadWithEnvOverride 验证环境变量可覆盖配置文件值。
func TestLoadWithEnvOverride(t *testing.T) {
	t.Setenv("FLASHSALE_MYSQL_HOST", "127.0.0.2")
	t.Setenv("FLASHSALE_KAFKA_BROKERS", "k1:9092,k2:9092")

	cfg, err := Load(filepath.Join("..", "..", "..", "configs", "local", "dev.yaml"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.MySQL.Host != "127.0.0.2" {
		t.Fatalf("expected mysql.host override, got %s", cfg.MySQL.Host)
	}
	if len(cfg.Kafka.Brokers) != 2 {
		t.Fatalf("expected 2 kafka brokers, got %d", len(cfg.Kafka.Brokers))
	}
}

// TestLoadMissingFile 验证配置文件缺失时返回错误。
func TestLoadMissingFile(t *testing.T) {
	_, err := Load("missing.yaml")
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
	if _, statErr := os.Stat("missing.yaml"); !os.IsNotExist(statErr) {
		t.Fatalf("unexpected stat result: %v", statErr)
	}
}
