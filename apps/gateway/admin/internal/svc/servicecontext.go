// Package svc 负责组装管理员网关运行时依赖。
package svc

import (
	"fmt"
	"strings"

	"flashsale/apps/gateway/admin/internal/authz"
	"flashsale/apps/gateway/admin/internal/config"
	"flashsale/apps/product/rpc/productrpc"
	"flashsale/apps/user/rpc/userrpc"
	baseauth "flashsale/pkg/base/authx"
	baseconfig "flashsale/pkg/base/config"
	baselog "flashsale/pkg/base/logx"
	"github.com/zeromicro/go-zero/zrpc"
	"go.uber.org/zap"
)

// ServiceContext 封装管理员网关依赖。
type ServiceContext struct {
	Config        config.Config
	AppConfig     *baseconfig.AppConfig
	Logger        *zap.Logger
	Authorizer    authz.Authorizer
	UserRPCCli    userrpc.UserRpc
	ProductRPCCli productrpc.ProductRpc
}

// NewServiceContext 初始化管理员网关依赖。
func NewServiceContext(c config.Config) (*ServiceContext, error) {
	appCfg, err := baseconfig.Load(c.BaseConfigPath)
	if err != nil {
		return nil, fmt.Errorf("load base config: %w", err)
	}

	logger, err := baselog.New(baselog.LogConfig{
		Service: c.Name,
		Level:   appCfg.Log.Level,
		Format:  appCfg.Log.Format,
	})
	if err != nil {
		return nil, fmt.Errorf("init logger: %w", err)
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
		return nil, fmt.Errorf("init auth: %w", err)
	}

	rpcClient, err := zrpc.NewClient(c.UserRPC)
	if err != nil {
		return nil, fmt.Errorf("init user rpc client: %w", err)
	}
	productRPCClient, err := zrpc.NewClient(c.ProductRPC)
	if err != nil {
		return nil, fmt.Errorf("init product rpc client: %w", err)
	}

	return &ServiceContext{
		Config:        c,
		AppConfig:     appCfg,
		Logger:        logger,
		Authorizer:    authz.NewStaticAuthorizer(),
		UserRPCCli:    userrpc.NewUserRpc(rpcClient),
		ProductRPCCli: productrpc.NewProductRpc(productRPCClient),
	}, nil
}

// Close 释放上下文资源。
func (s *ServiceContext) Close() error {
	if s == nil || s.Logger == nil {
		return nil
	}
	if err := s.Logger.Sync(); err != nil && !strings.Contains(strings.ToLower(err.Error()), "invalid argument") {
		return fmt.Errorf("sync logger: %w", err)
	}
	return nil
}
