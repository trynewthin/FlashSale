// svc 包封装 media-store 的服务上下文依赖。
package svc

import (
	"fmt"
	"log/slog"
	"os"

	"flashsale/apps/media-store/internal/config"
	"flashsale/apps/media-store/internal/storage"
)

// ServiceContext 封装 media-store 所有依赖。
type ServiceContext struct {
	Config config.Config
	Logger *slog.Logger
	Store  storage.Store
}

// NewServiceContext 初始化服务上下文。
func NewServiceContext(cfg config.Config) (*ServiceContext, error) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	store, err := storage.NewLocalStore(cfg.StorageDir)
	if err != nil {
		return nil, fmt.Errorf("init storage: %w", err)
	}

	return &ServiceContext{
		Config: cfg,
		Logger: logger,
		Store:  store,
	}, nil
}
