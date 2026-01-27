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

package rbac

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	appError "github.com/ydcloud-dy/opshub/pkg/error"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

// ErrorResponse 错误响应辅助函数
func ErrorResponse(c *gin.Context, status int, message string) {
	var code appError.ErrorCode
	switch status {
	case http.StatusBadRequest:
		code = appError.ErrBadRequest
	case http.StatusUnauthorized:
		code = appError.ErrUnauthorized
	case http.StatusForbidden:
		code = appError.ErrForbidden
	case http.StatusNotFound:
		code = appError.ErrNotFound
	case http.StatusInternalServerError:
		code = appError.ErrInternalServer
	default:
		code = appError.ErrInternalServer
	}

	c.JSON(status, response.Response{
		Code:      int(code),
		Message:   message,
		Timestamp: 0,
	})
}

// AbortWithError 中止并返回错误
func AbortWithError(c *gin.Context, status int, message string) {
	ErrorResponse(c, status, message)
	c.Abort()
}

// Errorf 格式化错误
func Errorf(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}
