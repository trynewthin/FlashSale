package main

import (
	"flag"
	"time"

	mysqlprobe "flashsale/ops/backend/probe/mysql"
)

func main() {
	configPath := flag.String("config", "configs/dev.yaml", "config file path")
	timeout := flag.Duration("timeout", 5*time.Second, "ping timeout")
	flag.Parse()

	if err := mysqlprobe.Run(*configPath, *timeout); err != nil {
		panic(err)
	}
}
