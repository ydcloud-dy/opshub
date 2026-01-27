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

package response

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	appError "github.com/ydcloud-dy/opshub/pkg/error"
)

// Response 统一响应结构
type Response struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	Timestamp int64  `json:"timestamp"`
	TraceID   string `json:"trace_id,omitempty"`
}

// PaginationResponse 分页响应
type PaginationResponse struct {
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Data     any   `json:"data"`
}

// Success 成功响应
func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{
		Code:      int(appError.Success),
		Message:   "success",
		Data:      data,
		Timestamp: time.Now().Unix(),
	})
}

// SuccessWithMessage 成功响应(带消息)
func SuccessWithMessage(c *gin.Context, message string, data any) {
	c.JSON(http.StatusOK, Response{
		Code:      int(appError.Success),
		Message:   message,
		Data:      data,
		Timestamp: time.Now().Unix(),
	})
}

// Error 错误响应
func Error(c *gin.Context, err error) {
	var appErr *appError.AppError

	// 类型断言
	if ae, ok := err.(*appError.AppError); ok {
		appErr = ae
	} else {
		// 未知错误包装为内部服务器错误
		appErr = appError.Wrap(err, appError.ErrInternalServer, "服务器内部错误")
	}

	// 根据错误码确定 HTTP 状态码
	httpStatus := getHTTPStatus(appErr.Code)

	c.JSON(httpStatus, Response{
		Code:      int(appErr.Code),
		Message:   appErr.Message,
		Timestamp: time.Now().Unix(),
	})
}

// ErrorCode 错误响应(带状态码和消息) - 简化版本
func ErrorCode(c *gin.Context, httpStatus int, message string) {
	// 对于业务错误，使用5000作为错误码，HTTP状态码为200
	businessCode := 5000
	if httpStatus >= 400 && httpStatus < 600 {
		businessCode = httpStatus
	}

	c.JSON(http.StatusOK, Response{
		Code:      businessCode,
		Message:   message,
		Timestamp: time.Now().Unix(),
	})
}

// ErrorWithData 错误响应(带数据)
func ErrorWithData(c *gin.Context, err error, data any) {
	var appErr *appError.AppError

	if ae, ok := err.(*appError.AppError); ok {
		appErr = ae
	} else {
		appErr = appError.Wrap(err, appError.ErrInternalServer, "服务器内部错误")
	}

	httpStatus := getHTTPStatus(appErr.Code)

	c.JSON(httpStatus, Response{
		Code:      int(appErr.Code),
		Message:   appErr.Message,
		Data:      data,
		Timestamp: time.Now().Unix(),
	})
}

// Pagination 分页响应
func Pagination(c *gin.Context, total int64, page, pageSize int, data any) {
	c.JSON(http.StatusOK, Response{
		Code:      int(appError.Success),
		Message:   "success",
		Data: PaginationResponse{
			Total:    total,
			Page:     page,
			PageSize: pageSize,
			Data:     data,
		},
		Timestamp: time.Now().Unix(),
	})
}

// getHTTPStatus 根据业务错误码获取 HTTP 状态码
func getHTTPStatus(code appError.ErrorCode) int {
	switch code {
	case appError.Success:
		return http.StatusOK
	case appError.ErrBadRequest, appError.ErrValidation, appError.ErrDuplicate, appError.ErrInvalidOperation:
		return http.StatusBadRequest
	case appError.ErrUnauthorized:
		return http.StatusUnauthorized
	case appError.ErrForbidden:
		return http.StatusForbidden
	case appError.ErrNotFound:
		return http.StatusNotFound
	case appError.ErrConflict:
		return http.StatusConflict
	case appError.ErrMethodNotAllowed:
		return http.StatusMethodNotAllowed
	case appError.ErrRequestTimeout:
		return http.StatusRequestTimeout
	case appError.ErrInternalServer, appError.ErrDatabase, appError.ErrCache, appError.ErrExternalAPI:
		return http.StatusInternalServerError
	default:
		// 业务错误默认返回 200
		if code >= 2000 && code < 3000 {
			return http.StatusOK
		}
		return http.StatusInternalServerError
	}
}
