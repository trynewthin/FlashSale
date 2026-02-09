// logic 包包含相关应用代码。
package logic

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"flashsale/apps/admin/rpc/internal/model"
	"flashsale/apps/admin/rpc/internal/repository"
	"flashsale/apps/admin/rpc/internal/svc"
	"flashsale/apps/admin/rpc/pb"
	baseauth "flashsale/pkg/base/authx"
	"flashsale/pkg/base/errorx"
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/bcrypt"
)

var (
	usernamePattern = regexp.MustCompile(`^[a-z0-9_]{4,32}$`)
	passwordHasLetter = regexp.MustCompile(`[A-Za-z]`)
	passwordHasDigit = regexp.MustCompile(`\d`)
)

var validDomains = map[string]struct{}{
	"operations": {},
	"user_management": {},
	"product_management": {},
	"order_management": {},
	"seckill_management": {},
	"admin_management": {},
}

// AdminLogic 封装管理员逻辑公共依赖。
type AdminLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewAdminLogic 创建 AdminLogic。
func NewAdminLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminLogic {
	return &AdminLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func normalizePagination(page, pageSize int64) (int64, int64) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func normalizeUsername(username string) (string, error) {
	username = strings.TrimSpace(strings.ToLower(username))
	if !usernamePattern.MatchString(username) {
		return "", errorx.New(errorx.CodeSysBadRequest, "用户名格式不合法")
	}
	return username, nil
}

func normalizeDisplayName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if l := utf8.RuneCountInString(name); l < 1 || l > 64 {
		return "", errorx.New(errorx.CodeSysBadRequest, "显示名长度需在1到64之间")
	}
	return name, nil
}

func validatePasswordStrength(password string) error {
	if len(password) < 8 || len(password) > 32 {
		return errorx.New(errorx.CodeAuthWeakPassword, "密码强度不足")
	}
	if !passwordHasLetter.MatchString(password) || !passwordHasDigit.MatchString(password) {
		return errorx.New(errorx.CodeAuthWeakPassword, "密码强度不足")
	}
	return nil
}

func normalizeDataScope(scope string) (string, error) {
	scope = strings.TrimSpace(strings.ToLower(scope))
	switch scope {
	case model.DataScopeAll, model.DataScopeSelf:
		return scope, nil
	default:
		return "", errorx.New(errorx.CodeAdminInvalidScope, "data_scope 非法")
	}
}

func normalizeStatus(status int32) (int8, error) {
	switch int8(status) {
	case model.AdminStatusEnabled, model.AdminStatusDisabled:
		return int8(status), nil
	default:
		return 0, errorx.New(errorx.CodeSysBadRequest, "status 非法")
	}
}

func normalizeRoleStatus(status int32) (int8, error) {
	switch int8(status) {
	case model.RoleStatusEnabled, model.RoleStatusDisabled:
		return int8(status), nil
	default:
		return 0, errorx.New(errorx.CodeSysBadRequest, "status 非法")
	}
}

func normalizeRoleCode(code string) (string, error) {
	code = strings.TrimSpace(strings.ToLower(code))
	if !usernamePattern.MatchString(code) {
		return "", errorx.New(errorx.CodeSysBadRequest, "role_code 格式不合法")
	}
	return code, nil
}

