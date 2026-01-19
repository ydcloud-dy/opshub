package asset

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"github.com/ydcloud-dy/opshub/internal/biz/asset"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

type HostService struct {
	hostUseCase       *asset.HostUseCase
	credentialUseCase *asset.CredentialUseCase
	cloudUseCase      *asset.CloudAccountUseCase
}

func NewHostService(hostUseCase *asset.HostUseCase, credentialUseCase *asset.CredentialUseCase, cloudUseCase *asset.CloudAccountUseCase) *HostService {
	return &HostService{
		hostUseCase:       hostUseCase,
		credentialUseCase: credentialUseCase,
		cloudUseCase:      cloudUseCase,
	}
}

// CreateHost 创建主机
func (s *HostService) CreateHost(c *gin.Context) {
	var req asset.HostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	host, err := s.hostUseCase.Create(c.Request.Context(), &req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}

	response.Success(c, host)
}

// UpdateHost 更新主机
func (s *HostService) UpdateHost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的主机ID")
		return
	}

	var req asset.HostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	req.ID = uint(id)
	if err := s.hostUseCase.Update(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "更新成功", nil)
}

// DeleteHost 删除主机
func (s *HostService) DeleteHost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的主机ID")
		return
	}

	if err := s.hostUseCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetHost 获取主机详情
func (s *HostService) GetHost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的主机ID")
		return
	}

	host, err := s.hostUseCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "主机不存在")
		return
	}

	response.Success(c, host)
}

// ListHosts 主机列表
func (s *HostService) ListHosts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	keyword := c.Query("keyword")

	// 支持分组ID筛选
	var groupID *uint
	if groupIDStr := c.Query("groupId"); groupIDStr != "" {
		id, err := strconv.ParseUint(groupIDStr, 10, 32)
		if err == nil {
			gid := uint(id)
			groupID = &gid
		}
	}

	hosts, total, err := s.hostUseCase.List(c.Request.Context(), page, pageSize, keyword, groupID)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":  hosts,
		"total": total,
		"page":  page,
		"pageSize": pageSize,
	})
}

// CreateCredential 创建凭证
func (s *HostService) CreateCredential(c *gin.Context) {
	var req asset.CredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	credential, err := s.credentialUseCase.Create(c.Request.Context(), &req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}

	response.Success(c, credential)
}

// UpdateCredential 更新凭证
func (s *HostService) UpdateCredential(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的凭证ID")
		return
	}

	var req asset.CredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	req.ID = uint(id)
	if err := s.credentialUseCase.Update(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "更新成功", nil)
}

// DeleteCredential 删除凭证
func (s *HostService) DeleteCredential(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的凭证ID")
		return
	}

	if err := s.credentialUseCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetCredential 获取凭证详情
func (s *HostService) GetCredential(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的凭证ID")
		return
	}

	// 获取解密后的凭证详情（用于编辑时回显私钥）
	credential, err := s.credentialUseCase.GetByIDDecrypted(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "凭证不存在")
		return
	}

	response.Success(c, credential)
}

// ListCredentials 凭证列表
func (s *HostService) ListCredentials(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	keyword := c.Query("keyword")

	credentials, total, err := s.credentialUseCase.List(c.Request.Context(), page, pageSize, keyword)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":     credentials,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetAllCredentials 获取所有凭证（用于下拉选择）
func (s *HostService) GetAllCredentials(c *gin.Context) {
	credentials, err := s.credentialUseCase.GetAll(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, credentials)
}

// CreateCloudAccount 创建云平台账号
func (s *HostService) CreateCloudAccount(c *gin.Context) {
	var req asset.CloudAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	account, err := s.cloudUseCase.Create(c.Request.Context(), &req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}

	response.Success(c, account)
}

// UpdateCloudAccount 更新云平台账号
func (s *HostService) UpdateCloudAccount(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的账号ID")
		return
	}

	var req asset.CloudAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	req.ID = uint(id)
	if err := s.cloudUseCase.Update(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "更新成功", nil)
}

// DeleteCloudAccount 删除云平台账号
func (s *HostService) DeleteCloudAccount(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的账号ID")
		return
	}

	if err := s.cloudUseCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetCloudAccount 获取云平台账号详情
func (s *HostService) GetCloudAccount(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的账号ID")
		return
	}

	account, err := s.cloudUseCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "账号不存在")
		return
	}

	response.Success(c, account)
}

// ListCloudAccounts 云平台账号列表
func (s *HostService) ListCloudAccounts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	accounts, total, err := s.cloudUseCase.List(c.Request.Context(), page, pageSize)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":     accounts,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetAllCloudAccounts 获取所有启用的云平台账号
func (s *HostService) GetAllCloudAccounts(c *gin.Context) {
	accounts, err := s.cloudUseCase.GetAll(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, accounts)
}

