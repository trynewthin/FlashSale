// logic 包包含相关应用代码。
package logic

import (
	"context"
	"testing"
	"time"

	"flashsale/apps/user/rpc/internal/svc"
	"flashsale/apps/user/rpc/pb"
	baseauth "flashsale/pkg/base/authx"
	"flashsale/pkg/base/errorx"

	"github.com/bwmarrin/snowflake"
)

// newTestSvcCtx 创建测试用 ServiceContext。
func newTestSvcCtx(t *testing.T) (*svc.ServiceContext, *memoryUserRepo) {
	t.Helper()
	if err := baseauth.Init(baseauth.JWTConfig{
		User:  baseauth.JWTDomainConfig{Secret: "test-secret", Issuer: "test-iss", Audience: "test-aud", TTL: time.Hour},
		Admin: baseauth.JWTDomainConfig{Secret: "admin-secret", Issuer: "admin-iss", Audience: "admin-aud", TTL: time.Hour},
	}); err != nil {
		t.Fatalf("init auth failed: %v", err)
	}
	node, _ := snowflake.NewNode(10)
	repo := newMemoryUserRepo()
	return &svc.ServiceContext{UserRepo: repo, IDNode: node, AccessTokenTTL: time.Hour}, repo
}

// registerTestUser 注册一个测试用户并返回其 ID。
func registerTestUser(t *testing.T, svcCtx *svc.ServiceContext, phone, password, nickname string) int64 {
	t.Helper()
	reg := NewRegisterLogic(context.Background(), svcCtx)
	resp, err := reg.Register(&pb.RegisterReq{Phone: phone, Password: password, Nickname: nickname})
	if err != nil {
		t.Fatalf("register test user failed: %v", err)
	}
	return resp.UserId
}

// --- ChangePassword Tests ---

func TestChangePassword_Success(t *testing.T) {
	svcCtx, _ := newTestSvcCtx(t)
	uid := registerTestUser(t, svcCtx, "13100001111", "abc12345", "测试")

	logic := NewChangePasswordLogic(context.Background(), svcCtx)
	resp, err := logic.ChangePassword(&pb.ChangePasswordReq{
		UserId:      uid,
		OldPassword: "abc12345",
		NewPassword: "newPass99",
	})
	if err != nil {
		t.Fatalf("change password failed: %v", err)
	}
	if !resp.Success {
		t.Fatal("expected success=true")
	}

	// 验证新密码能登录
	login := NewLoginLogic(context.Background(), svcCtx)
	_, err = login.Login(&pb.LoginReq{Phone: "13100001111", Password: "newPass99"})
	if err != nil {
		t.Fatalf("login with new password should succeed: %v", err)
	}

	// 验证旧密码不能登录
	_, err = login.Login(&pb.LoginReq{Phone: "13100001111", Password: "abc12345"})
	if err == nil {
		t.Fatal("login with old password should fail")
	}
}

func TestChangePassword_WrongOldPassword(t *testing.T) {
	svcCtx, _ := newTestSvcCtx(t)
	uid := registerTestUser(t, svcCtx, "13100002222", "abc12345", "测试")

	logic := NewChangePasswordLogic(context.Background(), svcCtx)
	_, err := logic.ChangePassword(&pb.ChangePasswordReq{
		UserId:      uid,
		OldPassword: "wrongPwd1",
		NewPassword: "newPass99",
	})
	if err == nil {
		t.Fatal("should fail with wrong old password")
	}
	appErr := errorx.FromError(err)
	if appErr.Code != errorx.CodeAuthInvalidCredentials {
		t.Fatalf("expected %s got %s", errorx.CodeAuthInvalidCredentials, appErr.Code)
	}
}

func TestChangePassword_SameAsOld(t *testing.T) {
	svcCtx, _ := newTestSvcCtx(t)
	uid := registerTestUser(t, svcCtx, "13100003333", "abc12345", "测试")

	logic := NewChangePasswordLogic(context.Background(), svcCtx)
	_, err := logic.ChangePassword(&pb.ChangePasswordReq{
		UserId:      uid,
		OldPassword: "abc12345",
		NewPassword: "abc12345",
	})
	if err == nil {
		t.Fatal("should fail when new == old")
	}
	appErr := errorx.FromError(err)
	if appErr.Code != errorx.CodeSysBadRequest {
		t.Fatalf("expected %s got %s", errorx.CodeSysBadRequest, appErr.Code)
	}
}

func TestChangePassword_WeakNewPassword(t *testing.T) {
	svcCtx, _ := newTestSvcCtx(t)
	uid := registerTestUser(t, svcCtx, "13100004444", "abc12345", "测试")

	logic := NewChangePasswordLogic(context.Background(), svcCtx)
	_, err := logic.ChangePassword(&pb.ChangePasswordReq{
		UserId:      uid,
		OldPassword: "abc12345",
		NewPassword: "123",
	})
	if err == nil {
		t.Fatal("should fail with weak new password")
	}
	appErr := errorx.FromError(err)
	if appErr.Code != errorx.CodeAuthWeakPassword {
		t.Fatalf("expected %s got %s", errorx.CodeAuthWeakPassword, appErr.Code)
	}
}

func TestChangePassword_UserNotFound(t *testing.T) {
	svcCtx, _ := newTestSvcCtx(t)

	logic := NewChangePasswordLogic(context.Background(), svcCtx)
	_, err := logic.ChangePassword(&pb.ChangePasswordReq{
		UserId:      999999,
		OldPassword: "abc12345",
		NewPassword: "newPass99",
	})
	if err == nil {
		t.Fatal("should fail with user not found")
	}
	appErr := errorx.FromError(err)
	if appErr.Code != errorx.CodeUserNotFound {
		t.Fatalf("expected %s got %s", errorx.CodeUserNotFound, appErr.Code)
	}
}

func TestChangePassword_NilRequest(t *testing.T) {
	svcCtx, _ := newTestSvcCtx(t)
	logic := NewChangePasswordLogic(context.Background(), svcCtx)
	_, err := logic.ChangePassword(nil)
	if err == nil {
		t.Fatal("should fail with nil request")
	}
}

func TestChangePassword_InvalidUserID(t *testing.T) {
	svcCtx, _ := newTestSvcCtx(t)
	logic := NewChangePasswordLogic(context.Background(), svcCtx)
	_, err := logic.ChangePassword(&pb.ChangePasswordReq{
		UserId:      0,
		OldPassword: "abc12345",
		NewPassword: "newPass99",
	})
	if err == nil {
		t.Fatal("should fail with invalid user_id")
	}
}
