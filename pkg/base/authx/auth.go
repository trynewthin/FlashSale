// Package authx 提供用户端与管理端双域 JWT 的签发与解析能力。
package authx

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenType 表示令牌所属域。
type TokenType string

const (
	// TokenTypeUser 表示用户端令牌。
	TokenTypeUser TokenType = "user"
	// TokenTypeAdmin 表示管理端令牌。
	TokenTypeAdmin TokenType = "admin"
)

// JWTDomainConfig 表示单个域（user/admin）的 JWT 配置。
type JWTDomainConfig struct {
	Secret   string
	Issuer   string
	Audience string
	TTL      time.Duration
}

// JWTConfig 表示双域 JWT 配置集合。
type JWTConfig struct {
	User  JWTDomainConfig
	Admin JWTDomainConfig
}

// Claims 是项目统一 JWT Claims。
type Claims struct {
	TokenType TokenType `json:"token_type"`
	Domains   []string  `json:"domains,omitempty"`
	DataScope string    `json:"data_scope,omitempty"`
	jwt.RegisteredClaims
}

var (
	configMu  sync.RWMutex
	globalCfg JWTConfig
	inited    bool
)

// Init 初始化全局 JWT 配置。
func Init(cfg JWTConfig) error {
	if cfg.User.Secret == "" || cfg.Admin.Secret == "" {
		return fmt.Errorf("jwt secrets are required")
	}
	if cfg.User.Issuer == "" || cfg.Admin.Issuer == "" {
		return fmt.Errorf("jwt issuer is required")
	}
	if cfg.User.Audience == "" || cfg.Admin.Audience == "" {
		return fmt.Errorf("jwt audience is required")
	}

	configMu.Lock()
	defer configMu.Unlock()
	globalCfg = cfg
	inited = true
	return nil
}

// Issue 根据令牌域签发 JWT。
func Issue(tt TokenType, subject string, ttl time.Duration) (string, error) {
	return issue(tt, subject, ttl, nil, "")
}

// IssueWithClaims 根据令牌域签发带扩展 claims 的 JWT。
func IssueWithClaims(tt TokenType, subject string, ttl time.Duration, domains []string, dataScope string) (string, error) {
	return issue(tt, subject, ttl, domains, dataScope)
}

func issue(tt TokenType, subject string, ttl time.Duration, domains []string, dataScope string) (string, error) {
	domain, err := domainConfig(tt)
	if err != nil {
		return "", err
	}
	if subject == "" {
		return "", fmt.Errorf("subject is required")
	}
	if ttl <= 0 {
		ttl = domain.TTL
	}
	if ttl <= 0 {
		ttl = time.Hour
	}

	now := time.Now()
	claims := Claims{
		TokenType: tt,
		Domains:   domains,
		DataScope: dataScope,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			Issuer:    domain.Issuer,
			Audience:  jwt.ClaimStrings{domain.Audience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(domain.Secret))
}

// Parse 解析并校验 JWT，保证域隔离与签名合法。
func Parse(tt TokenType, token string) (*Claims, error) {
	domain, err := domainConfig(tt)
	if err != nil {
		return nil, err
	}
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(domain.Secret), nil
	}, jwt.WithAudience(domain.Audience), jwt.WithIssuer(domain.Issuer))
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	if claims.TokenType != tt {
		return nil, fmt.Errorf("token type mismatch: expected %s got %s", tt, claims.TokenType)
	}
	return claims, nil
}

// domainConfig 返回指定域的 JWT 配置。
func domainConfig(tt TokenType) (JWTDomainConfig, error) {
	configMu.RLock()
	defer configMu.RUnlock()
	if !inited {
		return JWTDomainConfig{}, fmt.Errorf("auth not initialized")
	}
	switch tt {
	case TokenTypeUser:
		return globalCfg.User, nil
	case TokenTypeAdmin:
		return globalCfg.Admin, nil
	default:
		return JWTDomainConfig{}, fmt.Errorf("unsupported token type: %s", tt)
	}
}
