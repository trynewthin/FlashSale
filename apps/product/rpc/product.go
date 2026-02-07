// 商品 RPC 服务启动入口。
package main

import (
	"flag"
	"fmt"

	"flashsale/apps/product/rpc/internal/config"
	"flashsale/apps/product/rpc/internal/server"
	"flashsale/apps/product/rpc/internal/svc"
	"flashsale/apps/product/rpc/pb"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "apps/product/rpc/etc/product.yaml", "the config file")

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

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		pb.RegisterProductRpcServer(grpcServer, server.NewProductRpcServer(ctx))
		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting product rpc server at %s...\n", c.ListenOn)
	s.Start()
}
