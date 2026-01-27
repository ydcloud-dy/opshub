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

package repository

import (
	"context"
	"encoding/json"

	"github.com/ydcloud-dy/opshub/plugins/task/model"
	"gorm.io/gorm"
)

type JobTaskRepository struct {
	db *gorm.DB
}

func NewJobTaskRepository(db *gorm.DB) *JobTaskRepository {
	return &JobTaskRepository{db: db}
}

func (r *JobTaskRepository) Create(ctx context.Context, jobTask *model.JobTask) error {
	return r.db.WithContext(ctx).Create(jobTask).Error
}

func (r *JobTaskRepository) GetByID(ctx context.Context, id uint) (*model.JobTask, error) {
	var jobTask model.JobTask
	err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&jobTask).Error
	if err != nil {
		return nil, err
	}
	return &jobTask, nil
}

func (r *JobTaskRepository) Update(ctx context.Context, jobTask *model.JobTask) error {
	return r.db.WithContext(ctx).Save(jobTask).Error
}

func (r *JobTaskRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.JobTask{}).Error
}

func (r *JobTaskRepository) List(ctx context.Context, page, pageSize int, keyword, taskType, status string) ([]*model.JobTask, int64, error) {
	var jobTasks []*model.JobTask
	var total int64

	query := r.db.WithContext(ctx).Model(&model.JobTask{}).Where("deleted_at IS NULL")

	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}

	if taskType != "" {
		query = query.Where("task_type = ?", taskType)
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = query.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&jobTasks).Error

	return jobTasks, total, err
}

func (r *JobTaskRepository) UpdateStatus(ctx context.Context, id uint, status string, result map[string]interface{}, errorMessage string) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if result != nil {
		resultJSON, _ := json.Marshal(result)
		updates["result"] = string(resultJSON)
	}

	if errorMessage != "" {
		updates["error_message"] = errorMessage
	}

	return r.db.WithContext(ctx).Model(&model.JobTask{}).Where("id = ?", id).Updates(updates).Error
}
