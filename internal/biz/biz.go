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

package biz

import (
	"context"

	"gorm.io/gorm"

	"github.com/ydcloud-dy/opshub/internal/data"
)

// UseCase 业务用例接口
type UseCase interface {
	// 在这里定义业务方法
}

// Biz 业务层
type Biz struct {
	db  *gorm.DB
	rdb *data.Redis
}

// NewBiz 创建业务层
func NewBiz(data *data.Data, redis *data.Redis) *Biz {
	return &Biz{
		db:  data.DB(),
		rdb: redis,
	}
}

// Example 示例方法
func (b *Biz) Example(ctx context.Context) error {
	// 业务逻辑实现
	return nil
}
