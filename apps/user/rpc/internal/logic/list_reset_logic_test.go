// logic 包包含相关应用代码。
package logic

import (
	"context"
	"testing"

	"flashsale/apps/user/rpc/pb"
	"flashsale/pkg/base/errorx"
)

// --- ListUsers Tests ---

func TestListUsers_Success(t *testing.T) {
	svcCtx, _ := newTestSvcCtx(t)
	// 注册多个用户
	registerTestUser(t, svcCtx, "13200001111", "abc12345", "Alice")
	registerTestUser(t, svcCtx, "13200002222", "abc12345", "Bob")
	registerTestUser(t, svcCtx, "13200003333", "abc12345", "Charlie")

	logic := NewListUsersLogic(context.Background(), svcCtx)
	resp, err := logic.ListUsers(&pb.ListUsersReq{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list users failed: %v", err)
	}
	if resp.Total != 3 {
		t.Fatalf("expected total=3 got %d", resp.Total)
	}
	if len(resp.List) != 3 {
		t.Fatalf("expected list length=3 got %d", len(resp.List))
	}
	// 验证 UserView 字段
	for _, u := range resp.List {
		if u.UserId <= 0 {
			t.Fatal("user_id should be positive")
		}
		if u.Phone == "" {
			t.Fatal("phone should not be empty")
		}
		if u.Nickname == "" {
			t.Fatal("nickname should not be empty")
		}
	}
}

func TestListUsers_EmptyResult(t *testing.T) {
	svcCtx, _ := newTestSvcCtx(t)

	logic := NewListUsersLogic(context.Background(), svcCtx)
	resp, err := logic.ListUsers(&pb.ListUsersReq{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list users failed: %v", err)
	}
	if resp.Total != 0 {
		t.Fatalf("expected total=0 got %d", resp.Total)
	}
	if len(resp.List) != 0 {
		t.Fatalf("expected empty list got %d", len(resp.List))
	}
}

func TestListUsers_DefaultPagination(t *testing.T) {
	svcCtx, _ := newTestSvcCtx(t)
	registerTestUser(t, svcCtx, "13200004444", "abc12345", "Test")

	logic := NewListUsersLogic(context.Background(), svcCtx)
	// page=0, page_size=0 应使用默认值
	resp, err := logic.ListUsers(&pb.ListUsersReq{Page: 0, PageSize: 0})
	if err != nil {
		t.Fatalf("list users failed: %v", err)
	}
	if resp.Page != 1 {
		t.Fatalf("expected page=1 got %d", resp.Page)
	}
	if resp.PageSize != 20 {
		t.Fatalf("expected page_size=20 got %d", resp.PageSize)
	}
}

func TestListUsers_NilRequest(t *testing.T) {
	svcCtx, _ := newTestSvcCtx(t)
	logic := NewListUsersLogic(context.Background(), svcCtx)
	_, err := logic.ListUsers(nil)
	if err == nil {
		t.Fatal("should fail with nil request")
	}
}

// --- ResetUserPassword Tests ---

func TestResetUserPassword_Success(t *testing.T) {
	svcCtx, _ := newTestSvcCtx(t)
	uid := registerTestUser(t, svcCtx, "13300001111", "abc12345", "Reset")

	logic := NewResetUserPasswordLogic(context.Background(), svcCtx)
	resp, err := logic.ResetUserPassword(&pb.ResetUserPasswordReq{
		UserId:      uid,
		NewPassword: "resetPwd99",
	})
	if err != nil {
		t.Fatalf("reset password failed: %v", err)
	}
	if !resp.Success {
		t.Fatal("expected success=true")
	}

	// 验证新密码能登录
	login := NewLoginLogic(context.Background(), svcCtx)
	_, err = login.Login(&pb.LoginReq{Phone: "13300001111", Password: "resetPwd99"})
	if err != nil {
		t.Fatalf("login with reset password should succeed: %v", err)
	}
}

func TestResetUserPassword_UserNotFound(t *testing.T) {
	svcCtx, _ := newTestSvcCtx(t)

	logic := NewResetUserPasswordLogic(context.Background(), svcCtx)
	_, err := logic.ResetUserPassword(&pb.ResetUserPasswordReq{
		UserId:      999999,
		NewPassword: "resetPwd99",
	})
	if err == nil {
		t.Fatal("should fail with user not found")
	}
	appErr := errorx.FromError(err)
	if appErr.Code != errorx.CodeUserNotFound {
		t.Fatalf("expected %s got %s", errorx.CodeUserNotFound, appErr.Code)
	}
}

func TestResetUserPassword_WeakPassword(t *testing.T) {
	svcCtx, _ := newTestSvcCtx(t)
	uid := registerTestUser(t, svcCtx, "13300002222", "abc12345", "Weak")

	logic := NewResetUserPasswordLogic(context.Background(), svcCtx)
	_, err := logic.ResetUserPassword(&pb.ResetUserPasswordReq{
		UserId:      uid,
		NewPassword: "123",
	})
	if err == nil {
		t.Fatal("should fail with weak password")
	}
	appErr := errorx.FromError(err)
	if appErr.Code != errorx.CodeAuthWeakPassword {
		t.Fatalf("expected %s got %s", errorx.CodeAuthWeakPassword, appErr.Code)
	}
}

func TestResetUserPassword_NilRequest(t *testing.T) {
	svcCtx, _ := newTestSvcCtx(t)
	logic := NewResetUserPasswordLogic(context.Background(), svcCtx)
	_, err := logic.ResetUserPassword(nil)
	if err == nil {
		t.Fatal("should fail with nil request")
	}
}

func TestResetUserPassword_InvalidUserID(t *testing.T) {
	svcCtx, _ := newTestSvcCtx(t)
	logic := NewResetUserPasswordLogic(context.Background(), svcCtx)
	_, err := logic.ResetUserPassword(&pb.ResetUserPasswordReq{
		UserId:      -1,
		NewPassword: "resetPwd99",
	})
	if err == nil {
		t.Fatal("should fail with invalid user_id")
	}
}
