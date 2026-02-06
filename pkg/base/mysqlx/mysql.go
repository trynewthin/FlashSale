// Package mysqlx 提供 MySQL 连接初始化、DSN 构造与连接池默认策略。
package mysqlx

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"flashsale/pkg/base/config"
	_ "github.com/go-sql-driver/mysql"
)

const (
	defaultMaxOpenConns = 50
	defaultMaxIdleConns = 10
	defaultMaxLifetime  = 30 * time.Minute
)

// Open 根据配置创建 MySQL 连接并应用连接池默认值。
func Open(cfg config.MySQLConfig) (*sql.DB, error) {
	dsn, err := BuildDSN(cfg)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql open: %w", err)
	}
	applyPoolDefaults(db, cfg)
	return db, nil
}

// Ping 检查数据库连通性。
func Ping(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return db.PingContext(ctx)
}

// BuildDSN 根据结构化配置组装 DSN。
func BuildDSN(cfg config.MySQLConfig) (string, error) {
	if cfg.DSN != "" {
		return cfg.DSN, nil
	}
	host := cfg.Host
	if host == "" {
		host = "localhost"
	}
	port := cfg.Port
	if port == 0 {
		port = 3306
	}
	user := cfg.User
	if user == "" {
		return "", fmt.Errorf("mysql user is required")
	}
	database := cfg.Database
	if database == "" {
		return "", fmt.Errorf("mysql database is required")
	}
	params := cfg.Params
	if params == "" {
		params = "parseTime=true&loc=Local&charset=utf8mb4"
	}
	if !strings.Contains(params, "parseTime=") {
		params += "&parseTime=true"
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s", user, cfg.Password, host, port, database, params), nil
}

// applyPoolDefaults 为连接池补齐默认参数，避免零值导致资源策略失控。
func applyPoolDefaults(db *sql.DB, cfg config.MySQLConfig) {
	maxOpen := cfg.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = defaultMaxOpenConns
	}
	maxIdle := cfg.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = defaultMaxIdleConns
	}
	maxLifetime := cfg.ConnMaxLifetime
	if maxLifetime <= 0 {
		maxLifetime = defaultMaxLifetime
	}
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(maxLifetime)
}
