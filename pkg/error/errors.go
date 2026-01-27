// Copyright (c) 2026 DYCloud J.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package error

import (
	"fmt"
)

// ErrorCode 错误码类型
type ErrorCode int

const (
	// 成功
	Success ErrorCode = 0

	// 客户端错误 1000-1999
	ErrBadRequest        ErrorCode = 1000 // 请求参数错误
	ErrUnauthorized      ErrorCode = 1001 // 未授权
	ErrForbidden         ErrorCode = 1003 // 禁止访问
	ErrNotFound          ErrorCode = 1004 // 资源不存在
	ErrMethodNotAllowed  ErrorCode = 1005 // 方法不允许
	ErrRequestTimeout    ErrorCode = 1006 // 请求超时
	ErrConflict          ErrorCode = 1007 // 资源冲突

	// 业务错误 2000-2999
	ErrBusiness          ErrorCode = 2000 // 业务逻辑错误
	ErrValidation        ErrorCode = 2001 // 数据验证失败
	ErrDuplicate         ErrorCode = 2002 // 数据重复
	ErrInvalidOperation  ErrorCode = 2003 // 无效操作

	// 服务器错误 5000-5999
	ErrInternalServer    ErrorCode = 5000 // 服务器内部错误
	ErrDatabase          ErrorCode = 5001 // 数据库错误
	ErrCache             ErrorCode = 5002 // 缓存错误
	ErrExternalAPI       ErrorCode = 5003 // 外部接口错误
)

// AppError 应用错误
type AppError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"`
	Err     error     `json:"-"`
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 实现 errors.Unwrap
func (e *AppError) Unwrap() error {
	return e.Err
}

// New 创建新的应用错误
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Wrap 包装错误
func Wrap(err error, code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// WithDetails 添加错误详情
func (e *AppError) WithDetails(details string) *AppError {
	e.Details = details
	return e
}

// 预定义常用错误
var (
	ErrBadRequestError       = New(ErrBadRequest, "请求参数错误")
	ErrUnauthorizedError     = New(ErrUnauthorized, "未授权访问")
	ErrForbiddenError        = New(ErrForbidden, "禁止访问")
	ErrNotFoundError         = New(ErrNotFound, "资源不存在")
	ErrMethodNotAllowedError = New(ErrMethodNotAllowed, "请求方法不允许")
	ErrConflictError         = New(ErrConflict, "资源冲突")
	ErrInternalServerError   = New(ErrInternalServer, "服务器内部错误")
	ErrDatabaseError         = New(ErrDatabase, "数据库错误")
	ErrValidationError       = New(ErrValidation, "数据验证失败")
)
