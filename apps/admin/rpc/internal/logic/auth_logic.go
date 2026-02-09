// logic 包包含相关应用代码。
package logic

import (
	"strings"
	"time"

	"flashsale/apps/admin/rpc/internal/model"
	"flashsale/apps/admin/rpc/internal/repository"
	"flashsale/apps/admin/rpc/pb"
	"flashsale/pkg/base/errorx"
)

const (
	maxLoginFailures = 5
	lockDuration     = 15 * time.Minute
)

// AdminLogin 处理管理员登录。
func (l *AdminLogic) AdminLogin(in *pb.AdminLoginReq) (*pb.AdminAuthResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	username, err := normalizeUsername(in.Username)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Password) == "" {
		return nil, errorx.New(errorx.CodeAuthInvalidCredentials, "账号或密码错误")
	}
	admin, err := l.svcCtx.AdminRepo.FindAdminByUsername(l.ctx, username)
	if err != nil {
		if err == repository.ErrAdminNotFound {
			writeAudit(l.ctx, l.svcCtx, 0, "admin_login", "admin", 0, "failed", map[string]any{"username": username, "reason": "not_found"})
			return nil, errorx.New(errorx.CodeAuthInvalidCredentials, "账号或密码错误")
		}
		return nil, mapRepoErr(err)
	}
	now := time.Now()
	if err := checkLocked(admin, now); err != nil {
		writeAudit(l.ctx, l.svcCtx, admin.ID, "admin_login", "admin", admin.ID, "failed", map[string]any{"reason": err.Error()})
		return nil, err
	}
	if !comparePassword(admin.PasswordHash, in.Password) {
		failedCount := admin.FailedLoginCount + 1
		var lockedUntil *time.Time
		if failedCount >= maxLoginFailures {
			until := now.Add(lockDuration)
			lockedUntil = &until
		}
		_ = l.svcCtx.AdminRepo.SetAdminLoginFailure(l.ctx, admin.ID, failedCount, lockedUntil)
		writeAudit(l.ctx, l.svcCtx, admin.ID, "admin_login", "admin", admin.ID, "failed", map[string]any{"reason": "invalid_password"})
		if lockedUntil != nil {
			return nil, errorx.New(errorx.CodeAdminAccountLocked, "管理员账号已锁定")
		}
		return nil, errorx.New(errorx.CodeAuthInvalidCredentials, "账号或密码错误")
	}

	if err := l.svcCtx.AdminRepo.ResetAdminLoginFailures(l.ctx, admin.ID, now, strings.TrimSpace(in.ClientIp)); err != nil {
		return nil, mapRepoErr(err)
	}
	admin.LastLoginAt = &now
	admin.LastLoginIP = strings.TrimSpace(in.ClientIp)
	if err := setAdminRelations(l.ctx, l.svcCtx.AdminRepo, admin); err != nil {
		return nil, mapRepoErr(err)
	}
	accessToken, accessTTL, err := issueAdminAccessToken(l.svcCtx, admin)
	if err != nil {
		return nil, err
	}
	tokenID, refreshToken, tokenHash, err := buildRefreshToken()
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeSysInternal, "生成 refresh token 失败", err)
	}
	expiresAt := now.Add(l.svcCtx.RefreshTokenTTL)
	if err := l.svcCtx.AdminRepo.CreateRefreshToken(l.ctx, &model.RefreshToken{
		ID:        l.svcCtx.IDNode.Generate().Int64(),
		AdminID:   admin.ID,
		TokenID:   tokenID,
		TokenHash: tokenHash,
		IssuedAt:  now,
		ExpiresAt: expiresAt,
		IssuedIP:  strings.TrimSpace(in.ClientIp),
		UserAgent: strings.TrimSpace(in.UserAgent),
	}); err != nil {
		return nil, mapRepoErr(err)
	}
	writeAudit(l.ctx, l.svcCtx, admin.ID, "admin_login", "admin", admin.ID, "success", map[string]any{})
	return &pb.AdminAuthResp{
		Admin:               toAdminView(admin),
		AccessToken:         accessToken,
		AccessExpiresInSec:  accessTTL,
		RefreshToken:        refreshToken,
		RefreshExpiresInSec: int64(l.svcCtx.RefreshTokenTTL.Seconds()),
	}, nil
}

