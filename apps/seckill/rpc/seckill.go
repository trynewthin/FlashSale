// 秒杀 RPC 服务启动入口。
// rpc 命令提供可执行入口。
package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"flashsale/apps/seckill/rpc/internal/config"
	"flashsale/apps/seckill/rpc/internal/logic"
	"flashsale/apps/seckill/rpc/internal/server"
	"flashsale/apps/seckill/rpc/internal/svc"
	"flashsale/apps/seckill/rpc/pb"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "apps/seckill/rpc/etc/seckill.yaml", "the config file")

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
		// 避免启动瞬时竞争，延迟拉起消费者。
		time.Sleep(500 * time.Millisecond)
		logic.RunOrderStateConsumer(bgCtx, ctx)
	}()

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		pb.RegisterSeckillRpcServer(grpcServer, server.NewSeckillRpcServer(ctx))
		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting seckill rpc server at %s...\n", c.ListenOn)
	s.Start()
}
