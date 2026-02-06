// 用户 RPC 服务启动入口。
package main

import (
	"flag"
	"fmt"

	"flashsale/apps/user/rpc/internal/config"
	"flashsale/apps/user/rpc/internal/server"
	"flashsale/apps/user/rpc/internal/svc"
	"flashsale/apps/user/rpc/pb"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "apps/user/rpc/etc/user.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	ctx, err := svc.NewServiceContext(c)
	if err != nil {
		panic(fmt.Sprintf("init service context failed: %v", err))
	}
	defer func() {
		_ = ctx.Close()
	}()

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		pb.RegisterUserRpcServer(grpcServer, server.NewUserRpcServer(ctx))
		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting user rpc server at %s...\n", c.ListenOn)
	s.Start()
}
