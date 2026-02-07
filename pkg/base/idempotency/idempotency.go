// Package idempotency 提供基于 Redis 的通用幂等执行保护。
package idempotency

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	// ErrInProgress 表示同一幂等键有请求正在执行。
	ErrInProgress = errors.New("idempotency in progress")
	// ErrDuplicateRequest 表示同一幂等键已执行完成，不允许重复执行。
	ErrDuplicateRequest = errors.New("idempotency duplicate request")
)

// unlockScript 通过 token 校验删除锁，防止误释放。
var unlockScript = redis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("del", KEYS[1])
end
return 0
`)

var (
	mu     sync.RWMutex
	client *redis.Client
	prefix = "idempotency"
)

// Init 初始化幂等组件。
func Init(c *redis.Client, keyPrefix string) error {
	if c == nil {
		return fmt.Errorf("redis client is nil")
	}
	mu.Lock()
	defer mu.Unlock()
	client = c
	if keyPrefix != "" {
		prefix = keyPrefix
	}
	return nil
}

// Guard 在幂等键维度保证业务回调最多成功执行一次。
//
// 关键流程说明：
// 1. 参数校验：run/key/ttl 必须可用，避免无效调用。
// 2. 读取组件状态：拿到 Redis 客户端与 key 前缀。
// 3. 检查 doneKey：若已存在，说明请求已成功执行过，直接返回重复请求错误。
// 4. 抢占 lockKey：用 SETNX+TTL 争抢执行权，失败则说明并发请求正在处理。
// 5. 注册延迟解锁：无论 run 成功或失败，都尝试释放 lockKey。
// 6. 执行业务回调：仅拿到执行权的请求会进入此阶段。
// 7. 写入 doneKey：业务成功后标记完成，阻止后续重复执行。
func Guard(ctx context.Context, key string, ttl time.Duration, run func() error) error {
	if run == nil {
		return fmt.Errorf("run function is nil")
	}
	if key == "" {
		return fmt.Errorf("idempotency key is empty")
	}
	if ttl <= 0 {
		ttl = time.Minute
	}

	c, keyPrefix, err := state()
	if err != nil {
		return err
	}

	doneKey := fmt.Sprintf("%s:done:%s", keyPrefix, key)
	lockKey := fmt.Sprintf("%s:lock:%s", keyPrefix, key)

	exists, err := c.Exists(ctx, doneKey).Result()
	if err != nil {
		return err
	}
	if exists > 0 {
		return ErrDuplicateRequest
	}

	token := uuid.NewString()
	ok, err := c.SetNX(ctx, lockKey, token, ttl).Result()
	if err != nil {
		return err
	}
	if !ok {
		return ErrInProgress
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, _ = unlockScript.Run(unlockCtx, c, []string{lockKey}, token).Result()
	}()

	if err := run(); err != nil {
		return err
	}
	if err := c.Set(ctx, doneKey, "1", ttl).Err(); err != nil {
		return err
	}
	return nil
}

// state 返回当前初始化状态中的 Redis 客户端和前缀。
func state() (*redis.Client, string, error) {
	mu.RLock()
	defer mu.RUnlock()
	if client == nil {
		return nil, "", fmt.Errorf("idempotency not initialized")
	}
	return client, prefix, nil
}
