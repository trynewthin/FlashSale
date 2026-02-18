// logic 包包含相关应用代码。
package logic

import (
	"context"
	"testing"
	"time"

	"flashsale/apps/user/rpc/internal/model"
	"flashsale/apps/user/rpc/internal/repository"
	"flashsale/apps/user/rpc/internal/svc"
	"flashsale/apps/user/rpc/pb"
	baseauth "flashsale/pkg/base/authx"
	"flashsale/pkg/base/errorx"

	"github.com/bwmarrin/snowflake"
	mysqlDriver "github.com/go-sql-driver/mysql"
)

// memoryUserRepo 是逻辑测试使用的内存仓储。
type memoryUserRepo struct {
	usersByPhone map[string]*model.User
	usersByID    map[int64]*model.User
	deleted      map[int64]bool
}

// newMemoryUserRepo 创建内存仓储。
func newMemoryUserRepo() *memoryUserRepo {
	return &memoryUserRepo{
		usersByPhone: make(map[string]*model.User),
		usersByID:    make(map[int64]*model.User),
		deleted:      make(map[int64]bool),
	}
}

// Create 创建用户。
func (m *memoryUserRepo) Create(_ context.Context, user *model.User) error {
	if _, ok := m.usersByPhone[user.Phone]; ok {
		return &mysqlDriver.MySQLError{Number: 1062, Message: "duplicate"}
	}
	cp := *user
	m.usersByPhone[user.Phone] = &cp
	m.usersByID[user.ID] = &cp
	return nil
}

// FindByPhone 按手机号查询用户。
func (m *memoryUserRepo) FindByPhone(_ context.Context, phone string) (*model.User, error) {
	if user, ok := m.usersByPhone[phone]; ok {
		if m.deleted[user.ID] {
			return nil, repository.ErrUserNotFound
		}
		cp := *user
		return &cp, nil
	}
	return nil, repository.ErrUserNotFound
}

// FindByID 按用户 ID 查询用户。
func (m *memoryUserRepo) FindByID(_ context.Context, userID int64) (*model.User, error) {
	if user, ok := m.usersByID[userID]; ok {
		if m.deleted[userID] {
			return nil, repository.ErrUserNotFound
		}
		cp := *user
		return &cp, nil
	}
	return nil, repository.ErrUserNotFound
}

// UpdateNickname 更新昵称。
func (m *memoryUserRepo) UpdateNickname(_ context.Context, userID int64, nickname string) error {
	if user, ok := m.usersByID[userID]; ok {
		if m.deleted[userID] {
			return repository.ErrUserNotFound
		}
		user.Nickname = nickname
		return nil
	}
	return repository.ErrUserNotFound
}

// UpdateLoginAudit 更新登录审计信息。
func (m *memoryUserRepo) UpdateLoginAudit(_ context.Context, userID int64, ip string, at time.Time) error {
	if user, ok := m.usersByID[userID]; ok {
		if m.deleted[userID] {
			return repository.ErrUserNotFound
		}
		user.LastLoginAt = &at
		user.LastLoginIP = ip
		return nil
	}
	return repository.ErrUserNotFound
}

// SoftDelete 软删除用户。
func (m *memoryUserRepo) SoftDelete(_ context.Context, userID int64, at time.Time) error {
	if user, ok := m.usersByID[userID]; ok {
		user.DeletedAt = &at
		user.Status = 0
		m.deleted[userID] = true
		return nil
	}
	return repository.ErrUserNotFound
}

// UpdatePasswordHash 更新密码哈希。
func (m *memoryUserRepo) UpdatePasswordHash(_ context.Context, userID int64, passwordHash string) error {
	if user, ok := m.usersByID[userID]; ok {
		if m.deleted[userID] {
			return repository.ErrUserNotFound
		}
		user.PasswordHash = passwordHash
		return nil
	}
	return repository.ErrUserNotFound
}

// ListUsers 分页查询用户（测试桩）。
func (m *memoryUserRepo) ListUsers(_ context.Context, _ repository.UserListQuery) ([]*model.User, int64, error) {
	var users []*model.User
	for _, u := range m.usersByID {
		if !m.deleted[u.ID] {
			cp := *u
			users = append(users, &cp)
		}
	}
	return users, int64(len(users)), nil
}

