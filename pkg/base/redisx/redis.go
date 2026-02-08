// redisx 包包含相关应用代码。
package redisx

import (
	"context"
	"fmt"
	"time"

	"flashsale/pkg/base/config"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/maintnotifications"
)

// unlockScript 通过 token 校验删除锁，避免误删他人锁。
var unlockScript = redis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("del", KEYS[1])
end
return 0
`)

// New 创建 Redis 客户端并设置默认超时参数。
func New(cfg config.RedisConfig) (*redis.Client, error) {
	addr := cfg.Addr
	if addr == "" {
		addr = "localhost:6379"
	}
	client := redis.NewClient(&redis.Options{
		Addr:            addr,
		Password:        cfg.Password,
		DB:              cfg.DB,
		DialTimeout:     defaultDuration(cfg.DialTimeout, 5*time.Second),
		ReadTimeout:     defaultDuration(cfg.ReadTimeout, 3*time.Second),
		WriteTimeout:    defaultDuration(cfg.WriteTimeout, 3*time.Second),
		DisableIdentity: true,
		MaintNotificationsConfig: &maintnotifications.Config{
			Mode: maintnotifications.ModeDisabled,
		},
	})
	return client, nil
}

// Ping 检查 Redis 连通性。
func Ping(ctx context.Context, c *redis.Client) error {
	if c == nil {
		return fmt.Errorf("redis client is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return c.Ping(ctx).Err()
}

// TryLock 尝试获取分布式锁，并返回释放函数。
//
// 关键流程说明：
// 1. 参数校验：校验 client/key/ttl，避免无意义锁操作。
// 2. 生成 token：每次加锁生成唯一 token，用于后续解锁所有权校验。
// 3. SETNX 加锁：在 ttl 内原子抢锁，失败直接返回 ok=false。
// 4. 生成 unlock 闭包：通过 Lua 对比 token 后删除，确保“仅持有者可解锁”。
// 5. 返回 unlock：调用方在业务结束后执行，完成安全释放。
func TryLock(ctx context.Context, c *redis.Client, key string, ttl time.Duration) (func() error, bool, error) {
	if c == nil {
		return nil, false, fmt.Errorf("redis client is nil")
	}
	if key == "" {
		return nil, false, fmt.Errorf("lock key is empty")
	}
	if ttl <= 0 {
		ttl = 5 * time.Second
	}
	token := uuid.NewString()

	ok, err := c.SetNX(ctx, key, token, ttl).Result()
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}

	unlock := func() error {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, err := unlockScript.Run(unlockCtx, c, []string{key}, token).Result()
		return err
	}
	return unlock, true, nil
}

// defaultDuration 在配置未设置时回退到默认值。
func defaultDuration(v time.Duration, fallback time.Duration) time.Duration {
	if v <= 0 {
		return fallback
	}
	return v
}
