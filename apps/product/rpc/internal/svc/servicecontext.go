// svc 包包含相关应用代码。
package svc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"flashsale/apps/product/rpc/internal/config"
	"flashsale/apps/product/rpc/internal/repository"
	baseauth "flashsale/pkg/base/authx"
	baseconfig "flashsale/pkg/base/config"
	baselog "flashsale/pkg/base/logx"
	"flashsale/pkg/base/mysqlx"
	basetracing "flashsale/pkg/base/tracing"
	"github.com/bwmarrin/snowflake"
	"go.uber.org/zap"
)

const defaultSnowflakeNode int64 = 2

// ServiceContext 封装商品 RPC 所需依赖。
type ServiceContext struct {
	Config        config.Config
	AppConfig     *baseconfig.AppConfig
	Logger        *zap.Logger
	DB            *sql.DB
	ProductRepo   repository.ProductRepository
	IDNode        *snowflake.Node
	traceShutdown func(context.Context) error
}

// NewServiceContext 初始化商品 RPC 上下文。
func NewServiceContext(c config.Config) (_ *ServiceContext, err error) {
	var (
		logger        *zap.Logger
		traceShutdown func(context.Context) error
		db            *sql.DB
	)
	defer func() {
		if err == nil {
			return
		}
		if db != nil {
			_ = db.Close()
		}
		if traceShutdown != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = traceShutdown(ctx)
			cancel()
		}
		if logger != nil {
			_ = logger.Sync()
		}
	}()

	appCfg, err := baseconfig.Load(c.BaseConfigPath)
	if err != nil {
		return nil, fmt.Errorf("load base config: %w", err)
	}

	logger, err = baselog.New(baselog.LogConfig{Service: appCfg.Service, Level: appCfg.Log.Level, Format: appCfg.Log.Format})
	if err != nil {
		return nil, fmt.Errorf("init logger: %w", err)
	}

	traceShutdown, err = basetracing.Init(basetracing.TraceConfig{
		Endpoint:    appCfg.OTEL.Endpoint,
		Insecure:    appCfg.OTEL.Insecure,
		ServiceName: "product-rpc",
	})
	if err != nil {
		return nil, fmt.Errorf("init tracing: %w", err)
	}

	mysqlCfg := appCfg.MySQL
	mysqlCfg.Database = "flash_product"
	db, err = mysqlx.Open(mysqlCfg)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := mysqlx.Ping(pingCtx, db); err != nil {
		return nil, fmt.Errorf("ping mysql: %w", err)
	}

	if err := baseauth.Init(baseauth.JWTConfig{
		User:  baseauth.JWTDomainConfig{Secret: appCfg.JWT.User.Secret, Issuer: appCfg.JWT.User.Issuer, Audience: appCfg.JWT.User.Audience, TTL: appCfg.JWT.User.TTL},
		Admin: baseauth.JWTDomainConfig{Secret: appCfg.JWT.Admin.Secret, Issuer: appCfg.JWT.Admin.Issuer, Audience: appCfg.JWT.Admin.Audience, TTL: appCfg.JWT.Admin.TTL},
	}); err != nil {
		return nil, fmt.Errorf("init auth: %w", err)
	}

	nodeID := c.SnowflakeNode
	if nodeID <= 0 {
		nodeID = defaultSnowflakeNode
	}
	node, err := snowflake.NewNode(nodeID)
	if err != nil {
		return nil, fmt.Errorf("new snowflake node: %w", err)
	}

	return &ServiceContext{
		Config:        c,
		AppConfig:     appCfg,
		Logger:        logger,
		DB:            db,
		ProductRepo:   repository.NewMySQLProductRepository(db),
		IDNode:        node,
		traceShutdown: traceShutdown,
	}, nil
}

// Close 释放 ServiceContext 管理的外部资源。
func (s *ServiceContext) Close() error {
	if s == nil {
		return nil
	}
	var errs []error
	if s.traceShutdown != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := s.traceShutdown(ctx)
		cancel()
		if err != nil {
			errs = append(errs, fmt.Errorf("shutdown tracing: %w", err))
		}
	}
	if s.DB != nil {
		if err := s.DB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close mysql: %w", err))
		}
	}
	if s.Logger != nil {
		if err := s.Logger.Sync(); err != nil && !strings.Contains(strings.ToLower(err.Error()), "invalid argument") {
			errs = append(errs, fmt.Errorf("sync logger: %w", err))
		}
	}
	return errors.Join(errs...)
}
