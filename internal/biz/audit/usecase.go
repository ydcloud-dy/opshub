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

// OperationLogUseCase 操作日志用例
type OperationLogUseCase struct {
	repo OperationLogRepo
}

func NewOperationLogUseCase(repo OperationLogRepo) *OperationLogUseCase {
	return &OperationLogUseCase{
		repo: repo,
	}
}

func (uc *OperationLogUseCase) Create(ctx context.Context, log *SysOperationLog) error {
	return uc.repo.Create(ctx, log)
}

func (uc *OperationLogUseCase) GetByID(ctx context.Context, id uint) (*SysOperationLog, error) {
	return uc.repo.GetByID(ctx, id)
}

func (uc *OperationLogUseCase) List(ctx context.Context, page, pageSize int, username, module, action, status, startTime, endTime string) ([]*SysOperationLog, int64, error) {
	return uc.repo.List(ctx, page, pageSize, username, module, action, status, startTime, endTime)
}

func (uc *OperationLogUseCase) Delete(ctx context.Context, id uint) error {
	return uc.repo.Delete(ctx, id)
}

func (uc *OperationLogUseCase) DeleteBatch(ctx context.Context, ids []uint) error {
	return uc.repo.DeleteBatch(ctx, ids)
}

// LoginLogUseCase 登录日志用例
type LoginLogUseCase struct {
	repo LoginLogRepo
}

func NewLoginLogUseCase(repo LoginLogRepo) *LoginLogUseCase {
	return &LoginLogUseCase{
		repo: repo,
	}
}

func (uc *LoginLogUseCase) Create(ctx context.Context, log *SysLoginLog) error {
	return uc.repo.Create(ctx, log)
}

func (uc *LoginLogUseCase) GetByID(ctx context.Context, id uint) (*SysLoginLog, error) {
	return uc.repo.GetByID(ctx, id)
}

func (uc *LoginLogUseCase) List(ctx context.Context, page, pageSize int, username, loginType, loginStatus, startTime, endTime string) ([]*SysLoginLog, int64, error) {
	return uc.repo.List(ctx, page, pageSize, username, loginType, loginStatus, startTime, endTime)
}

func (uc *LoginLogUseCase) UpdateLogout(ctx context.Context, userID uint, logoutTime *SysLoginLog) error {
	return uc.repo.UpdateLogout(ctx, userID, logoutTime)
}

func (uc *LoginLogUseCase) Delete(ctx context.Context, id uint) error {
	return uc.repo.Delete(ctx, id)
}

func (uc *LoginLogUseCase) DeleteBatch(ctx context.Context, ids []uint) error {
	return uc.repo.DeleteBatch(ctx, ids)
}

// DataLogUseCase 数据日志用例
type DataLogUseCase struct {
	repo DataLogRepo
}

func NewDataLogUseCase(repo DataLogRepo) *DataLogUseCase {
	return &DataLogUseCase{
		repo: repo,
	}
}

func (uc *DataLogUseCase) Create(ctx context.Context, log *SysDataLog) error {
	return uc.repo.Create(ctx, log)
}

func (uc *DataLogUseCase) GetByID(ctx context.Context, id uint) (*SysDataLog, error) {
	return uc.repo.GetByID(ctx, id)
}

func (uc *DataLogUseCase) List(ctx context.Context, page, pageSize int, username, tableName, action, startTime, endTime string) ([]*SysDataLog, int64, error) {
	return uc.repo.List(ctx, page, pageSize, username, tableName, action, startTime, endTime)
}

func (uc *DataLogUseCase) Delete(ctx context.Context, id uint) error {
	return uc.repo.Delete(ctx, id)
}

func (uc *DataLogUseCase) DeleteBatch(ctx context.Context, ids []uint) error {
	return uc.repo.DeleteBatch(ctx, ids)
}
