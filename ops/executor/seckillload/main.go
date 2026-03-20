// seckillload 提供秒杀链路压测与典型问题场景测试入口。
package main

import (
	"time"
)

func main() {
	cfg := parseFlags()
	if err := validateConfig(cfg); err != nil {
		fatalf("invalid config: %v", err)
	}

	tokens, err := loadTokens(cfg)
	if err != nil {
		fatalf("load tokens failed: %v", err)
	}

	client := newHTTPClient(cfg)
	start := time.Now()
	results := runScenario(client, cfg, tokens)
	elapsed := time.Since(start)

	s := summarize(results, elapsed)
	if cfg.Output == outputJSON {
		printSummaryJSON(cfg, s)
	} else {
		printSummary(cfg, s)
	}

	if err := assertScenario(cfg, s); err != nil {
		fatalf("%v", err)
	}
}
