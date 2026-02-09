// grpcerr 包包含相关应用代码。
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
	case errorx.CodeSysBadRequest, errorx.CodeAuthInvalidPhone, errorx.CodeAuthWeakPassword, errorx.CodeProductInvalidStatus, errorx.CodeOrderInvalidState, errorx.CodeOrderOutOfStock, errorx.CodeOrderInvalidReceiverInfo, errorx.CodeSeckillInvalidConfig, errorx.CodeAdminInvalidScope:
		return codes.InvalidArgument
	case errorx.CodeAuthUnauthorized, errorx.CodeAuthInvalidCredentials, errorx.CodeAdminRefreshTokenInvalid:
		return codes.Unauthenticated
	case errorx.CodeAuthForbidden, errorx.CodeAdminAccountDisabled, errorx.CodeAdminAccountLocked:
		return codes.PermissionDenied
	case errorx.CodeUserNotFound, errorx.CodeProductNotFound, errorx.CodeOrderNotFound, errorx.CodeSeckillActivityNotFound, errorx.CodeSeckillItemNotFound, errorx.CodeAdminNotFound, errorx.CodeAdminRoleNotFound:
		return codes.NotFound
	case errorx.CodeAuthPhoneAlreadyRegistered, errorx.CodeProductSKUAlreadyExists, errorx.CodeOrderAlreadyClosed, errorx.CodeOrderAlreadyPaid, errorx.CodeOrderPayTimeout, errorx.CodeOrderReviewTimeout, errorx.CodeOrderReviewRejected, errorx.CodeSeckillActivityNotPublished, errorx.CodeSeckillActivityNotStarted, errorx.CodeSeckillActivityEnded, errorx.CodeSeckillOutOfStock, errorx.CodeSeckillLimitExceeded, errorx.CodeSeckillPurchaseConflict, errorx.CodeAdminUsernameAlreadyExists, errorx.CodeAdminRoleCodeAlreadyExists:
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
		// Fallback only: 具体资源错误码应优先从 ErrorInfo.Reason 解析。
		return errorx.CodeUserNotFound
	case codes.AlreadyExists:
		// Fallback only: 具体业务冲突码应优先从 ErrorInfo.Reason 解析。
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
		errorx.CodeOrderNotFound,
		errorx.CodeOrderInvalidState,
		errorx.CodeOrderOutOfStock,
		errorx.CodeOrderPayTimeout,
		errorx.CodeOrderReviewTimeout,
		errorx.CodeOrderReviewRejected,
		errorx.CodeOrderAlreadyClosed,
		errorx.CodeOrderAlreadyPaid,
		errorx.CodeOrderInvalidReceiverInfo,
		errorx.CodeSeckillActivityNotFound,
		errorx.CodeSeckillActivityNotPublished,
		errorx.CodeSeckillActivityNotStarted,
		errorx.CodeSeckillActivityEnded,
		errorx.CodeSeckillItemNotFound,
		errorx.CodeSeckillOutOfStock,
		errorx.CodeSeckillLimitExceeded,
		errorx.CodeSeckillPurchaseConflict,
		errorx.CodeSeckillInvalidConfig,
		errorx.CodeAdminNotFound,
		errorx.CodeAdminUsernameAlreadyExists,
		errorx.CodeAdminAccountDisabled,
		errorx.CodeAdminAccountLocked,
		errorx.CodeAdminRoleNotFound,
		errorx.CodeAdminRoleCodeAlreadyExists,
		errorx.CodeAdminInvalidScope,
		errorx.CodeAdminRefreshTokenInvalid,
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
