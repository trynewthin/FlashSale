// Package errorx 定义服务统一错误模型与错误码。
package errorx

import (
	"errors"
	"fmt"
)

// Code 表示业务或系统错误码。
type Code string

const (
	// CodeSysInternal 表示系统内部错误。
	CodeSysInternal Code = "SYS_INTERNAL"
	// CodeSysBadRequest 表示请求参数非法。
	CodeSysBadRequest Code = "SYS_BAD_REQUEST"
	// CodeAuthUnauthorized 表示未认证。
	CodeAuthUnauthorized Code = "AUTH_UNAUTHORIZED"
	// CodeAuthForbidden 表示无权限。
	CodeAuthForbidden Code = "AUTH_FORBIDDEN"
	// CodeDBError 表示数据库错误。
	CodeDBError Code = "DB_ERROR"
	// CodeMQError 表示消息队列错误。
	CodeMQError Code = "MQ_ERROR"
	// CodeCacheError 表示缓存错误。
	CodeCacheError Code = "CACHE_ERROR"
)

// AppError 是统一错误结构，包含错误码、可读信息和底层原因。
type AppError struct {
	Code    Code
	Message string
	Cause   error
}

// Error 实现 error 接口，返回可用于日志的完整错误字符串。
func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause == nil {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
}

// Unwrap 返回底层错误，支持 errors.Is/errors.As 链式判断。
func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// New 创建不带底层错误的业务错误。
func New(code Code, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

// Wrap 创建带底层错误的业务错误。
func Wrap(code Code, message string, cause error) *AppError {
	return &AppError{Code: code, Message: message, Cause: cause}
}

// FromError 将任意 error 转换为 *AppError，保证上层统一处理。
func FromError(err error) *AppError {
	if err == nil {
		return nil
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return Wrap(CodeSysInternal, "internal error", err)
}
