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

package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	rbacBiz "github.com/ydcloud-dy/opshub/internal/biz/rbac"
	rbacData "github.com/ydcloud-dy/opshub/internal/data/rbac"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetCurrentUserID 从 gin.Context 获取当前登录用户的 ID
// 返回 userID 和 是否成功，如果失败会自动向客户端返回错误响应
func GetCurrentUserID(c *gin.Context) (uint, bool) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权：无法获取用户信息",
		})
		return 0, false
	}

	currentUserID, ok := userIDVal.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "用户ID类型错误",
		})
		return 0, false
	}

	return currentUserID, true
}

// RequireAdmin 检查当前用户是否为管理员
// 返回 是否为管理员，如果不是管理员会自动向客户端返回错误响应
func RequireAdmin(c *gin.Context, db *gorm.DB) bool {
	userID, ok := GetCurrentUserID(c)
	if !ok {
		return false
	}

	// 创建 RoleUseCase 来查询用户角色
	roleRepo := rbacData.NewRoleRepo(db)
	roleUseCase := rbacBiz.NewRoleUseCase(roleRepo)

	roles, err := roleUseCase.GetByUserID(context.Background(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取用户角色失败: " + err.Error(),
		})
		return false
	}

	// 检查是否有admin角色
	for _, role := range roles {
		if role.Code == "admin" {
			return true
		}
	}

	// 不是管理员，返回权限不足
	c.JSON(http.StatusForbidden, gin.H{
		"code":    403,
		"message": "权限不足：此操作仅限管理员执行",
	})
	return false
}

// HandleK8sError 处理 K8s API 错误，返回友好的错误提示
func HandleK8sError(c *gin.Context, err error, resourceName string) {
	if err == nil {
		return
	}

	errorMsg := err.Error()

	// 权限不足错误 (403 Forbidden)
	if strings.Contains(errorMsg, "forbidden") {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "权限不足：您没有访问" + resourceName + "的权限，请联系管理员在「集群授权」中为您分配相应角色",
		})
		return
	}

	// 资源不存在错误 (404 Not Found)
	if strings.Contains(errorMsg, "not found") {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": resourceName + "不存在",
		})
		return
	}

	// 未授权错误 (401 Unauthorized)
	if strings.Contains(errorMsg, "Unauthorized") {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "认证失败：凭据无效或已过期，请重新申请集群访问凭据",
		})
		return
	}

	// 其他错误
	c.JSON(http.StatusInternalServerError, gin.H{
		"code":    500,
		"message": "操作失败: " + errorMsg,
	})
}

// calculateAge 计算资源年龄
func calculateAge(creationTime time.Time) string {
	duration := time.Since(creationTime)

	days := int(duration.Hours() / 24)
	hours := int(duration.Hours())
	minutes := int(duration.Minutes())

	if days > 0 {
		if days == 1 {
			return "1d"
		}
		return strconv.Itoa(days) + "d"
	}

	if hours > 0 {
		if hours == 1 {
			return "1h"
		}
		return strconv.Itoa(hours) + "h"
	}

	if minutes > 0 {
		if minutes == 1 {
			return "1m"
		}
		return strconv.Itoa(minutes) + "m"
	}

	return "<1m"
}

// RecordingWebSocketReader 带录制功能的 WebSocket 读取器
type RecordingWebSocketReader struct {
	conn      *websocket.Conn
	data      chan []byte
	recorder  *AsciinemaRecorder
	startTime time.Time
}

// Read 实现 io.Reader 接口
func (r *RecordingWebSocketReader) Read(p []byte) (int, error) {
	data, ok := <-r.data
	if !ok {
		return 0, fmt.Errorf("channel closed")
	}

	// 记录用户输入
	if r.recorder != nil {
		_ = r.recorder.RecordInput(data)
	}

	n := copy(p, data)
	return n, nil
}

// RecordingWebSocketWriter 带录制功能的 WebSocket 写入器
type RecordingWebSocketWriter struct {
	conn      *websocket.Conn
	recorder  *AsciinemaRecorder
	startTime time.Time
}

