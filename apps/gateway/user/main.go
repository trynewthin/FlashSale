// 用户网关启动入口。
package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"flashsale/apps/gateway/user/internal/config"
	"flashsale/apps/gateway/user/internal/handler"
	"flashsale/apps/gateway/user/internal/svc"
	"github.com/zeromicro/go-zero/core/conf"
)

var configFile = flag.String("f", "apps/gateway/user/etc/user-gateway.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	svcCtx, err := svc.NewServiceContext(c)
	if err != nil {
		panic(fmt.Sprintf("init user gateway failed: %v", err))
	}
	defer func() {
		_ = svcCtx.Close()
	}()

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, svcCtx)

	srv := &http.Server{
		Addr:              c.ListenOn,
		Handler:           mux,
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	fmt.Printf("starting user gateway at %s...\n", c.ListenOn)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		panic(fmt.Sprintf("start user gateway failed: %v", err))
	}
}