func normalizeDomains(domains []string) ([]string, error) {
	set := make(map[string]struct{})
	for _, domain := range domains {
		domain = strings.TrimSpace(domain)
		if domain == "" {
			continue
		}
		if _, ok := validDomains[domain]; !ok {
			return nil, errorx.New(errorx.CodeSysBadRequest, "domains 包含非法值")
		}
		set[domain] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for domain := range set {
		out = append(out, domain)
	}
	sort.Strings(out)
	return out, nil
}

func mapRepoErr(err error) error {
	switch err {
	case nil:
		return nil
	case repository.ErrAdminNotFound:
		return errorx.New(errorx.CodeAdminNotFound, "管理员不存在")
	case repository.ErrRoleNotFound:
		return errorx.New(errorx.CodeAdminRoleNotFound, "角色不存在")
	case repository.ErrDuplicateUsername:
		return errorx.New(errorx.CodeAdminUsernameAlreadyExists, "管理员用户名已存在")
	case repository.ErrDuplicateRoleCode:
		return errorx.New(errorx.CodeAdminRoleCodeAlreadyExists, "角色编码已存在")
	case repository.ErrRefreshTokenNotFound:
		return errorx.New(errorx.CodeAdminRefreshTokenInvalid, "refresh token 无效")
	default:
		return errorx.Wrap(errorx.CodeDBError, "数据库操作失败", err)
	}
}

func toAdminView(admin *model.Admin) *pb.AdminView {
	if admin == nil {
		return nil
	}
	view := &pb.AdminView{
		AdminId:      admin.ID,
		Username:     admin.Username,
		DisplayName:  admin.DisplayName,
		Status:       int32(admin.Status),
		DataScope:    admin.DataScope,
		IsSuperAdmin: admin.IsSuperAdmin,
		LastLoginIp:  admin.LastLoginIP,
		RoleIds:      append([]int64{}, admin.RoleIDs...),
		Domains:      append([]string{}, admin.Domains...),
		CreatedAtUnix: admin.CreatedAt.Unix(),
		UpdatedAtUnix: admin.UpdatedAt.Unix(),
	}
	if admin.LastLoginAt != nil {
		view.LastLoginAtUnix = admin.LastLoginAt.Unix()
	}
	return view
}

func toRoleView(role *model.Role) *pb.RoleView {
	if role == nil {
		return nil
	}
	return &pb.RoleView{
		RoleId:        role.ID,
		RoleCode:      role.RoleCode,
		RoleName:      role.RoleName,
		Status:        int32(role.Status),
		IsSystem:      role.IsSystem,
		Domains:       append([]string{}, role.Domains...),
		CreatedAtUnix: role.CreatedAt.Unix(),
		UpdatedAtUnix: role.UpdatedAt.Unix(),
	}
}

func toAuditLogView(logItem *model.AuditLog) *pb.AdminAuditLogView {
	if logItem == nil {
		return nil
	}
	return &pb.AdminAuditLogView{
		LogId:        logItem.ID,
		AdminId:      logItem.AdminID,
		Action:       logItem.Action,
		TargetType:   logItem.TargetType,
		TargetId:     logItem.TargetID,
		Result:       logItem.Result,
		RequestId:    logItem.RequestID,
		Ip:           logItem.IP,
		UserAgent:    logItem.UserAgent,
		DetailJson:   logItem.DetailJSON,
		CreatedAtUnix: logItem.CreatedAt.Unix(),
	}
}

func issueAdminAccessToken(svcCtx *svc.ServiceContext, admin *model.Admin) (string, int64, error) {
	if svcCtx == nil || admin == nil {
		return "", 0, errorx.New(errorx.CodeSysInternal, "service context invalid")
	}
	token, err := baseauth.IssueWithClaims(baseauth.TokenTypeAdmin, strconv.FormatInt(admin.ID, 10), svcCtx.AccessTokenTTL, admin.Domains, admin.DataScope)
	if err != nil {
		return "", 0, errorx.Wrap(errorx.CodeAuthUnauthorized, "签发令牌失败", err)
	}
	expiresIn := int64(svcCtx.AccessTokenTTL.Seconds())
	if expiresIn <= 0 {
		expiresIn = int64((15 * time.Minute).Seconds())
	}
	return token, expiresIn, nil
}

func buildRefreshToken() (tokenID string, rawToken string, tokenHash string, err error) {
	randomBytes := make([]byte, 32)
	if _, err = rand.Read(randomBytes); err != nil {
		return "", "", "", err
	}
	tokenID = hex.EncodeToString(randomBytes[:8])
	secret := base64.RawURLEncoding.EncodeToString(randomBytes)
	rawToken = tokenID + "." + secret
	sum := sha256.Sum256([]byte(rawToken))
	tokenHash = hex.EncodeToString(sum[:])
	return tokenID, rawToken, tokenHash, nil
}

func splitRefreshToken(raw string) (string, string, error) {
	raw = strings.TrimSpace(raw)
	parts := strings.Split(raw, ".")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", errorx.New(errorx.CodeAdminRefreshTokenInvalid, "refresh token 非法")
	}
	return parts[0], parts[1], nil
}

func refreshHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func checkLocked(admin *model.Admin, now time.Time) error {
	if admin == nil {
		return errorx.New(errorx.CodeAdminNotFound, "管理员不存在")
	}
	if admin.Status != model.AdminStatusEnabled {
		return errorx.New(errorx.CodeAdminAccountDisabled, "管理员账号已禁用")
	}
	if admin.LockedUntil != nil && admin.LockedUntil.After(now) {
		return errorx.New(errorx.CodeAdminAccountLocked, "管理员账号已锁定")
	}
	return nil
}

func setAdminRelations(ctx context.Context, repo repository.AdminRepository, admin *model.Admin) error {
	if admin == nil || repo == nil {
		return nil
	}
	roleIDs, err := repo.ListAdminRoleIDs(ctx, admin.ID)
	if err != nil {
		return err
	}
	domains, err := repo.ListAdminDomains(ctx, admin.ID)
	if err != nil {
		return err
	}
	admin.RoleIDs = roleIDs
	if admin.IsSuperAdmin {
		domains = make([]string, 0, len(validDomains))
		for domain := range validDomains {
			domains = append(domains, domain)
		}
		sort.Strings(domains)
	}
	admin.Domains = domains
	return nil
}

func writeAudit(ctx context.Context, svcCtx *svc.ServiceContext, adminID int64, action, targetType string, targetID int64, result string, detail map[string]any) {
	if svcCtx == nil || svcCtx.AdminRepo == nil || svcCtx.IDNode == nil {
		return
	}
	detailJSON := "{}"
	if len(detail) > 0 {
		if buf, err := json.Marshal(detail); err == nil {
			detailJSON = string(buf)
		}
	}
	_ = svcCtx.AdminRepo.CreateAuditLog(ctx, &model.AuditLog{
		ID:         svcCtx.IDNode.Generate().Int64(),
		AdminID:    adminID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Result:     result,
		DetailJSON: detailJSON,
		CreatedAt:  time.Now(),
	})
}

func hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func comparePassword(hash, raw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(raw)) == nil
}

func mustPositiveID(id int64, field string) error {
	if id <= 0 {
		return errorx.New(errorx.CodeSysBadRequest, fmt.Sprintf("%s 非法", field))
	}
	return nil
}
