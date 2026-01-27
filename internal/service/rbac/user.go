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
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/biz/audit"
	"github.com/ydcloud-dy/opshub/internal/biz/rbac"
	"github.com/ydcloud-dy/opshub/pkg/response"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"go.uber.org/zap"
)

type UserService struct {
	userUseCase     *rbac.UserUseCase
	authService     *AuthService
	captchaService  *CaptchaService
	loginLogUseCase *audit.LoginLogUseCase
}

func NewUserService(userUseCase *rbac.UserUseCase, authService *AuthService) *UserService {
	return &UserService{
		userUseCase: userUseCase,
		authService: authService,
	}
}

// SetCaptchaService 设置验证码服务（通过依赖注入）
func (s *UserService) SetCaptchaService(captchaService *CaptchaService) {
	s.captchaService = captchaService
}

// SetLoginLogUseCase 设置登录日志用例（通过依赖注入）
func (s *UserService) SetLoginLogUseCase(loginLogUseCase *audit.LoginLogUseCase) {
	s.loginLogUseCase = loginLogUseCase
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password" binding:"required"`
	CaptchaId  string `json:"captchaId"`
	CaptchaId2 string `json:"captchaId2"` // 兼容前端可能的字段名
	CaptchaCode string `json:"captchaCode"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token string       `json:"token"`
	User  *rbac.SysUser `json:"user"`
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
	RealName string `json:"realName"`
	Email    string `json:"email" binding:"required,email"`
	Phone    string `json:"phone"`
}

// Login 用户登录
// @Summary 用户登录
// @Description 用户使用用户名、密码和验证码登录系统
// @Tags 认证管理
// @Accept json
// @Produce json
// @Param body body LoginRequest true "登录信息"
// @Success 200 {object} response.Response "登录成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/public/login [post]
func (s *UserService) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	// 添加调试日志
	appLogger.Info("用户登录尝试", zap.String("username", req.Username))

	// 获取客户端信息
	clientIP := c.ClientIP()
	userAgent := c.Request.UserAgent()

	// 验证验证码
	captchaId := req.CaptchaId
	if captchaId == "" {
		captchaId = req.CaptchaId2
	}

	if captchaId == "" || req.CaptchaCode == "" {
		// 记录登录日志 - 验证码为空
		s.recordLoginLog(req.Username, "web", "failed", clientIP, userAgent, "验证码为空", 0)
		response.ErrorCode(c, http.StatusOK, "请输入验证码")
		return
	}

	// 使用验证码服务验证
	if s.captchaService == nil {
		s.recordLoginLog(req.Username, "web", "failed", clientIP, userAgent, "验证码服务未初始化", 0)
		response.ErrorCode(c, http.StatusOK, "验证码服务未初始化")
		return
	}

	if !s.captchaService.VerifyCaptchaDirect(captchaId, req.CaptchaCode) {
		appLogger.Info("验证码验证失败", zap.String("username", req.Username), zap.String("captchaId", captchaId))
		// 记录登录日志 - 验证码错误
		s.recordLoginLog(req.Username, "web", "failed", clientIP, userAgent, "验证码错误", 0)
		response.ErrorCode(c, http.StatusOK, "验证码错误，请重新输入")
		return
	}

	user, err := s.userUseCase.ValidatePassword(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		appLogger.Error("登录失败", zap.String("username", req.Username), zap.Error(err))
		// 记录登录日志 - 用户名或密码错误
		s.recordLoginLog(req.Username, "web", "failed", clientIP, userAgent, err.Error(), 0)
		response.ErrorCode(c, http.StatusOK, err.Error())
		return
	}

	if user.Status != 1 {
		// 记录登录日志 - 用户被禁用
		s.recordLoginLog(req.Username, "web", "failed", clientIP, userAgent, "用户已被禁用", user.ID)
		response.ErrorCode(c, http.StatusOK, "用户已被禁用")
		return
	}

	token, err := s.authService.GenerateToken(user.ID, user.Username)
	if err != nil {
		// 记录登录日志 - 生成token失败
		s.recordLoginLog(req.Username, "web", "failed", clientIP, userAgent, "生成token失败", user.ID)
		response.ErrorCode(c, http.StatusInternalServerError, "生成token失败")
		return
	}

	// 清空密码字段，防止返回给前端
	user.Password = ""

	// 更新最后登录时间
	_ = s.userUseCase.Update(c.Request.Context(), user)

	// 记录登录日志 - 登录成功
	s.recordLoginLog(req.Username, "web", "success", clientIP, userAgent, "", user.ID)

	appLogger.Info("用户登录成功", zap.String("username", req.Username))

	response.Success(c, LoginResponse{
		Token: token,
		User:  user,
	})
}

// recordLoginLog 记录登录日志
func (s *UserService) recordLoginLog(username, loginType, loginStatus, ip, userAgent, failReason string, userID uint) {
	if s.loginLogUseCase == nil {
		return
	}

	realName := ""
	if userID != 0 {
		// 获取用户真实姓名
		if user, err := s.userUseCase.GetByID(context.Background(), userID); err == nil {
			realName = user.RealName
		}
	}

	log := &audit.SysLoginLog{
		UserID:      userID,
		Username:    username,
		RealName:    realName,
		LoginType:   loginType,
		LoginStatus: loginStatus,
		LoginTime:   time.Now(),
		IP:          ip,
		Location:    "", // 可以根据IP解析地理位置
		UserAgent:   userAgent,
		FailReason:  failReason,
	}

	// 异步保存登录日志
	go func() {
		if err := s.loginLogUseCase.Create(context.Background(), log); err != nil {
			appLogger.Error("保存登录日志失败",
				zap.Error(err),
				zap.String("username", username),
			)
		}
	}()
}

// Register 用户注册
// @Summary 用户注册
// @Description 新用户使用用户名、密码等信息注册账号
// @Tags 认证管理
// @Accept json
// @Produce json
// @Param body body RegisterRequest true "注册信息"
// @Success 200 {object} response.Response{} "注册成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/public/register [post]
func (s *UserService) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	user := &rbac.SysUser{
		Username: req.Username,
		Password: req.Password,
		RealName: req.RealName,
		Email:    req.Email,
		Phone:    req.Phone,
		Status:   1,
	}

	if err := s.userUseCase.Create(c.Request.Context(), user); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "注册失败: "+err.Error())
		return
	}

	response.Success(c, user)
}

// GetProfile 获取当前用户信息
// @Summary 获取当前用户信息
// @Description 获取登录用户的个人信息（需要Bearer Token认证）
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{} "获取成功"
// @Failure 401 {object} response.Response "未登录"
// @Router /api/v1/profile [get]
func (s *UserService) GetProfile(c *gin.Context) {
	userID := GetUserID(c)
	if userID == 0 {
		response.ErrorCode(c, http.StatusUnauthorized, "未登录")
		return
	}

	user, err := s.userUseCase.GetByID(c.Request.Context(), userID)
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "用户不存在")
		return
	}

	// 清空密码字段，防止返回给前端
	user.Password = ""

	response.Success(c, user)
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6"`
}

