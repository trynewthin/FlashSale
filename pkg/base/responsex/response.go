// Package responsex 提供统一响应结构与构造函数。
package responsex

import "flashsale/pkg/base/errorx"

// Envelope 是统一接口响应体。
type Envelope struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	TraceID string `json:"trace_id,omitempty"`
}

// OK 构造成功响应。
func OK(data any) Envelope {
	return Envelope{
		Code:    "OK",
		Message: "success",
		Data:    data,
	}
}

// Fail 构造失败响应，并统一把 error 映射为 AppError。
func Fail(err error) Envelope {
	appErr := errorx.FromError(err)
	if appErr == nil {
		appErr = errorx.New(errorx.CodeSysInternal, "internal error")
	}
	return Envelope{
		Code:    string(appErr.Code),
		Message: appErr.Message,
	}
}
