// redischeck 用于校验 Redis 读写与分布式锁行为。
// redischeck 命令提供可执行入口。
package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"flashsale/pkg/base/config"
	"flashsale/pkg/base/redisx"
)

// main 执行 Redis smoke 检查：Ping、Set/Get、TryLock。
func main() {
	configPath := flag.String("config", "configs/dev.yaml", "config file path")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		panic(err)
	}

	client, err := redisx.New(cfg.Redis)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisx.Ping(ctx, client); err != nil {
		panic(err)
	}
	if err := client.Set(ctx, "smoke:key", "ok", time.Minute).Err(); err != nil {
		panic(err)
	}
	v, err := client.Get(ctx, "smoke:key").Result()
	if err != nil {
		panic(err)
	}
	if v != "ok" {
		panic(fmt.Errorf("unexpected value: %s", v))
	}

	unlock, ok, err := redisx.TryLock(ctx, client, "smoke:lock", 5*time.Second)
	if err != nil {
		panic(err)
	}
	if !ok {
		panic("lock should be acquired")
	}
	if err := unlock(); err != nil {
		panic(err)
	}

	fmt.Println("redis ok")
}