// ChangePassword 修改自己的密码
// @Summary 修改用户密码
// @Description 用户修改自己的登录密码
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body ChangePasswordRequest true "密码信息"
// @Success 200 {object} response.Response "修改成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/profile/password [put]
func (s *UserService) ChangePassword(c *gin.Context) {
	userID := GetUserID(c)
	if userID == 0 {
		response.ErrorCode(c, http.StatusUnauthorized, "未登录")
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := s.userUseCase.UpdatePassword(c.Request.Context(), userID, req.OldPassword, req.NewPassword); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "密码修改成功", nil)
}

// CreateUser 创建用户
// @Summary 创建用户
// @Description 管理员创建新用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body rbac.SysUser true "用户信息"
// @Success 200 {object} response.Response{} "创建成功"
// @Router /api/v1/users [post]
func (s *UserService) CreateUser(c *gin.Context) {
	var req rbac.SysUser
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := s.userUseCase.Create(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}

	// 清空密码字段，防止返回给前端
	req.Password = ""

	response.Success(c, req)
}

// UpdateUser 更新用户
// @Summary 更新用户信息
// @Description 管理员更新用户信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "用户ID"
// @Param body body rbac.SysUser true "用户信息"
// @Success 200 {object} response.Response{} "更新成功"
// @Router /api/v1/users/{id} [put]
func (s *UserService) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的用户ID")
		return
	}

	var req rbac.SysUser
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	req.ID = uint(id)
	if err := s.userUseCase.Update(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}

	// 重新获取完整的用户数据，包含Roles和Positions
	user, err := s.userUseCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取用户信息失败")
		return
	}

	// 清空密码字段，防止返回给前端
	user.Password = ""

	response.Success(c, user)
}

