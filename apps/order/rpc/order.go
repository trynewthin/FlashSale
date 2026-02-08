// 订单 RPC 服务启动入口。
// rpc 命令提供可执行入口。
package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"flashsale/apps/order/rpc/internal/config"
	"flashsale/apps/order/rpc/internal/logic"
	"flashsale/apps/order/rpc/internal/server"
	"flashsale/apps/order/rpc/internal/svc"
	"flashsale/apps/order/rpc/pb"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "apps/order/rpc/etc/order.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	config.ApplyEnvOverrides(&c)

	ctx, err := svc.NewServiceContext(c)
	if err != nil {
		panic(fmt.Sprintf("init service context failed: %v", err))
	}
	defer func() {
		_ = ctx.Close()
	}()
	bgCtx, bgCancel := context.WithCancel(context.Background())
	defer bgCancel()
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		job := logic.NewTimeoutJobLogic(bgCtx, ctx)
		for {
			select {
			case <-bgCtx.Done():
				return
			case <-ticker.C:
				job.RunOnce()
			}
		}
	}()

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		pb.RegisterOrderRpcServer(grpcServer, server.NewOrderRpcServer(ctx))
		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting order rpc server at %s...\n", c.ListenOn)
	s.Start()
}
