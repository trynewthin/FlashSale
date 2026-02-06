// Package svc 负责组装用户 RPC 运行时依赖。
package svc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"flashsale/apps/user/rpc/internal/config"
	"flashsale/apps/user/rpc/internal/repository"
	baseauth "flashsale/pkg/base/authx"
	baseconfig "flashsale/pkg/base/config"
	baselog "flashsale/pkg/base/logx"
	"flashsale/pkg/base/mysqlx"
	basetracing "flashsale/pkg/base/tracing"
	"github.com/bwmarrin/snowflake"
	"go.uber.org/zap"
)

const (
	defaultAccessTokenTTL = 24 * time.Hour
	defaultSnowflakeNode  = 1
)

// ServiceContext 封装业务逻辑层所需依赖。
type ServiceContext struct {
	Config         config.Config
	AppConfig      *baseconfig.AppConfig
	Logger         *zap.Logger
	DB             *sql.DB
	UserRepo       repository.UserRepository
	IDNode         *snowflake.Node
	AccessTokenTTL time.Duration
	traceShutdown  func(context.Context) error
}

// NewServiceContext 初始化服务上下文并加载基础层依赖。
func NewServiceContext(c config.Config) (*ServiceContext, error) {
	appCfg, err := baseconfig.Load(c.BaseConfigPath)
	if err != nil {
		return nil, fmt.Errorf("load base config: %w", err)
	}

	logger, err := baselog.New(baselog.LogConfig{
		Service: appCfg.Service,
		Level:   appCfg.Log.Level,
		Format:  appCfg.Log.Format,
	})
	if err != nil {
		return nil, fmt.Errorf("init logger: %w", err)
	}

	traceShutdown, err := basetracing.Init(basetracing.TraceConfig{
		Endpoint:    appCfg.OTEL.Endpoint,
		Insecure:    appCfg.OTEL.Insecure,
		ServiceName: appCfg.OTEL.ServiceName,
	})
	if err != nil {
		return nil, fmt.Errorf("init tracing: %w", err)
	}

	db, err := mysqlx.Open(appCfg.MySQL)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := mysqlx.Ping(pingCtx, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}

	if err := baseauth.Init(baseauth.JWTConfig{
		User: baseauth.JWTDomainConfig{
			Secret:   appCfg.JWT.User.Secret,
			Issuer:   appCfg.JWT.User.Issuer,
			Audience: appCfg.JWT.User.Audience,
			TTL:      appCfg.JWT.User.TTL,
		},
		Admin: baseauth.JWTDomainConfig{
			Secret:   appCfg.JWT.Admin.Secret,
			Issuer:   appCfg.JWT.Admin.Issuer,
			Audience: appCfg.JWT.Admin.Audience,
			TTL:      appCfg.JWT.Admin.TTL,
		},
	}); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init auth: %w", err)
	}

	nodeID := c.SnowflakeNode
	if nodeID <= 0 {
		nodeID = defaultSnowflakeNode
	}
	node, err := snowflake.NewNode(nodeID)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("new snowflake node: %w", err)
	}

	ttl := c.AccessTokenTTL
	if ttl <= 0 {
		ttl = appCfg.JWT.User.TTL
	}
	if ttl <= 0 {
		ttl = defaultAccessTokenTTL
	}

	return &ServiceContext{
		Config:         c,
		AppConfig:      appCfg,
		Logger:         logger,
		DB:             db,
		UserRepo:       repository.NewMySQLUserRepository(db),
		IDNode:         node,
		AccessTokenTTL: ttl,
		traceShutdown:  traceShutdown,
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
