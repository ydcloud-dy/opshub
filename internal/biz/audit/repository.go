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
	"context"
)

// OperationLogRepo 操作日志仓储接口
type OperationLogRepo interface {
	Create(ctx context.Context, log *SysOperationLog) error
	GetByID(ctx context.Context, id uint) (*SysOperationLog, error)
	List(ctx context.Context, page, pageSize int, username, module, action, status, startTime, endTime string) ([]*SysOperationLog, int64, error)
	Delete(ctx context.Context, id uint) error
	DeleteBatch(ctx context.Context, ids []uint) error
}

// LoginLogRepo 登录日志仓储接口
type LoginLogRepo interface {
	Create(ctx context.Context, log *SysLoginLog) error
	GetByID(ctx context.Context, id uint) (*SysLoginLog, error)
	List(ctx context.Context, page, pageSize int, username, loginType, loginStatus, startTime, endTime string) ([]*SysLoginLog, int64, error)
	UpdateLogout(ctx context.Context, userID uint, logoutTime *SysLoginLog) error
	Delete(ctx context.Context, id uint) error
	DeleteBatch(ctx context.Context, ids []uint) error
}

// DataLogRepo 数据日志仓储接口
type DataLogRepo interface {
	Create(ctx context.Context, log *SysDataLog) error
	GetByID(ctx context.Context, id uint) (*SysDataLog, error)
	List(ctx context.Context, page, pageSize int, username, tableName, action, startTime, endTime string) ([]*SysDataLog, int64, error)
	Delete(ctx context.Context, id uint) error
	DeleteBatch(ctx context.Context, ids []uint) error
}
