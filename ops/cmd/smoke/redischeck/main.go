package main

import (
	"flag"
	"time"

	redisprobe "flashsale/ops/backend/probe/redis"
)

func main() {
	configPath := flag.String("config", "configs/dev.yaml", "config file path")
	timeout := flag.Duration("timeout", 5*time.Second, "ping timeout")
	flag.Parse()

	if err := redisprobe.Run(*configPath, *timeout); err != nil {
		panic(err)
	}
}
