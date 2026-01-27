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
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	"github.com/ydcloud-dy/opshub/plugins/kubernetes/model"
	"github.com/ydcloud-dy/opshub/plugins/kubernetes/service"
)

// InspectionHandler 巡检处理器
type InspectionHandler struct {
	inspectionService *service.InspectionService
	db                *gorm.DB
}

// NewInspectionHandler 创建巡检处理器
func NewInspectionHandler(clusterService *service.ClusterService, db *gorm.DB) *InspectionHandler {
	return &InspectionHandler{
		inspectionService: service.NewInspectionService(db, clusterService),
		db:                db,
	}
}

// StartInspectionRequest 开始巡检请求
type StartInspectionRequest struct {
	ClusterIDs []uint64                `json:"clusterIds" binding:"required"`
	Options    model.InspectionOptions `json:"options"`
}

// StartInspection 开始巡检
// @Summary 开始集群巡检
// @Tags Kubernetes巡检
// @Accept json
// @Produce json
// @Param request body StartInspectionRequest true "巡检请求"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/plugins/kubernetes/inspection/start [post]
func (h *InspectionHandler) StartInspection(c *gin.Context) {
	var req StartInspectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	// 获取用户ID
	userID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	inspectionReq := &model.StartInspectionRequest{
		ClusterIDs: req.ClusterIDs,
		Options:    req.Options,
		UserID:     uint64(userID),
	}

	inspection, err := h.inspectionService.StartInspection(c.Request.Context(), inspectionReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "启动巡检失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"inspectionId": inspection.ID,
		},
	})
}

// GetInspectionProgress 获取巡检进度
// @Summary 获取巡检进度
// @Tags Kubernetes巡检
// @Accept json
// @Produce json
// @Param inspectionId path int true "巡检ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/plugins/kubernetes/inspection/progress/{inspectionId} [get]
func (h *InspectionHandler) GetInspectionProgress(c *gin.Context) {
	inspectionID, err := strconv.ParseUint(c.Param("inspectionId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的巡检ID",
		})
		return
	}

	progress, err := h.inspectionService.GetInspectionProgress(c.Request.Context(), inspectionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取进度失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    progress,
	})
}

// GetInspectionResult 获取巡检结果
// @Summary 获取巡检结果
// @Tags Kubernetes巡检
// @Accept json
// @Produce json
// @Param inspectionId path int true "巡检ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/plugins/kubernetes/inspection/result/{inspectionId} [get]
func (h *InspectionHandler) GetInspectionResult(c *gin.Context) {
	inspectionID, err := strconv.ParseUint(c.Param("inspectionId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的巡检ID",
		})
		return
	}

	inspection, result, err := h.inspectionService.GetInspectionResult(c.Request.Context(), inspectionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取结果失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"inspection": inspection,
			"result":     result,
		},
	})
}

// GetInspectionHistory 获取巡检历史
// @Summary 获取巡检历史
// @Tags Kubernetes巡检
// @Accept json
// @Produce json
// @Param clusterId query int false "集群ID"
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/plugins/kubernetes/inspection/history [get]
func (h *InspectionHandler) GetInspectionHistory(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Query("clusterId"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	items, total, err := h.inspectionService.GetInspectionHistory(c.Request.Context(), clusterID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取历史失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"total":    total,
			"list":     items,
			"page":     page,
			"pageSize": pageSize,
		},
	})
}

// DeleteInspection 删除巡检记录
// @Summary 删除巡检记录
// @Tags Kubernetes巡检
// @Accept json
// @Produce json
// @Param inspectionId path int true "巡检ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/plugins/kubernetes/inspection/{inspectionId} [delete]
func (h *InspectionHandler) DeleteInspection(c *gin.Context) {
	inspectionID, err := strconv.ParseUint(c.Param("inspectionId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的巡检ID",
		})
		return
	}

	if err := h.inspectionService.DeleteInspection(c.Request.Context(), inspectionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "删除失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}

// ExportInspection 导出巡检报告
// @Summary 导出巡检报告
// @Tags Kubernetes巡检
// @Accept json
// @Produce application/octet-stream
// @Param inspectionId path int true "巡检ID"
// @Param format query string false "导出格式(excel)"
// @Success 200 {file} file
// @Router /api/v1/plugins/kubernetes/inspection/export/{inspectionId} [get]
func (h *InspectionHandler) ExportInspection(c *gin.Context) {
	inspectionID, err := strconv.ParseUint(c.Param("inspectionId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的巡检ID",
		})
		return
	}

	format := c.DefaultQuery("format", "excel")

	inspection, result, err := h.inspectionService.GetInspectionResult(c.Request.Context(), inspectionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取结果失败: " + err.Error(),
		})
		return
	}

	if result == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "巡检结果不存在",
		})
		return
	}

	switch format {
	case "excel":
		h.exportExcel(c, inspection, result)
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "不支持的导出格式",
		})
	}
}