// Write 实现 io.Writer 接口
func (w *RecordingWebSocketWriter) Write(p []byte) (int, error) {
	// 记录终端输出
	if w.recorder != nil {
		_ = w.recorder.RecordOutput(p)
	}

	// 写入 WebSocket
	if err := w.conn.WriteMessage(websocket.TextMessage, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

// ==================== YAML 清理函数 ====================

// cleanMetadataForYAML 清理 metadata 字段用于 YAML 输出
func cleanMetadataForYAML(meta metav1.ObjectMeta) map[string]interface{} {
	metadata := make(map[string]interface{})

	if meta.Name != "" {
		metadata["name"] = meta.Name
	}
	if meta.Namespace != "" {
		metadata["namespace"] = meta.Namespace
	}
	if len(meta.Labels) > 0 {
		metadata["labels"] = meta.Labels
	}
	if len(meta.Annotations) > 0 {
		metadata["annotations"] = meta.Annotations
	}
	// 不包含 managedFields、resourceVersion、uid、generation 等字段

	return metadata
}

// cleanPersistentVolumeClaimForYAML 清理 PVC 对象用于 YAML 输出
func cleanPersistentVolumeClaimForYAML(pvc *v1.PersistentVolumeClaim) map[string]interface{} {
	result := make(map[string]interface{})

	// 设置 apiVersion 和 kind
	result["apiVersion"] = "v1"
	result["kind"] = "PersistentVolumeClaim"

	// 添加 metadata
	result["metadata"] = cleanMetadataForYAML(pvc.ObjectMeta)

	// 添加 spec
	result["spec"] = pvc.Spec

	// 不包含 status

	return result
}

// cleanPersistentVolumeForYAML 清理 PV 对象用于 YAML 输出
func cleanPersistentVolumeForYAML(pv *v1.PersistentVolume) map[string]interface{} {
	result := make(map[string]interface{})

	// 设置 apiVersion 和 kind
	result["apiVersion"] = "v1"
	result["kind"] = "PersistentVolume"

	// 添加 metadata
	result["metadata"] = cleanMetadataForYAML(pv.ObjectMeta)

	// 添加 spec
	result["spec"] = pv.Spec

	// 不包含 status

	return result
}

// cleanStorageClassForYAML 清理 StorageClass 对象用于 YAML 输出
func cleanStorageClassForYAML(sc *storagev1.StorageClass) map[string]interface{} {
	result := make(map[string]interface{})

	// 设置 apiVersion 和 kind
	result["apiVersion"] = "storage.k8s.io/v1"
	result["kind"] = "StorageClass"

	// 添加 metadata
	result["metadata"] = cleanMetadataForYAML(sc.ObjectMeta)

	// 添加 spec
	spec := make(map[string]interface{})
	if sc.Provisioner != "" {
		spec["provisioner"] = sc.Provisioner
	}
	if sc.Parameters != nil {
		spec["parameters"] = sc.Parameters
	}
	if sc.ReclaimPolicy != nil {
		spec["reclaimPolicy"] = *sc.ReclaimPolicy
	}
	if sc.MountOptions != nil {
		spec["mountOptions"] = sc.MountOptions
	}
	if sc.VolumeBindingMode != nil {
		spec["volumeBindingMode"] = *sc.VolumeBindingMode
	}
	if sc.AllowedTopologies != nil {
		spec["allowedTopologies"] = sc.AllowedTopologies
	}
	result["spec"] = spec

	// 不包含 status

	return result
}

// cleanEndpointsForYAML 清理 Endpoints 对象用于 YAML 输出
func cleanEndpointsForYAML(ep *v1.Endpoints) map[string]interface{} {
	result := make(map[string]interface{})

	// 设置 apiVersion 和 kind
	result["apiVersion"] = "v1"
	result["kind"] = "Endpoints"

	// 添加 metadata
	result["metadata"] = cleanMetadataForYAML(ep.ObjectMeta)

	// 添加 spec (Endpoints 没有 spec，直接返回)
	// 添加 subsets
	result["subsets"] = ep.Subsets

	// 不包含 status

	return result
}

// cleanConfigMapForYAML 清理 ConfigMap 对象用于 YAML 输出
func cleanConfigMapForYAML(cm *v1.ConfigMap) map[string]interface{} {
	result := make(map[string]interface{})

	// 设置 apiVersion 和 kind
	result["apiVersion"] = "v1"
	result["kind"] = "ConfigMap"

	// 添加 metadata
	result["metadata"] = cleanMetadataForYAML(cm.ObjectMeta)

	// 添加 data
	if len(cm.Data) > 0 {
		result["data"] = cm.Data
	}

	// 添加 binaryData
	if len(cm.BinaryData) > 0 {
		result["binaryData"] = cm.BinaryData
	}

	// ConfigMap 没有 spec 和 status

	return result
}

// cleanSecretForYAML 清理 Secret 对象用于 YAML 输出
func cleanSecretForYAML(secret *v1.Secret) map[string]interface{} {
	result := make(map[string]interface{})

	// 设置 apiVersion 和 kind
	result["apiVersion"] = "v1"
	result["kind"] = "Secret"

	// 添加 metadata
	result["metadata"] = cleanMetadataForYAML(secret.ObjectMeta)

	// 添加 data
	if len(secret.Data) > 0 {
		result["data"] = secret.Data
	}

	// 添加 type
	result["type"] = string(secret.Type)

	// Secret 没有 spec，不包含 status

	return result
}
