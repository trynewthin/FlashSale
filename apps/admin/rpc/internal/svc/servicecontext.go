// svc 包包含相关应用代码。
package svc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"flashsale/apps/admin/rpc/internal/config"
	"flashsale/apps/admin/rpc/internal/model"
	"flashsale/apps/admin/rpc/internal/repository"
	baseauth "flashsale/pkg/base/authx"
	baseconfig "flashsale/pkg/base/config"
	baselog "flashsale/pkg/base/logx"
	"flashsale/pkg/base/mysqlx"
	basetracing "flashsale/pkg/base/tracing"
	"github.com/bwmarrin/snowflake"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const (
	defaultSnowflakeNode int64 = 7
)

// ServiceContext 封装管理员 RPC 依赖。
type ServiceContext struct {
	Config          config.Config
	AppConfig       *baseconfig.AppConfig
	Logger          *zap.Logger
	DB              *sql.DB
	AdminRepo       repository.AdminRepository
	IDNode          *snowflake.Node
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	traceShutdown func(context.Context) error
}

// NewServiceContext 初始化管理员 RPC 依赖。
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

	logger, err = baselog.New(baselog.LogConfig{Service: "admin-rpc", Level: appCfg.Log.Level, Format: appCfg.Log.Format})
	if err != nil {
		return nil, fmt.Errorf("init logger: %w", err)
	}

	traceShutdown, err = basetracing.Init(basetracing.TraceConfig{Endpoint: appCfg.OTEL.Endpoint, Insecure: appCfg.OTEL.Insecure, ServiceName: "admin-rpc"})
	if err != nil {
		return nil, fmt.Errorf("init tracing: %w", err)
	}

	mysqlCfg := appCfg.MySQL
	mysqlCfg.Database = "flash_admin"
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

	svcCtx := &ServiceContext{
		Config:          c,
		AppConfig:       appCfg,
		Logger:          logger,
		DB:              db,
		AdminRepo:       repository.NewMySQLAdminRepository(db),
		IDNode:          node,
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		traceShutdown:   traceShutdown,
	}

	if err := svcCtx.bootstrapSuperAdmin(context.Background()); err != nil {
		return nil, err
	}

	return svcCtx, nil
}

func (s *ServiceContext) bootstrapSuperAdmin(ctx context.Context) error {
	if s == nil || s.AdminRepo == nil {
		return nil
	}
	if !parseEnvBool("ADMIN_BOOTSTRAP_ENABLED") {
		return nil
	}
	total, err := s.AdminRepo.CountAdmins(ctx)
	if err != nil {
		return fmt.Errorf("count admins for bootstrap: %w", err)
	}
	if total > 0 {
		return nil
	}
	username := strings.TrimSpace(os.Getenv("ADMIN_BOOTSTRAP_USERNAME"))
	password := strings.TrimSpace(os.Getenv("ADMIN_BOOTSTRAP_PASSWORD"))
	displayName := strings.TrimSpace(os.Getenv("ADMIN_BOOTSTRAP_DISPLAY_NAME"))
	if username == "" || password == "" {
		return fmt.Errorf("bootstrap admin username/password required")
	}
	if displayName == "" {
		displayName = "超级管理员"
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash bootstrap password: %w", err)
	}
	now := time.Now()
	adminID := s.IDNode.Generate().Int64()
	roleID := s.IDNode.Generate().Int64()
	if err := s.AdminRepo.CreateAdmin(ctx, &model.Admin{
		ID:           adminID,
		Username:     username,
		DisplayName:  displayName,
		PasswordHash: string(hashed),
		Status:       model.AdminStatusEnabled,
		DataScope:    model.DataScopeAll,
		IsSuperAdmin: true,
	}); err != nil {
		if !errors.Is(err, repository.ErrDuplicateUsername) {
			return fmt.Errorf("create bootstrap admin: %w", err)
		}
		existing, findErr := s.AdminRepo.FindAdminByUsername(ctx, username)
		if findErr != nil {
			return fmt.Errorf("find bootstrap admin: %w", findErr)
		}
		adminID = existing.ID
	}
	if err := s.AdminRepo.CreateRole(ctx, &model.Role{
		ID:       roleID,
		RoleCode: "super_admin",
		RoleName: "超级管理员",
		Status:   model.RoleStatusEnabled,
		IsSystem: true,
	}); err != nil {
		if !errors.Is(err, repository.ErrDuplicateRoleCode) {
			return fmt.Errorf("create bootstrap role: %w", err)
		}
	}
	if role, err := s.AdminRepo.FindRoleByCode(ctx, "super_admin"); err == nil && role != nil {
		roleID = role.ID
	}
	if err := s.AdminRepo.ReplaceRoleDomains(ctx, roleID, []string{"operations", "user_management", "product_management", "order_management", "seckill_management", "admin_management"}); err != nil {
		return fmt.Errorf("bootstrap replace role domains: %w", err)
	}
	if err := s.AdminRepo.ReplaceAdminRoles(ctx, adminID, []int64{roleID}); err != nil {
		return fmt.Errorf("bootstrap replace admin roles: %w", err)
	}
	if err := s.AdminRepo.CreateAuditLog(ctx, &model.AuditLog{
		ID:         s.IDNode.Generate().Int64(),
		AdminID:    adminID,
		Action:     "bootstrap_super_admin",
		TargetType: "admin",
		TargetID:   adminID,
		Result:     "success",
		DetailJSON: "{}",
		CreatedAt:  now,
	}); err != nil {
		return fmt.Errorf("bootstrap write audit log: %w", err)
	}
	return nil
}

// Close 释放 ServiceContext 管理资源。
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

func parseEnvBool(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}
