// perf_tempdata_db 提供直接数据库插入创建测试用户的能力，跳过 HTTP API 注册流程。
// 相比串行 HTTP 注册，批量 INSERT 可在数秒内创建万级用户。
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"flashsale/pkg/base/authx"

	"golang.org/x/crypto/bcrypt"

	_ "github.com/go-sql-driver/mysql"
)

const (
	// perfDBBatchSize 每批 INSERT 的行数。
	perfDBBatchSize = 500
)

// createTempUsersDirectDB 通过批量 MySQL INSERT + 本地签 JWT 创建测试用户。
// 比串行 HTTP 注册快 100 倍以上；phone 带 perf_test_ 前缀，与正常用户隔离。
func createTempUsersDirectDB(repoRoot string, userCount int, uniqueSuffix string) ([]int64, string, error) {
	if userCount <= 0 {
		return nil, "", nil
	}

	// 1. 初始化 authx（从环境变量或 dev 默认值）
	if err := ensureAuthxInit(); err != nil {
		return nil, "", fmt.Errorf("init authx: %w", err)
	}

	// 2. 生成固定密码的 bcrypt 哈希（只算一次）
	password := strings.TrimSpace(os.Getenv("FLASHSALE_PERF_USER_PASSWORD"))
	if password == "" {
		password = "abc12345"
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("bcrypt hash: %w", err)
	}
	passwordHash := string(hash)

	// 3. 连接用户库
	db, closeFn, err := openDB("flash_user")
	if err != nil {
		return nil, "", fmt.Errorf("open flash_user db: %w", err)
	}
	defer closeFn()

	// 4. 生成 ID 和 phone 列表
	// phone 使用 199 前缀 + suffix 尾 3 位 + 5 位序号 = 11 位数字，符合 phone 列长度限制。
	now := time.Now()
	baseID := now.UnixNano()/1000 + 9_000_000_000_000_000 // 确保不与正常 snowflake ID 冲突
	shortSuffix := strings.TrimSpace(uniqueSuffix)
	if len(shortSuffix) > 3 {
		shortSuffix = shortSuffix[len(shortSuffix)-3:]
	}
	userIDs := make([]int64, userCount)
	phones := make([]string, userCount)
	for i := 0; i < userCount; i++ {
		userIDs[i] = baseID + int64(i)
		phones[i] = fmt.Sprintf("199%s%05d", shortSuffix, i)
	}

	// 5. 批量 INSERT
	fmt.Printf("[perf.temp.db] inserting %d test users ...\n", userCount)
	inserted := 0
	for start := 0; start < userCount; start += perfDBBatchSize {
		end := start + perfDBBatchSize
		if end > userCount {
			end = userCount
		}
		batch := end - start

		// 构造 VALUES 子句
		placeholders := make([]string, batch)
		args := make([]any, 0, batch*5)
		for i := 0; i < batch; i++ {
			idx := start + i
			nickname := fmt.Sprintf("perf_test_%d", idx+1)
			placeholders[i] = "(?, ?, ?, ?, 1, ?, ?)"
			args = append(args, userIDs[idx], phones[idx], passwordHash, nickname, now, now)
		}
		query := "INSERT INTO users (id, phone, password_hash, nickname, status, created_at, updated_at) VALUES " +
			strings.Join(placeholders, ",")

		if _, err := db.Exec(query, args...); err != nil {
			return nil, "", fmt.Errorf("batch insert at offset %d: %w", start, err)
		}
		inserted += batch
		if inserted%2000 == 0 || inserted == userCount {
			fmt.Printf("[perf.temp.db] inserted %d/%d users\n", inserted, userCount)
		}
	}

	// 6. 本地签发 JWT（24h 有效期）
	fmt.Printf("[perf.temp.db] issuing %d JWT tokens ...\n", userCount)
	tokens := make([]string, userCount)
	for i := 0; i < userCount; i++ {
		token, err := authx.Issue(authx.TokenTypeUser, strconv.FormatInt(userIDs[i], 10), 24*time.Hour)
		if err != nil {
			return nil, "", fmt.Errorf("issue jwt for user %d: %w", userIDs[i], err)
		}
		tokens[i] = token
	}

	// 7. 写入 token 文件
	tokenFile := filepath.Join(repoRoot, "log", "data", fmt.Sprintf("perf-temp-%s.tokens.txt", strings.TrimSpace(uniqueSuffix)))
	if err := os.MkdirAll(filepath.Dir(tokenFile), 0o755); err != nil {
		return nil, "", err
	}
	content := strings.Join(tokens, "\n") + "\n"
	if err := os.WriteFile(tokenFile, []byte(content), 0o600); err != nil {
		return nil, "", err
	}

	fmt.Printf("[perf.temp.db] done: %d users created, token file: %s\n", userCount, tokenFile)
	return userIDs, tokenFile, nil
}

// cleanupTempUsersDirectDB 通过 SQL DELETE 批量清理测试用户（按 ID 范围）。
func cleanupTempUsersDirectDB(userIDs []int64) error {
	if len(userIDs) == 0 {
		return nil
	}
	db, closeFn, err := openDB("flash_user")
	if err != nil {
		return fmt.Errorf("open flash_user db: %w", err)
	}
	defer closeFn()

	minID, maxID := userIDs[0], userIDs[0]
	for _, id := range userIDs {
		if id < minID {
			minID = id
		}
		if id > maxID {
			maxID = id
		}
	}
	result, err := db.Exec("DELETE FROM users WHERE id >= ? AND id <= ?", minID, maxID)
	if err != nil {
		return fmt.Errorf("delete test users: %w", err)
	}
	affected, _ := result.RowsAffected()
	fmt.Printf("[perf.temp.db] cleaned %d test users (id range %d~%d)\n", affected, minID, maxID)
	return nil
}

// ensureAuthxInit 初始化 authx JWT 签发配置（仅需调用一次）。
func ensureAuthxInit() error {
	secret := strings.TrimSpace(os.Getenv("FLASHSALE_JWT_USER_SECRET"))
	if secret == "" {
		secret = "user-secret-change-me" // dev.yaml 默认值
	}
	adminSecret := strings.TrimSpace(os.Getenv("FLASHSALE_JWT_ADMIN_SECRET"))
	if adminSecret == "" {
		adminSecret = "admin-secret-change-me"
	}
	return authx.Init(authx.JWTConfig{
		User: authx.JWTDomainConfig{
			Secret:   secret,
			Issuer:   "flashsale-user",
			Audience: "flashsale-user-client",
			TTL:      24 * time.Hour,
		},
		Admin: authx.JWTDomainConfig{
			Secret:   adminSecret,
			Issuer:   "flashsale-admin",
			Audience: "flashsale-admin-client",
			TTL:      12 * time.Hour,
		},
	})
}
