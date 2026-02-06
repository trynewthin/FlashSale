// idempotency 包测试：验证并发场景下幂等保护生效。
package idempotency

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// TestGuardConcurrent 验证同一幂等键仅允许一次成功执行。
func TestGuardConcurrent(t *testing.T) {
	mini, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	defer mini.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	if err := Init(rdb, "test-idem"); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	var executed int32
	start := make(chan struct{})
	const workers = 12

	var wg sync.WaitGroup
	results := make(chan error, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			err := Guard(context.Background(), "same-key", time.Second, func() error {
				atomic.AddInt32(&executed, 1)
				time.Sleep(100 * time.Millisecond)
				return nil
			})
			results <- err
		}()
	}

	close(start)
	wg.Wait()
	close(results)

	if got := atomic.LoadInt32(&executed); got != 1 {
		t.Fatalf("expected run to execute once, got %d", got)
	}

	seenInProgress := false
	for err := range results {
		if err == nil {
			continue
		}
		if errors.Is(err, ErrInProgress) || errors.Is(err, ErrDuplicateRequest) {
			seenInProgress = true
			continue
		}
		t.Fatalf("unexpected error: %v", err)
	}
	if !seenInProgress {
		t.Fatal("expected in-progress or duplicate errors from concurrent calls")
	}

	err = Guard(context.Background(), "same-key", time.Second, func() error { return nil })
	if !errors.Is(err, ErrDuplicateRequest) {
		t.Fatalf("expected duplicate error on second call, got %v", err)
	}
}
