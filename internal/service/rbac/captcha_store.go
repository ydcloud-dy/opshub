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
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"go.uber.org/zap"
)

// RedisCaptchaStore Redis 验证码存储
type RedisCaptchaStore struct {
	client     *redis.Client
	expiration time.Duration
	keyPrefix  string
}

// NewRedisCaptchaStore 创建 Redis 验证码存储
func NewRedisCaptchaStore(client *redis.Client, expiration time.Duration) *RedisCaptchaStore {
	return &RedisCaptchaStore{
		client:     client,
		expiration: expiration,
		keyPrefix:  "captcha:",
	}
}

// Set 存储验证码
func (s *RedisCaptchaStore) Set(id string, value string) error {
	ctx := context.Background()
	key := s.keyPrefix + id
	err := s.client.Set(ctx, key, value, s.expiration).Err()
	appLogger.Info("Redis验证码存储", zap.String("key", key), zap.String("value", value), zap.Error(err))
	return err
}

// Get 获取验证码（不删除）
func (s *RedisCaptchaStore) Get(id string, clear bool) string {
	ctx := context.Background()
	key := s.keyPrefix + id
	val, err := s.client.Get(ctx, key).Result()
	appLogger.Info("Redis验证码获取", zap.String("key", key), zap.String("value", val), zap.Bool("clear", clear), zap.Error(err))
	if err != nil {
		return ""
	}
	if clear {
		s.client.Del(ctx, key)
	}
	return val
}

// Verify 验证验证码
func (s *RedisCaptchaStore) Verify(id, answer string, clear bool) bool {
	val := s.Get(id, clear)
	return val != "" && val == answer
}
