package perf

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	dataaction "flashsale/ops/backend/actions/data"
	"flashsale/pkg/base/authx"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

const perfDBBatchSize = 500

func createTempUsersDirectDB(repoRoot string, userCount int, uniqueSuffix string) ([]int64, string, error) {
	if userCount <= 0 {
		return nil, "", nil
	}
	if err := ensureAuthxInit(); err != nil {
		return nil, "", fmt.Errorf("init authx: %w", err)
	}

	password := strings.TrimSpace(os.Getenv("FLASHSALE_PERF_USER_PASSWORD"))
	if password == "" {
		password = "abc12345"
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("bcrypt hash: %w", err)
	}
	passwordHash := string(hash)

	db, closeFn, err := dataaction.OpenDB("flash_user")
	if err != nil {
		return nil, "", fmt.Errorf("open flash_user db: %w", err)
	}
	defer closeFn()

	now := time.Now()
	baseID := now.UnixNano()/1000 + 9_000_000_000_000_000
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

	fmt.Printf("[perf.temp.db] inserting %d test users ...\n", userCount)
	inserted := 0
	for start := 0; start < userCount; start += perfDBBatchSize {
		end := start + perfDBBatchSize
		if end > userCount {
			end = userCount
		}
		batch := end - start

		placeholders := make([]string, batch)
		args := make([]any, 0, batch*6)
		for i := 0; i < batch; i++ {
			idx := start + i
			nickname := fmt.Sprintf("perf_test_%d", idx+1)
			placeholders[i] = "(?, ?, ?, ?, 1, ?, ?)"
			args = append(args, userIDs[idx], phones[idx], passwordHash, nickname, now, now)
		}
		query := "INSERT INTO users (id, phone, password_hash, nickname, status, created_at, updated_at) VALUES " + strings.Join(placeholders, ",")
		if _, err := db.Exec(query, args...); err != nil {
			return nil, "", fmt.Errorf("batch insert at offset %d: %w", start, err)
		}
		inserted += batch
		if inserted%2000 == 0 || inserted == userCount {
			fmt.Printf("[perf.temp.db] inserted %d/%d users\n", inserted, userCount)
		}
	}

	fmt.Printf("[perf.temp.db] issuing %d JWT tokens ...\n", userCount)
	tokens := make([]string, userCount)
	for i := 0; i < userCount; i++ {
		token, err := authx.Issue(authx.TokenTypeUser, strconv.FormatInt(userIDs[i], 10), 24*time.Hour)
		if err != nil {
			return nil, "", fmt.Errorf("issue jwt for user %d: %w", userIDs[i], err)
		}
		tokens[i] = token
	}

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

func cleanupTempUsersDirectDB(userIDs []int64) error {
	if len(userIDs) == 0 {
		return nil
	}
	db, closeFn, err := dataaction.OpenDB("flash_user")
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

func ensureAuthxInit() error {
	secret := strings.TrimSpace(os.Getenv("FLASHSALE_JWT_USER_SECRET"))
	if secret == "" {
		secret = "user-secret-change-me"
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
