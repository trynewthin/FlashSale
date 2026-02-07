// Package grpcerr 提供 AppError 与 gRPC status 的双向转换。
package grpcerr

import (
	"errors"

	"flashsale/pkg/base/errorx"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ToStatus 把业务错误转换为可跨 gRPC 边界传递的 status error。
func ToStatus(err error) error {
	if err == nil {
		return nil
	}
	appErr := errorx.FromError(err)
	if appErr == nil {
		return nil
	}
	st := status.New(codeToGRPC(appErr.Code), appErr.Message)
	withDetails, detailErr := st.WithDetails(&errdetails.ErrorInfo{
		Reason: string(appErr.Code),
	})
	if detailErr == nil {
		st = withDetails
	}
	return st.Err()
}

// FromStatus 把 gRPC status error 还原为业务错误。
func FromStatus(err error) *errorx.AppError {
	if err == nil {
		return nil
	}
	var appErr *errorx.AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	st, ok := status.FromError(err)
	if !ok {
		return errorx.FromError(err)
	}

	code := grpcToCode(st.Code())
	for _, detail := range st.Details() {
		info, ok := detail.(*errdetails.ErrorInfo)
		if !ok {
			continue
		}
		if parsed, ok := parseCode(info.Reason); ok {
			code = parsed
		}
		break
	}
	return errorx.New(code, st.Message())
}

func codeToGRPC(code errorx.Code) codes.Code {
	switch code {
	case errorx.CodeSysBadRequest, errorx.CodeAuthInvalidPhone, errorx.CodeAuthWeakPassword, errorx.CodeProductInvalidStatus:
		return codes.InvalidArgument
	case errorx.CodeAuthUnauthorized, errorx.CodeAuthInvalidCredentials:
		return codes.Unauthenticated
	case errorx.CodeAuthForbidden:
		return codes.PermissionDenied
	case errorx.CodeUserNotFound, errorx.CodeProductNotFound:
		return codes.NotFound
	case errorx.CodeAuthPhoneAlreadyRegistered, errorx.CodeProductSKUAlreadyExists:
		return codes.AlreadyExists
	default:
		return codes.Internal
	}
}

func grpcToCode(code codes.Code) errorx.Code {
	switch code {
	case codes.InvalidArgument:
		return errorx.CodeSysBadRequest
	case codes.Unauthenticated:
		return errorx.CodeAuthUnauthorized
	case codes.PermissionDenied:
		return errorx.CodeAuthForbidden
	case codes.NotFound:
		return errorx.CodeUserNotFound
	case codes.AlreadyExists:
		return errorx.CodeAuthPhoneAlreadyRegistered
	default:
		return errorx.CodeSysInternal
	}
}

func parseCode(raw string) (errorx.Code, bool) {
	code := errorx.Code(raw)
	switch code {
	case errorx.CodeSysInternal,
		errorx.CodeSysBadRequest,
		errorx.CodeUserNotFound,
		errorx.CodeProductNotFound,
		errorx.CodeProductSKUAlreadyExists,
		errorx.CodeProductInvalidStatus,
		errorx.CodeAuthUnauthorized,
		errorx.CodeAuthForbidden,
		errorx.CodeAuthInvalidPhone,
		errorx.CodeAuthWeakPassword,
		errorx.CodeAuthPhoneAlreadyRegistered,
		errorx.CodeAuthInvalidCredentials,
		errorx.CodeDBError,
		errorx.CodeMQError,
		errorx.CodeCacheError:
		return code, true
	default:
		return "", false
	}
}