// GetCloudInstances 获取云平台的实例列表
func (s *HostService) GetCloudInstances(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的账号ID")
		return
	}

	// 从查询参数获取区域
	region := c.Query("region")
	if region == "" {
		response.ErrorCode(c, http.StatusBadRequest, "请指定区域")
		return
	}

	instances, err := s.cloudUseCase.GetInstances(c.Request.Context(), uint(id), region)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取实例列表失败: "+err.Error())
		return
	}

	response.Success(c, instances)
}

// GetCloudRegions 获取云平台的区域列表
func (s *HostService) GetCloudRegions(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的账号ID")
		return
	}

	regions, err := s.cloudUseCase.GetRegions(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取区域列表失败: "+err.Error())
		return
	}

	response.Success(c, regions)
}

// ImportFromCloud 从云平台导入主机
func (s *HostService) ImportFromCloud(c *gin.Context) {
	var req asset.CloudImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := s.cloudUseCase.ImportFromCloud(c.Request.Context(), &req, s.hostUseCase); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "导入失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "导入成功", nil)
}

// CollectHostInfo 采集主机信息
func (s *HostService) CollectHostInfo(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的主机ID")
		return
	}

	if err := s.hostUseCase.CollectHostInfo(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "采集失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "采集成功", nil)
}

// TestHostConnection 测试主机连接
func (s *HostService) TestHostConnection(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的主机ID")
		return
	}

	if err := s.hostUseCase.TestConnection(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "连接失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "连接成功", nil)
}

// BatchCollectHostInfo 批量采集主机信息
func (s *HostService) BatchCollectHostInfo(c *gin.Context) {
	var req struct {
		HostIDs []uint `json:"hostIds" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := s.hostUseCase.BatchCollectHostInfo(c.Request.Context(), req.HostIDs); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "批量采集失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "批量采集完成", nil)
}

// BatchDeleteHosts 批量删除主机
func (s *HostService) BatchDeleteHosts(c *gin.Context) {
	var req struct {
		HostIDs []uint `json:"hostIds" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := s.hostUseCase.BatchDelete(c.Request.Context(), req.HostIDs); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "批量删除失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "批量删除成功", nil)
}

// DownloadExcelTemplate 下载Excel导入模板
func (s *HostService) DownloadExcelTemplate(c *gin.Context) {
	f := excelize.NewFile()
	// 获取默认sheet名称 (默认是 "Sheet1")
	sheetName := f.GetSheetName(0)

	// 设置列标题
	headers := []string{"主机名称*", "分组编码", "SSH用户名*", "IP地址*", "SSH端口*", "凭证名称", "标签", "备注"}

	// 创建标题行样式
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#E6E6FA"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})

	// 设置列宽
	f.SetColWidth(sheetName, "A", "H", 20)

	// 写入标题行
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	// 添加示例数据
	examples := [][]string{
		{"Web服务器-01", "BIBF", "root", "192.168.1.100", "22", "生产环境凭证", "web,生产", "Web应用服务器"},
		{"数据库服务器", "TEST", "ubuntu", "192.168.1.200", "22", "", "db,测试", "MySQL数据库服务器"},
	}
	for i, example := range examples {
		for j, val := range example {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+2)
			f.SetCellValue(sheetName, cell, val)
		}
	}

	// 添加说明行
	for i, note := range []string{
		"说明：",
		"1. 带*号的是必填项",
		"2. 分组编码：需要在系统中已存在，可在资产分组管理中查看",
		"3. SSH端口：默认22",
		"4. 凭证名称：需要在系统中已存在，可在凭证管理中查看",
		"5. 标签：多个标签用逗号分隔",
	} {
		cell, _ := excelize.CoordinatesToCellName(1, 5+i)
		f.SetCellValue(sheetName, cell, note)
	}

	// 生成到buffer
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "生成模板文件失败")
		return
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=host_import_template.xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

// ImportFromExcel Excel批量导入主机
func (s *HostService) ImportFromExcel(c *gin.Context) {
	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "请选择要上传的文件")
		return
	}

	// 获取主机类型（默认为自建主机）
	hostType := c.PostForm("type")
	if hostType == "" {
		hostType = "self"
	}

	// 获取分组ID
	var groupID uint
	if groupIDStr := c.PostForm("groupId"); groupIDStr != "" {
		id, err := strconv.ParseUint(groupIDStr, 10, 32)
		if err == nil {
			groupID = uint(id)
		}
	}

	// 检查文件类型
	if file.Header.Get("Content-Type") != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" &&
		!strings.HasSuffix(file.Filename, ".xlsx") {
		response.ErrorCode(c, http.StatusBadRequest, "请上传Excel文件(.xlsx格式)")
		return
	}

	// 读取文件内容
	src, err := file.Open()
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "打开文件失败")
		return
	}
	defer src.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(src); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "读取文件失败")
		return
	}

	// 调用导入方法
	result, err := s.hostUseCase.ImportFromExcelWithType(c.Request.Context(), buf.Bytes(), hostType, groupID)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "导入失败: "+err.Error())
		return
	}

	response.Success(c, result)
}
