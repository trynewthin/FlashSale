// model 包包含相关应用代码。
package model

import "time"

// User 表示用户库中的用户实体。
type User struct {
	ID           int64
	Phone        string
	PasswordHash string
	Nickname     string
	Status       int8
	LastLoginAt  *time.Time
	LastLoginIP  string
	DeletedAt    *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
