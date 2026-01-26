package rbac

import (
	"github.com/gin-gonic/gin"
	"github.com/mojocn/base64Captcha"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

var store = base64Captcha.DefaultMemStore

// CaptchaService 验证码服务
type CaptchaService struct{}

// NewCaptchaService 创建验证码服务
func NewCaptchaService() *CaptchaService {
	return &CaptchaService{}
}

// CaptchaResponse 验证码响应
type CaptchaResponse struct {
	CaptchaId   string `json:"captchaId"`
	CaptchaId2  string `json:"captchaId2"` // 兼容前端可能的字段名
	Image       string `json:"image"`
	ImageBase64 string `json:"imageBase64"` // 兼容前端可能的字段名
}

// GetCaptcha 获取验证码
// @Summary 获取验证码
// @Description 获取登录用的验证码图片和ID
// @Tags 认证管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{} "获取成功"
// @Router /api/v1/captcha [get]
func (s *CaptchaService) GetCaptcha(c *gin.Context) {
	// 生成验证码配置
	driver := base64Captcha.NewDriverDigit(
		80,   // 高度
		240,  // 宽度
		5,    // 验证码长度
		0.7,  // 噪点强度
		4,    // 数字个数
	)

	// 生成验证码
	cp := base64Captcha.NewCaptcha(driver, store)
	id, b64s, _, err := cp.Generate()
	if err != nil {
		response.ErrorCode(c, 500, "验证码生成失败")
		return
	}

	response.Success(c, CaptchaResponse{
		CaptchaId:   id,
		CaptchaId2:  id,
		Image:       b64s,
		ImageBase64: b64s,
	})
}

// VerifyCaptchaRequest 验证码验证请求
type VerifyCaptchaRequest struct {
	CaptchaId  string `json:"captchaId"`
	CaptchaId2 string `json:"captchaId2"` // 兼容前端可能的字段名
	CaptchaCode string `json:"captchaCode"`
}

// VerifyCaptcha 验证验证码
// @Summary 验证验证码
// @Description 验证用户输入的验证码是否正确
// @Tags 认证管理
// @Accept json
// @Produce json
// @Param body body VerifyCaptchaRequest true "验证码信息"
// @Success 200 {object} response.Response{data=object} "验证成功"
// @Failure 400 {object} response.Response "验证码错误"
// @Router /api/v1/captcha/verify [post]
func (s *CaptchaService) VerifyCaptcha(c *gin.Context) {
	var req VerifyCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, 400, "参数错误")
		return
	}

	// 兼容可能的字段名差异
	captchaId := req.CaptchaId
	if captchaId == "" {
		captchaId = req.CaptchaId2
	}

	if captchaId == "" || req.CaptchaCode == "" {
		response.ErrorCode(c, 400, "验证码参数不能为空")
		return
	}

	// 验证验证码
	if store.Verify(captchaId, req.CaptchaCode, true) {
		response.Success(c, gin.H{"valid": true})
	} else {
		response.ErrorCode(c, 400, "验证码错误")
	}
}

// VerifyCaptchaDirect 直接验证验证码（用于内部调用）
func (s *CaptchaService) VerifyCaptchaDirect(captchaId, captchaCode string) bool {
	if captchaId == "" || captchaCode == "" {
		return false
	}
	return store.Verify(captchaId, captchaCode, true)
}
