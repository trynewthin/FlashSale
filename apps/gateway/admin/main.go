// 管理员网关启动入口。
package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"flashsale/apps/gateway/admin/internal/config"
	"flashsale/apps/gateway/admin/internal/handler"
	"flashsale/apps/gateway/admin/internal/svc"
	basemiddleware "flashsale/pkg/base/middleware"
	"github.com/zeromicro/go-zero/core/conf"
)

var configFile = flag.String("f", "apps/gateway/admin/etc/admin-gateway.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	config.ApplyEnvOverrides(&c)

	svcCtx, err := svc.NewServiceContext(c)
	if err != nil {
		panic(fmt.Sprintf("init admin gateway failed: %v", err))
	}
	defer func() {
		_ = svcCtx.Close()
	}()

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, svcCtx)
	httpHandler := http.Handler(mux)
	httpHandler = basemiddleware.RequestLogger(svcCtx.Logger, httpHandler)
	httpHandler = basemiddleware.Recover(svcCtx.Logger, httpHandler)
	httpHandler = basemiddleware.Trace(httpHandler)

	srv := &http.Server{
		Addr:              c.ListenOn,
		Handler:           httpHandler,
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	fmt.Printf("starting admin gateway at %s...\n", c.ListenOn)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		panic(fmt.Sprintf("start admin gateway failed: %v", err))
	}
}
