package redisprobe

import (
	"context"
	"fmt"
	"time"

	"flashsale/pkg/base/config"
	"flashsale/pkg/base/redisx"
)

func Run(configPath string, timeout time.Duration) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	client, err := redisx.New(cfg.Redis)
	if err != nil {
		return err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := redisx.Ping(ctx, client); err != nil {
		return err
	}
	if err := client.Set(ctx, "smoke:key", "ok", time.Minute).Err(); err != nil {
		return err
	}
	v, err := client.Get(ctx, "smoke:key").Result()
	if err != nil {
		return err
	}
	if v != "ok" {
		return fmt.Errorf("unexpected value: %s", v)
	}

	unlock, ok, err := redisx.TryLock(ctx, client, "smoke:lock", 5*time.Second)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("lock should be acquired")
	}
	if err := unlock(); err != nil {
		return err
	}

	fmt.Println("redis ok")
	return nil
}