// AdminRefreshToken 处理管理员会话刷新。
func (l *AdminLogic) AdminRefreshToken(in *pb.AdminRefreshTokenReq) (*pb.AdminAuthResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	tokenID, _, err := splitRefreshToken(in.RefreshToken)
	if err != nil {
		return nil, err
	}
	record, err := l.svcCtx.AdminRepo.FindRefreshToken(l.ctx, tokenID)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	if record.RevokedAt != nil || time.Now().After(record.ExpiresAt) {
		return nil, errorx.New(errorx.CodeAdminRefreshTokenInvalid, "refresh token 无效")
	}
	if refreshHash(in.RefreshToken) != record.TokenHash {
		return nil, errorx.New(errorx.CodeAdminRefreshTokenInvalid, "refresh token 无效")
	}
	admin, err := l.svcCtx.AdminRepo.FindAdminByID(l.ctx, record.AdminID)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	if err := checkLocked(admin, time.Now()); err != nil {
		return nil, err
	}
	if err := setAdminRelations(l.ctx, l.svcCtx.AdminRepo, admin); err != nil {
		return nil, mapRepoErr(err)
	}
	now := time.Now()
	newTokenID, newRefreshToken, newTokenHash, err := buildRefreshToken()
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeSysInternal, "生成 refresh token 失败", err)
	}
	newExpiresAt := now.Add(l.svcCtx.RefreshTokenTTL)
	if err := l.svcCtx.AdminRepo.RotateRefreshToken(l.ctx, tokenID, now, &model.RefreshToken{
		ID:        l.svcCtx.IDNode.Generate().Int64(),
		AdminID:   admin.ID,
		TokenID:   newTokenID,
		TokenHash: newTokenHash,
		IssuedAt:  now,
		ExpiresAt: newExpiresAt,
		IssuedIP:  strings.TrimSpace(in.ClientIp),
		UserAgent: strings.TrimSpace(in.UserAgent),
	}); err != nil {
		return nil, mapRepoErr(err)
	}
	accessToken, accessTTL, err := issueAdminAccessToken(l.svcCtx, admin)
	if err != nil {
		return nil, err
	}
	writeAudit(l.ctx, l.svcCtx, admin.ID, "admin_refresh_token", "admin", admin.ID, "success", map[string]any{})
	return &pb.AdminAuthResp{
		Admin:               toAdminView(admin),
		AccessToken:         accessToken,
		AccessExpiresInSec:  accessTTL,
		RefreshToken:        newRefreshToken,
		RefreshExpiresInSec: int64(l.svcCtx.RefreshTokenTTL.Seconds()),
	}, nil
}

// AdminLogout 处理管理员退出登录。
func (l *AdminLogic) AdminLogout(in *pb.AdminLogoutReq) (*pb.AdminLogoutResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if err := mustPositiveID(in.AdminId, "admin_id"); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.RefreshToken) == "" {
		return nil, errorx.New(errorx.CodeSysBadRequest, "refresh_token 不能为空")
	}
	tokenID, _, err := splitRefreshToken(in.RefreshToken)
	if err != nil {
		return nil, err
	}
	record, err := l.svcCtx.AdminRepo.FindRefreshToken(l.ctx, tokenID)
	if err != nil && err != repository.ErrRefreshTokenNotFound {
		return nil, mapRepoErr(err)
	}
	if err == nil && record != nil && record.AdminID != in.AdminId {
		return nil, errorx.New(errorx.CodeAdminRefreshTokenInvalid, "refresh token 无效")
	}
	if err == nil {
		if revokeErr := l.svcCtx.AdminRepo.RevokeRefreshToken(l.ctx, tokenID, time.Now()); revokeErr != nil && revokeErr != repository.ErrRefreshTokenNotFound {
			return nil, mapRepoErr(revokeErr)
		}
	}
	writeAudit(l.ctx, l.svcCtx, in.AdminId, "admin_logout", "admin", in.AdminId, "success", map[string]any{})
	return &pb.AdminLogoutResp{Success: true}, nil
}

// GetMyAdminProfile 获取当前管理员资料。
func (l *AdminLogic) GetMyAdminProfile(in *pb.GetMyAdminProfileReq) (*pb.GetMyAdminProfileResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if err := mustPositiveID(in.AdminId, "admin_id"); err != nil {
		return nil, err
	}
	admin, err := l.svcCtx.AdminRepo.FindAdminByID(l.ctx, in.AdminId)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	if err := setAdminRelations(l.ctx, l.svcCtx.AdminRepo, admin); err != nil {
		return nil, mapRepoErr(err)
	}
	return &pb.GetMyAdminProfileResp{Admin: toAdminView(admin)}, nil
}

// ChangeMyPassword 修改当前管理员密码。
func (l *AdminLogic) ChangeMyPassword(in *pb.ChangeMyPasswordReq) (*pb.ChangeMyPasswordResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if err := mustPositiveID(in.AdminId, "admin_id"); err != nil {
		return nil, err
	}
	if err := validatePasswordStrength(in.NewPassword); err != nil {
		return nil, err
	}
	admin, err := l.svcCtx.AdminRepo.FindAdminByID(l.ctx, in.AdminId)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	if !comparePassword(admin.PasswordHash, in.OldPassword) {
		return nil, errorx.New(errorx.CodeAuthInvalidCredentials, "原密码错误")
	}
	if in.OldPassword == in.NewPassword {
		return nil, errorx.New(errorx.CodeSysBadRequest, "新密码不能与原密码相同")
	}
	hashed, err := hashPassword(in.NewPassword)
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeSysInternal, "密码加密失败", err)
	}
	now := time.Now()
	if err := l.svcCtx.AdminRepo.UpdateAdminPassword(l.ctx, in.AdminId, hashed, now); err != nil {
		return nil, mapRepoErr(err)
	}
	_ = l.svcCtx.AdminRepo.RevokeRefreshTokensByAdmin(l.ctx, in.AdminId, now)
	writeAudit(l.ctx, l.svcCtx, in.AdminId, "change_my_password", "admin", in.AdminId, "success", map[string]any{})
	return &pb.ChangeMyPasswordResp{Success: true}, nil
}