// DeleteUser 删除用户
// @Summary 删除用户
// @Description 管理员删除用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response "删除成功"
// @Router /api/v1/users/{id} [delete]
func (s *UserService) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的用户ID")
		return
	}

	if err := s.userUseCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetUser 获取用户详情
// @Summary 获取用户详情
// @Description 获取单个用户的详细信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response{} "获取成功"
// @Router /api/v1/users/{id} [get]
func (s *UserService) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的用户ID")
		return
	}

	user, err := s.userUseCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "用户不存在")
		return
	}

	// 清空密码字段，防止返回给前端
	user.Password = ""

	response.Success(c, user)
}

// ListUsers 用户列表
// @Summary 获取用户列表
// @Description 分页获取用户列表，支持按关键字和部门筛选
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Param keyword query string false "搜索关键字"
// @Param departmentId query int false "部门ID"
// @Success 200 {object} response.Response{} "获取成功"
// @Router /api/v1/users [get]
func (s *UserService) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	keyword := c.Query("keyword")
	departmentID, _ := strconv.ParseUint(c.Query("departmentId"), 10, 32)

	users, total, err := s.userUseCase.List(c.Request.Context(), page, pageSize, keyword, uint(departmentID))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	// 清空所有用户的密码字段，防止返回给前端
	for _, user := range users {
		user.Password = ""
	}

	response.Success(c, gin.H{
		"list":     users,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// AssignUserRoles 分配用户角色
type AssignUserRolesRequest struct {
	RoleIDs []uint `json:"roleIds" binding:"required"`
}

// AssignUserRoles 分配用户角色
// @Summary 分配用户角色
// @Description 为用户分配角色
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "用户ID"
// @Param body body AssignUserRolesRequest true "角色IDs"
// @Success 200 {object} response.Response "分配成功"
// @Router /api/v1/users/{id}/roles [post]
func (s *UserService) AssignUserRoles(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的用户ID")
		return
	}

	var req AssignUserRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := s.userUseCase.AssignRoles(c.Request.Context(), uint(id), req.RoleIDs); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "分配失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

// AssignUserPositions 分配用户岗位
type AssignUserPositionsRequest struct {
	PositionIDs []uint `json:"positionIds"`
}

// AssignUserPositions 分配用户岗位
// @Summary 分配用户岗位
// @Description 为用户分配岗位
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "用户ID"
// @Param body body AssignUserPositionsRequest true "岗位IDs"
// @Success 200 {object} response.Response "分配成功"
// @Router /api/v1/users/{id}/positions [post]
func (s *UserService) AssignUserPositions(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的用户ID")
		return
	}

	var req AssignUserPositionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := s.userUseCase.AssignPositions(c.Request.Context(), uint(id), req.PositionIDs); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "分配失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

// ResetPasswordRequest 重置密码请求
type ResetPasswordRequest struct {
	Password string `json:"password" binding:"required,min=6"`
}

// ResetPassword 重置用户密码
// @Summary 重置用户密码
// @Description 管理员重置用户密码
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "用户ID"
// @Param body body ResetPasswordRequest true "新密码"
// @Success 200 {object} response.Response "重置成功"
// @Router /api/v1/users/{id}/password/reset [post]
func (s *UserService) ResetPassword(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的用户ID")
		return
	}

	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := s.userUseCase.ResetPassword(c.Request.Context(), uint(id), req.Password); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "重置密码失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "密码重置成功", nil)
}