// exportExcel 导出Excel报告
func (h *InspectionHandler) exportExcel(c *gin.Context, inspection *model.ClusterInspection, result *model.InspectionResult) {
	f := excelize.NewFile()
	defer f.Close()

	// 设置样式
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Size:  16,
			Color: "#FFFFFF",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#000000"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Size:  11,
			Color: "#FFFFFF",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#333333"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
		},
	})

	successStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Color: "#52c41a",
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
		},
	})

	warningStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Color: "#faad14",
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
		},
	})

	errorStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Color: "#ff4d4f",
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
		},
	})

	// ==================== 概览 Sheet ====================
	f.SetSheetName("Sheet1", "巡检概览")
	sheet := "巡检概览"

	// 标题
	f.MergeCell(sheet, "A1", "E1")
	f.SetCellValue(sheet, "A1", "集群巡检报告")
	f.SetCellStyle(sheet, "A1", "E1", titleStyle)
	f.SetRowHeight(sheet, 1, 35)

	// 基本信息
	f.SetCellValue(sheet, "A3", "集群名称")
	f.SetCellValue(sheet, "B3", inspection.ClusterName)
	f.SetCellValue(sheet, "A4", "健康评分")
	f.SetCellValue(sheet, "B4", fmt.Sprintf("%d/100", inspection.Score))
	f.SetCellValue(sheet, "A5", "巡检时间")
	f.SetCellValue(sheet, "B5", inspection.CreatedAt.Format("2006-01-02 15:04:05"))
	f.SetCellValue(sheet, "A6", "巡检耗时")
	f.SetCellValue(sheet, "B6", fmt.Sprintf("%d秒", inspection.Duration))

	// 统计信息
	f.SetCellValue(sheet, "D3", "检查项总数")
	f.SetCellValue(sheet, "E3", inspection.CheckCount)
	f.SetCellValue(sheet, "D4", "通过项数")
	f.SetCellValue(sheet, "E4", inspection.PassCount)
	f.SetCellValue(sheet, "D5", "警告项数")
	f.SetCellValue(sheet, "E5", inspection.WarningCount)
	f.SetCellValue(sheet, "D6", "失败项数")
	f.SetCellValue(sheet, "E6", inspection.FailCount)

	// 检查项详情表格
	f.SetCellValue(sheet, "A8", "类别")
	f.SetCellValue(sheet, "B8", "检查项")
	f.SetCellValue(sheet, "C8", "状态")
	f.SetCellValue(sheet, "D8", "检查值")
	f.SetCellValue(sheet, "E8", "详情")
	f.SetCellValue(sheet, "F8", "建议")
	f.SetCellStyle(sheet, "A8", "F8", headerStyle)

	// 收集所有检查项
	allItems := []model.CheckItem{}
	allItems = append(allItems, result.ClusterInfo.Items...)
	allItems = append(allItems, result.NodeHealth.Items...)
	allItems = append(allItems, result.Components.Items...)
	allItems = append(allItems, result.Workloads.Items...)
	allItems = append(allItems, result.Network.Items...)
	allItems = append(allItems, result.Storage.Items...)
	allItems = append(allItems, result.Security.Items...)
	allItems = append(allItems, result.Config.Items...)
	allItems = append(allItems, result.Capacity.Items...)
	allItems = append(allItems, result.Events.Items...)

	row := 9
	for _, item := range allItems {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), item.Category)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), item.Name)

		statusCell := fmt.Sprintf("C%d", row)
		switch item.Status {
		case model.CheckStatusSuccess:
			f.SetCellValue(sheet, statusCell, "正常")
			f.SetCellStyle(sheet, statusCell, statusCell, successStyle)
		case model.CheckStatusWarning:
			f.SetCellValue(sheet, statusCell, "警告")
			f.SetCellStyle(sheet, statusCell, statusCell, warningStyle)
		case model.CheckStatusError:
			f.SetCellValue(sheet, statusCell, "异常")
			f.SetCellStyle(sheet, statusCell, statusCell, errorStyle)
		}

		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), item.Value)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), item.Detail)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), item.Suggestion)
		row++
	}

	// 设置列宽
	f.SetColWidth(sheet, "A", "A", 15)
	f.SetColWidth(sheet, "B", "B", 20)
	f.SetColWidth(sheet, "C", "C", 10)
	f.SetColWidth(sheet, "D", "D", 15)
	f.SetColWidth(sheet, "E", "E", 40)
	f.SetColWidth(sheet, "F", "F", 40)

	// ==================== 节点健康 Sheet ====================
	f.NewSheet("节点健康")
	sheet = "节点健康"

	f.SetCellValue(sheet, "A1", "节点名称")
	f.SetCellValue(sheet, "B1", "状态")
	f.SetCellValue(sheet, "C1", "CPU容量")
	f.SetCellValue(sheet, "D1", "内存容量")
	f.SetCellValue(sheet, "E1", "Pod数量")
	f.SetCellValue(sheet, "F1", "Pod容量")
	f.SetCellStyle(sheet, "A1", "F1", headerStyle)

	row = 2
	for _, node := range result.NodeHealth.NodeUtilization {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), node.Name)
		statusCell := fmt.Sprintf("B%d", row)
		if node.Status == "Ready" {
			f.SetCellValue(sheet, statusCell, "Ready")
			f.SetCellStyle(sheet, statusCell, statusCell, successStyle)
		} else {
			f.SetCellValue(sheet, statusCell, "NotReady")
			f.SetCellStyle(sheet, statusCell, statusCell, errorStyle)
		}
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), node.CPUCapacity)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), node.MemoryCapacity)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), node.PodCount)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), node.PodCapacity)
		row++
	}

	f.SetColWidth(sheet, "A", "F", 15)

	// ==================== 工作负载 Sheet ====================
	if len(result.Workloads.UnhealthyWorkloads) > 0 {
		f.NewSheet("异常工作负载")
		sheet = "异常工作负载"

		f.SetCellValue(sheet, "A1", "类型")
		f.SetCellValue(sheet, "B1", "命名空间")
		f.SetCellValue(sheet, "C1", "名称")
		f.SetCellValue(sheet, "D1", "就绪状态")
		f.SetCellValue(sheet, "E1", "原因")
		f.SetCellStyle(sheet, "A1", "E1", headerStyle)

		row = 2
		for _, wl := range result.Workloads.UnhealthyWorkloads {
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), wl.Kind)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), wl.Namespace)
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), wl.Name)
			f.SetCellValue(sheet, fmt.Sprintf("D%d", row), wl.Ready)
			f.SetCellValue(sheet, fmt.Sprintf("E%d", row), wl.Reason)
			row++
		}

		f.SetColWidth(sheet, "A", "E", 20)
	}

	// ==================== 事件 Sheet ====================
	if len(result.Events.RecentEvents) > 0 {
		f.NewSheet("最近事件")
		sheet = "最近事件"

		f.SetCellValue(sheet, "A1", "类型")
		f.SetCellValue(sheet, "B1", "原因")
		f.SetCellValue(sheet, "C1", "对象")
		f.SetCellValue(sheet, "D1", "命名空间")
		f.SetCellValue(sheet, "E1", "次数")
		f.SetCellValue(sheet, "F1", "最后发生")
		f.SetCellValue(sheet, "G1", "消息")
		f.SetCellStyle(sheet, "A1", "G1", headerStyle)

		row = 2
		for _, event := range result.Events.RecentEvents {
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), event.Type)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), event.Reason)
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), event.Object)
			f.SetCellValue(sheet, fmt.Sprintf("D%d", row), event.Namespace)
			f.SetCellValue(sheet, fmt.Sprintf("E%d", row), event.Count)
			f.SetCellValue(sheet, fmt.Sprintf("F%d", row), event.LastSeen)
			f.SetCellValue(sheet, fmt.Sprintf("G%d", row), event.Message)
			row++
		}

		f.SetColWidth(sheet, "A", "A", 10)
		f.SetColWidth(sheet, "B", "B", 15)
		f.SetColWidth(sheet, "C", "C", 30)
		f.SetColWidth(sheet, "D", "D", 15)
		f.SetColWidth(sheet, "E", "E", 8)
		f.SetColWidth(sheet, "F", "F", 20)
		f.SetColWidth(sheet, "G", "G", 50)
	}

	// 生成文件
	var buffer bytes.Buffer
	if err := f.Write(&buffer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "生成Excel失败: " + err.Error(),
		})
		return
	}

	filename := fmt.Sprintf("cluster-inspection-%s-%s.xlsx", inspection.ClusterName, time.Now().Format("20060102150405"))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buffer.Bytes())
}
