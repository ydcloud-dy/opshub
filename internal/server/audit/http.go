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

package audit

import (
	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/service/audit"
)

type HTTPService struct {
	operationLogService *audit.OperationLogService
	loginLogService     *audit.LoginLogService
	dataLogService      *audit.DataLogService
}

func NewHTTPService(
	operationLogService *audit.OperationLogService,
	loginLogService *audit.LoginLogService,
	dataLogService *audit.DataLogService,
) *HTTPService {
	return &HTTPService{
		operationLogService: operationLogService,
		loginLogService:     loginLogService,
		dataLogService:      dataLogService,
	}
}

// RegisterRoutes 注册审计模块路由
func (s *HTTPService) RegisterRoutes(r *gin.RouterGroup) {
	// API v1 审计路由
	audit := r.Group("/audit")
	{
		// 操作日志路由
		operationLogs := audit.Group("/operation-logs")
		{
			operationLogs.GET("", s.operationLogService.ListOperationLogs)
			operationLogs.GET("/:id", s.operationLogService.GetOperationLog)
			operationLogs.DELETE("/:id", s.operationLogService.DeleteOperationLog)
			operationLogs.POST("/batch-delete", s.operationLogService.DeleteOperationLogsBatch)
		}

		// 登录日志路由
		loginLogs := audit.Group("/login-logs")
		{
			loginLogs.GET("", s.loginLogService.ListLoginLogs)
			loginLogs.GET("/:id", s.loginLogService.GetLoginLog)
			loginLogs.DELETE("/:id", s.loginLogService.DeleteLoginLog)
			loginLogs.POST("/batch-delete", s.loginLogService.DeleteLoginLogsBatch)
		}

		// 数据日志路由
		dataLogs := audit.Group("/data-logs")
		{
			dataLogs.GET("", s.dataLogService.ListDataLogs)
			dataLogs.GET("/:id", s.dataLogService.GetDataLog)
			dataLogs.DELETE("/:id", s.dataLogService.DeleteDataLog)
			dataLogs.POST("/batch-delete", s.dataLogService.DeleteDataLogsBatch)
		}
	}
}
