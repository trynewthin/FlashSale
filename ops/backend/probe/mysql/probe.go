package mysqlprobe

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"flashsale/pkg/base/config"
	"flashsale/pkg/base/mysqlx"
)

func Run(configPath string, timeout time.Duration) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	databases := cfg.MySQL.Databases
	if len(databases) == 0 {
		databases = []string{cfg.MySQL.Database}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for _, dbName := range databases {
		dbCfg := cfg.MySQL
		dbCfg.Database = dbName
		db, err := mysqlx.Open(dbCfg)
		if err != nil {
			return fmt.Errorf("open db[%s]: %w", dbName, err)
		}
		if err := mysqlx.Ping(ctx, db); err != nil {
			_ = db.Close()
			return fmt.Errorf("ping db[%s]: %w", dbName, err)
		}
		if err := selectOne(ctx, db); err != nil {
			_ = db.Close()
			return fmt.Errorf("query db[%s]: %w", dbName, err)
		}
		_ = db.Close()
		fmt.Printf("mysql ok: %s\n", dbName)
	}
	return nil
}

func selectOne(ctx context.Context, db *sql.DB) error {
	var one int
	return db.QueryRowContext(ctx, "SELECT 1").Scan(&one)
}
