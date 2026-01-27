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

package service

import (
	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/biz"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

// Service 服务层
type Service struct {
	biz *biz.Biz
}

// NewService 创建服务层
func NewService(biz *biz.Biz) *Service {
	return &Service{
		biz: biz,
	}
}

// Example 示例接口
// @Summary 示例接口
// @Description 这是一个示例接口
// @Tags 示例
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=map[string]string}
// @Router /api/v1/example [get]
func (s *Service) Example(c *gin.Context) {
	// 调用业务层
	if err := s.biz.Example(c.Request.Context()); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "hello world",
	})
}

// Health 健康检查
// @Summary 健康检查
// @Description 检查服务健康状态
// @Tags 系统
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=map[string]string}
// @Router /health [get]
func (s *Service) Health(c *gin.Context) {
	response.Success(c, gin.H{
		"status": "ok",
	})
}
