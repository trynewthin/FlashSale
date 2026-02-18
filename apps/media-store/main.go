// media-store 文件管理服务启动入口。
//
// 与 cdn 容器配合使用：cdn 提供只读的静态文件对外服务，
// media-store 提供管理操作（上传/删除/列表），通过共享 volume 协作。
//
// 纯标准库实现，零外部依赖。
package main

import (
	"fmt"
	"net/http"
	"time"

	"flashsale/apps/media-store/internal/config"
	"flashsale/apps/media-store/internal/handler"
	"flashsale/apps/media-store/internal/middleware"
	"flashsale/apps/media-store/internal/svc"
)

func main() {
	cfg := config.Load()

	svcCtx, err := svc.NewServiceContext(cfg)
	if err != nil {
		panic(fmt.Sprintf("init media-store failed: %v", err))
	}

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, svcCtx)

	// 中间件链：CORS → 路由
	httpHandler := middleware.CORS(mux)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpHandler,
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       30 * time.Second, // 上传文件可能较慢
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	svcCtx.Logger.Info("media-store starting", "addr", cfg.Addr, "storage", cfg.StorageDir)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		panic(fmt.Sprintf("start media-store failed: %v", err))
	}
}
