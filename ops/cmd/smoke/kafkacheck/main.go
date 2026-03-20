package main

import (
	"flag"
	"time"

	kafkaprobe "flashsale/ops/backend/probe/kafka"
)

func main() {
	configPath := flag.String("config", "configs/dev.yaml", "config file path")
	timeout := flag.Duration("timeout", 20*time.Second, "consume timeout")
	flag.Parse()

	if err := kafkaprobe.Run(*configPath, *timeout); err != nil {
		panic(err)
	}
}