// TestRegisterAndLoginFlow 覆盖注册、登录与 token 域隔离。
func TestRegisterAndLoginFlow(t *testing.T) {
	if err := baseauth.Init(baseauth.JWTConfig{
		User:  baseauth.JWTDomainConfig{Secret: "user-secret", Issuer: "user-iss", Audience: "user-aud", TTL: time.Hour},
		Admin: baseauth.JWTDomainConfig{Secret: "admin-secret", Issuer: "admin-iss", Audience: "admin-aud", TTL: time.Hour},
	}); err != nil {
		t.Fatalf("init auth failed: %v", err)
	}
	node, _ := snowflake.NewNode(1)
	repo := newMemoryUserRepo()
	svcCtx := &svc.ServiceContext{UserRepo: repo, IDNode: node, AccessTokenTTL: time.Hour}
	ctx := context.Background()

	reg := NewRegisterLogic(ctx, svcCtx)
	regResp, err := reg.Register(&pb.RegisterReq{Phone: "13800138000", Password: "abc12345", Nickname: "测试用户", ClientIp: "127.0.0.1"})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if regResp.AccessToken == "" {
		t.Fatalf("register token should not be empty")
	}
	if _, err := baseauth.Parse(baseauth.TokenTypeUser, regResp.AccessToken); err != nil {
		t.Fatalf("parse user token failed: %v", err)
	}
	if _, err := baseauth.Parse(baseauth.TokenTypeAdmin, regResp.AccessToken); err == nil {
		t.Fatalf("admin domain should not parse user token")
	}

	login := NewLoginLogic(ctx, svcCtx)
	loginResp, err := login.Login(&pb.LoginReq{Phone: "13800138000", Password: "abc12345", ClientIp: "10.0.0.1"})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if loginResp.AccessToken == "" {
		t.Fatalf("login token should not be empty")
	}

	_, err = login.Login(&pb.LoginReq{Phone: "13800138000", Password: "wrong123", ClientIp: "10.0.0.2"})
	if err == nil {
		t.Fatalf("login with wrong password should fail")
	}
	appErr := errorx.FromError(err)
	if appErr.Code != errorx.CodeAuthInvalidCredentials {
		t.Fatalf("expected %s got %s", errorx.CodeAuthInvalidCredentials, appErr.Code)
	}
}

// TestRegisterDuplicatePhone 覆盖重复手机号注册场景。
func TestRegisterDuplicatePhone(t *testing.T) {
	if err := baseauth.Init(baseauth.JWTConfig{
		User:  baseauth.JWTDomainConfig{Secret: "user-secret2", Issuer: "user-iss2", Audience: "user-aud2", TTL: time.Hour},
		Admin: baseauth.JWTDomainConfig{Secret: "admin-secret2", Issuer: "admin-iss2", Audience: "admin-aud2", TTL: time.Hour},
	}); err != nil {
		t.Fatalf("init auth failed: %v", err)
	}
	node, _ := snowflake.NewNode(2)
	repo := newMemoryUserRepo()
	svcCtx := &svc.ServiceContext{UserRepo: repo, IDNode: node, AccessTokenTTL: time.Hour}
	logic := NewRegisterLogic(context.Background(), svcCtx)

	if _, err := logic.Register(&pb.RegisterReq{Phone: "13900139000", Password: "abc12345", Nickname: "u1"}); err != nil {
		t.Fatalf("first register failed: %v", err)
	}
	_, err := logic.Register(&pb.RegisterReq{Phone: "13900139000", Password: "abc12345", Nickname: "u2"})
	if err == nil {
		t.Fatalf("duplicate register should fail")
	}
	appErr := errorx.FromError(err)
	if appErr.Code != errorx.CodeAuthPhoneAlreadyRegistered {
		t.Fatalf("expected %s got %s", errorx.CodeAuthPhoneAlreadyRegistered, appErr.Code)
	}
}

// TestDeleteUserFlow 覆盖用户软删除流程。
func TestDeleteUserFlow(t *testing.T) {
	if err := baseauth.Init(baseauth.JWTConfig{
		User:  baseauth.JWTDomainConfig{Secret: "user-secret3", Issuer: "user-iss3", Audience: "user-aud3", TTL: time.Hour},
		Admin: baseauth.JWTDomainConfig{Secret: "admin-secret3", Issuer: "admin-iss3", Audience: "admin-aud3", TTL: time.Hour},
	}); err != nil {
		t.Fatalf("init auth failed: %v", err)
	}
	node, _ := snowflake.NewNode(3)
	repo := newMemoryUserRepo()
	svcCtx := &svc.ServiceContext{UserRepo: repo, IDNode: node, AccessTokenTTL: time.Hour}
	ctx := context.Background()

	reg := NewRegisterLogic(ctx, svcCtx)
	regResp, err := reg.Register(&pb.RegisterReq{Phone: "13700137000", Password: "abc12345", Nickname: "to-delete"})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	del := NewDeleteUserLogic(ctx, svcCtx)
	if _, err := del.DeleteUser(&pb.DeleteUserReq{UserId: regResp.UserId}); err != nil {
		t.Fatalf("delete user failed: %v", err)
	}

	profile := NewGetProfileLogic(ctx, svcCtx)
	_, err = profile.GetProfile(&pb.GetProfileReq{UserId: regResp.UserId})
	if err == nil {
		t.Fatalf("get profile after delete should fail")
	}
	appErr := errorx.FromError(err)
	if appErr.Code != errorx.CodeUserNotFound {
		t.Fatalf("expected %s got %s", errorx.CodeUserNotFound, appErr.Code)
	}
}
