// svc 包包含相关应用代码。
package svc

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"flashsale/apps/gateway/user/internal/config"
	"flashsale/apps/order/rpc/orderrpc"
	"flashsale/apps/product/rpc/productrpc"
	"flashsale/apps/seckill/rpc/seckillrpc"
	"flashsale/apps/user/rpc/userrpc"
	baseauth "flashsale/pkg/base/authx"
	baseconfig "flashsale/pkg/base/config"
	baselog "flashsale/pkg/base/logx"
	"github.com/zeromicro/go-zero/zrpc"
	"go.uber.org/zap"
)

// ServiceContext 封装用户网关业务依赖。
type ServiceContext struct {
	Config                      config.Config
	AppConfig                   *baseconfig.AppConfig
	Logger                      *zap.Logger
	UserRPCCli                  userrpc.UserRpc
	ProductRPCCli               productrpc.ProductRpc
	OrderRPCCli                 orderrpc.OrderRpc
	SeckillRPCCli               seckillrpc.SeckillRpc
	seckillPurchaseRPCTimeout   time.Duration
	seckillTrackEventRPCTimeout time.Duration
}

// NewServiceContext 初始化用户网关依赖。
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
	orderRPCClient, err := zrpc.NewClient(c.OrderRPC)
	if err != nil {
		return nil, fmt.Errorf("init order rpc client: %w", err)
	}
	seckillRPCConf := c.SeckillRPC
	if c.SeckillRPCTimeoutMs > 0 {
		seckillRPCConf.Timeout = int64(c.SeckillRPCTimeoutMs)
	}
	seckillRPCClient, err := zrpc.NewClient(seckillRPCConf)
	if err != nil {
		return nil, fmt.Errorf("init seckill rpc client: %w", err)
	}

	return &ServiceContext{
		Config:                      c,
		AppConfig:                   appCfg,
		Logger:                      logger,
		UserRPCCli:                  userrpc.NewUserRpc(rpcClient),
		ProductRPCCli:               productrpc.NewProductRpc(productRPCClient),
		OrderRPCCli:                 orderrpc.NewOrderRpc(orderRPCClient),
		SeckillRPCCli:               seckillrpc.NewSeckillRpc(seckillRPCClient),
		seckillPurchaseRPCTimeout:   time.Duration(maxInt(c.SeckillPurchaseRPCTimeoutMs, 0)) * time.Millisecond,
		seckillTrackEventRPCTimeout: time.Duration(maxInt(c.SeckillTrackEventRPCTimeoutMs, 0)) * time.Millisecond,
	}, nil
}

// SeckillPurchaseRPCTimeout 返回秒杀购买 RPC 超时配置。
func (s *ServiceContext) SeckillPurchaseRPCTimeout() time.Duration {
	if s == nil || s.seckillPurchaseRPCTimeout <= 0 {
		return 8 * time.Second
	}
	return s.seckillPurchaseRPCTimeout
}

// SeckillTrackEventRPCTimeout 返回秒杀埋点 RPC 超时配置。
func (s *ServiceContext) SeckillTrackEventRPCTimeout() time.Duration {
	if s == nil || s.seckillTrackEventRPCTimeout <= 0 {
		return 4 * time.Second
	}
	return s.seckillTrackEventRPCTimeout
}

// Close 释放上下文中的外部资源。
func (s *ServiceContext) Close() error {
	if s == nil || s.Logger == nil {
		return nil
	}
	if err := s.Logger.Sync(); err != nil && !strings.Contains(strings.ToLower(err.Error()), "invalid argument") {
		return errors.New(err.Error())
	}
	return nil
}

func maxInt(v, fallback int) int {
	if v > 0 {
		return v
	}
	return fallback
}
