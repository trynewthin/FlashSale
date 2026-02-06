// mysqlcheck 用于批量校验 MySQL 分库连通性。
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"time"

	"flashsale/pkg/base/config"
	"flashsale/pkg/base/mysqlx"
)

// main 执行 MySQL smoke 检查：连接、Ping 和 SELECT 1。
func main() {
	configPath := flag.String("config", "configs/local/dev.yaml", "config file path")
	timeout := flag.Duration("timeout", 5*time.Second, "ping timeout")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		panic(err)
	}

	databases := cfg.MySQL.Databases
	if len(databases) == 0 {
		databases = []string{cfg.MySQL.Database}
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	for _, dbName := range databases {
		dbCfg := cfg.MySQL
		dbCfg.Database = dbName
		db, err := mysqlx.Open(dbCfg)
		if err != nil {
			panic(fmt.Errorf("open db[%s]: %w", dbName, err))
		}
		if err := mysqlx.Ping(ctx, db); err != nil {
			_ = db.Close()
			panic(fmt.Errorf("ping db[%s]: %w", dbName, err))
		}
		if err := selectOne(ctx, db); err != nil {
			_ = db.Close()
			panic(fmt.Errorf("query db[%s]: %w", dbName, err))
		}
		_ = db.Close()
		fmt.Printf("mysql ok: %s\n", dbName)
	}
}

// selectOne 执行最小 SQL 校验查询。
func selectOne(ctx context.Context, db *sql.DB) error {
	var one int
	return db.QueryRowContext(ctx, "SELECT 1").Scan(&one)
}
