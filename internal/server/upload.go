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
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/biz/rbac"
	rbaccustom "github.com/ydcloud-dy/opshub/internal/service/rbac"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// UploadServer 上传服务
type UploadServer struct {
	db        *gorm.DB
	uploadDir string
	uploadURL string
}

// NewUploadServer 创建上传服务
func NewUploadServer(db *gorm.DB, uploadDir, uploadURL string) *UploadServer {
	// 确保上传目录存在
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		appLogger.Error("创建上传目录失败", zap.Error(err))
	}

	return &UploadServer{
		db:        db,
		uploadDir: uploadDir,
		uploadURL: uploadURL,
	}
}

// UploadAvatar 上传头像
// @Summary 上传用户头像
// @Description 上传用户头像图片，支持常见图片格式，最大2MB
// @Tags 文件上传
// @Accept multipart/form-data
// @Produce json
// @Security Bearer
// @Param file formData file true "头像图片文件"
// @Success 200 {object} map[string]interface{} "上传成功"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 401 {object} map[string]interface{} "未登录"
// @Router /api/v1/upload/avatar [post]
func (s *UploadServer) UploadAvatar(c *gin.Context) {
	// 获取上传的文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "获取文件失败",
		})
		return
	}
	defer file.Close()

	// 验证文件类型
	if !strings.HasPrefix(header.Header.Get("Content-Type"), "image/") {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "只能上传图片文件",
		})
		return
	}

	// 验证文件大小 (2MB)
	if header.Size > 2*1024*1024 {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "图片大小不能超过 2MB",
		})
		return
	}

	// 获取当前用户ID (从JWT中获取)
	userID := rbaccustom.GetUserID(c)
	if userID == 0 {
		c.JSON(401, gin.H{
			"code":    401,
			"message": "未登录",
		})
		return
	}

	// 生成唯一文件名
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("avatar_%d_%d%s", userID, time.Now().Unix(), ext)

	// 保存文件
	dst := filepath.Join(s.uploadDir, filename)
	out, err := os.Create(dst)
	if err != nil {
		appLogger.Error("创建文件失败", zap.Error(err))
		c.JSON(500, gin.H{
			"code":    500,
			"message": "保存文件失败",
		})
		return
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		appLogger.Error("写入文件失败", zap.Error(err))
		c.JSON(500, gin.H{
			"code":    500,
			"message": "保存文件失败",
		})
		return
	}

	// 生成访问URL
	fileURL := fmt.Sprintf("%s/%s", s.uploadURL, filename)

	appLogger.Info("头像上传成功",
		zap.Uint("userID", userID),
		zap.String("filename", filename),
		zap.String("url", fileURL),
	)

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"url":  fileURL,
			"path": filename,
		},
	})
}

// UpdateUserAvatar 更新用户头像
// @Summary 更新用户头像地址
// @Description 更新当前用户的头像地址
// @Tags 文件上传
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body object true "头像地址" example({"avatar": "/uploads/avatar_1_1234567890.png"})
// @Success 200 {object} map[string]interface{} "更新成功"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 401 {object} map[string]interface{} "未登录"
// @Router /api/v1/profile/avatar [put]
func (s *UploadServer) UpdateUserAvatar(c *gin.Context) {
	var req struct {
		Avatar string `json:"avatar" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "参数错误",
		})
		return
	}

	// 获取当前用户ID
	userID := rbaccustom.GetUserID(c)
	if userID == 0 {
		c.JSON(401, gin.H{
			"code":    401,
			"message": "未登录",
		})
		return
	}

	// 更新用户头像
	if err := s.db.Model(&rbac.SysUser{}).Where("id = ?", userID).Update("avatar", req.Avatar).Error; err != nil {
		appLogger.Error("更新头像失败",
			zap.Uint("userID", userID),
			zap.Error(err),
		)
		c.JSON(500, gin.H{
			"code":    500,
			"message": "更新头像失败",
		})
		return
	}

	appLogger.Info("用户头像更新成功",
		zap.Uint("userID", userID),
		zap.String("avatar", req.Avatar),
	)

	c.JSON(200, gin.H{
		"code":    0,
		"message": "头像更新成功",
	})
}

// UploadPlugin 上传并安装插件
// @Summary 上传插件包
// @Description 上传并安装插件包，只支持 .zip 格式，最大50MB
// @Tags 插件管理
// @Accept multipart/form-data
// @Produce json
// @Security Bearer
// @Param file formData file true "插件zip包"
// @Success 200 {object} map[string]interface{} "安装成功"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Router /api/v1/plugins/upload [post]
func (s *UploadServer) UploadPlugin(c *gin.Context) {
	// 获取上传的文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "获取文件失败",
		})
		return
	}
	defer file.Close()

	// 验证文件类型
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".zip") {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "只能上传 .zip 格式的插件包",
		})
		return
	}

	// 验证文件大小 (50MB)
	if header.Size > 50*1024*1024 {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "插件包大小不能超过 50MB",
		})
		return
	}

	// 创建临时文件保存上传的zip
	tempDir := os.TempDir()
	timestamp := time.Now().Unix()
	zipPath := filepath.Join(tempDir, fmt.Sprintf("plugin_%d.zip", timestamp))

	// 保存上传的文件
	out, err := os.Create(zipPath)
	if err != nil {
		appLogger.Error("创建临时文件失败", zap.Error(err))
		c.JSON(500, gin.H{
			"code":    500,
			"message": "保存文件失败",
		})
		return
	}

	_, err = io.Copy(out, file)
	out.Close()
	if err != nil {
		appLogger.Error("写入文件失败", zap.Error(err))
		os.Remove(zipPath)
		c.JSON(500, gin.H{
			"code":    500,
			"message": "保存文件失败",
		})
		return
	}

	// 延迟清理临时文件
	defer os.Remove(zipPath)

	// 获取项目根目录
	currentDir, err := os.Getwd()
	if err != nil {
		appLogger.Error("获取当前目录失败", zap.Error(err))
		c.JSON(500, gin.H{
			"code":    500,
			"message": "获取当前目录失败",
		})
		return
	}

	// 先检测插件名称（在解压之前）
	pluginName, err := s.detectPluginNameFromZip(zipPath)
	if err != nil {
		appLogger.Warn("检测插件名称失败", zap.Error(err))
		// 不中断，继续尝试
	}

	// 解压插件
	if err := s.extractPlugin(zipPath); err != nil {
		appLogger.Error("解压插件失败", zap.Error(err))
		c.JSON(500, gin.H{
			"code":    500,
			"message": fmt.Sprintf("解压插件失败: %v", err),
		})
		return
	}

	// 自动注册插件（前后端导入和路由）
	if pluginName != "" {
		if err := s.registerPluginToFrontend(currentDir, pluginName); err != nil {
			appLogger.Warn("自动注册前端插件失败",
				zap.String("plugin", pluginName),
				zap.Error(err),
			)
			// 不中断，警告即可
		}

		if err := s.registerPluginToBackend(currentDir, pluginName); err != nil {
			appLogger.Warn("自动注册后端插件失败",
				zap.String("plugin", pluginName),
				zap.Error(err),
			)
			// 不中断，警告即可
		}
	}

	appLogger.Info("插件上传并安装成功",
		zap.String("filename", header.Filename),
		zap.String("pluginName", pluginName),
		zap.Int64("size", header.Size),
	)

	c.JSON(200, gin.H{
		"code":    0,
		"message": "插件安装成功，请重启服务使插件生效",
		"data": gin.H{
			"pluginName": pluginName,
		},
	})
}

// detectPluginNameFromZip 从 zip 包检测插件名称
func (s *UploadServer) detectPluginNameFromZip(zipPath string) (string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", fmt.Errorf("打开zip文件失败: %w", err)
	}
	defer r.Close()

	// 查找后端插件目录结构：backend/{pluginName}/
	detectedNames := make(map[string]bool)
	for _, f := range r.File {
		// 跳过 __MACOSX 等系统目录
		if strings.Contains(f.Name, "__MACOSX") || strings.Contains(f.Name, ".DS_Store") {
			continue
		}

		// 检查结构：{wrapper}/backend/{pluginName}/...
		if strings.Contains(f.Name, "backend/") {
			idx := strings.Index(f.Name, "backend/")
			afterBackend := f.Name[idx+8:] // 长度为 "backend/" = 8
			pluginParts := strings.Split(afterBackend, "/")
			if len(pluginParts) > 0 && pluginParts[0] != "" {
				pluginName := pluginParts[0]
				// 排除文件名，只要目录名（不包含 .）
				if !strings.Contains(pluginName, ".") && len(pluginName) > 0 {
					detectedNames[pluginName] = true
				}
			}
		}
	}

	if len(detectedNames) > 0 {
		// 返回找到的第一个插件名
		for name := range detectedNames {
			appLogger.Info("检测到插件名称",
				zap.String("pluginName", name),
			)
			return name, nil
		}
	}

	return "", fmt.Errorf("无法检测插件名称（未找到 backend/{pluginName} 结构）")
}

// registerPluginToFrontend 自动添加前端插件导入
func (s *UploadServer) registerPluginToFrontend(currentDir, pluginName string) error {
	mainTsPath := filepath.Join(currentDir, "web", "src", "main.ts")

	// 读取文件
	content, err := os.ReadFile(mainTsPath)
	if err != nil {
		return fmt.Errorf("读取 main.ts 失败: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	importLine := fmt.Sprintf("import '@/plugins/%s'", pluginName)

	// 检查导入是否已存在
	for _, line := range lines {
		if strings.Contains(line, importLine) {
			appLogger.Info("前端插件导入已存在，跳过",
				zap.String("plugin", pluginName),
			)
			return nil
		}
	}

	// 找到最后一个现有的插件导入，或 @element-plus/icons-vue 导入
	var insertIdx int = -1
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.Contains(lines[i], "import '@/plugins/") || strings.Contains(lines[i], "import '@element-plus/icons-vue'") {
			insertIdx = i + 1
			break
		}
	}

	// 如果没找到，找第一个空行
	if insertIdx == -1 {
		for i, line := range lines {
			if strings.TrimSpace(line) == "" && i > 0 && strings.Contains(lines[i-1], "import") {
				insertIdx = i
				break
			}
		}
	}

	// 如果还是没找到，插到第14行之后（在 Element Plus 导入之后）
	if insertIdx == -1 {
		insertIdx = 14
	}

	// 在指定位置插入导入语句
	newLines := make([]string, 0, len(lines)+1)
	newLines = append(newLines, lines[:insertIdx]...)
	newLines = append(newLines, importLine)
	newLines = append(newLines, lines[insertIdx:]...)

	// 写回文件
	newContent := strings.Join(newLines, "\n")
	if err := os.WriteFile(mainTsPath, []byte(newContent), 0o644); err != nil {
		return fmt.Errorf("写入 main.ts 失败: %w", err)
	}

	appLogger.Info("前端插件导入添加成功",
		zap.String("plugin", pluginName),
		zap.String("line", importLine),
	)
	return nil
}

// registerPluginToBackend 自动添加后端插件导入和注册
func (s *UploadServer) registerPluginToBackend(currentDir, pluginName string) error {
	httpGoPath := filepath.Join(currentDir, "internal", "server", "http.go")

	// 读取文件
	content, err := os.ReadFile(httpGoPath)
	if err != nil {
		return fmt.Errorf("读取 http.go 失败: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	// 生成导入和注册代码
	// 导入格式：{pluginName}plugin "github.com/ydcloud-dy/opshub/plugins/{pluginName}"
	// 注册格式：pluginMgr.Register({pluginName}plugin.New())
	pluginAlias := pluginName + "plugin"
	importStatement := fmt.Sprintf("\t%s \"github.com/ydcloud-dy/opshub/plugins/%s\"", pluginAlias, pluginName)

	// 检查导入和注册是否已存在
	importExists := false
	registrationExists := false
	for _, line := range lines {
		if strings.Contains(line, fmt.Sprintf("github.com/ydcloud-dy/opshub/plugins/%s", pluginName)) {
			importExists = true
		}
		if strings.Contains(line, fmt.Sprintf("%s.New()", pluginAlias)) {
			registrationExists = true
		}
	}

	if importExists && registrationExists {
		appLogger.Info("后端插件导入和注册已存在，跳过",
			zap.String("plugin", pluginName),
		)
		return nil
	}

	// 添加导入
	if !importExists {
		// 找到最后一个导入语句（在 import 块中）
		var importInsertIdx int = -1
		inImportBlock := false

		for i, line := range lines {
			if strings.Contains(line, "import (") {
				inImportBlock = true
			}
			if inImportBlock && strings.TrimSpace(line) == ")" {
				importInsertIdx = i
				break
			}
		}

		if importInsertIdx > -1 {
			lines = append(lines[:importInsertIdx], append([]string{importStatement}, lines[importInsertIdx:]...)...)
			appLogger.Info("后端插件导入添加成功",
				zap.String("plugin", pluginName),
				zap.String("line", importStatement),
			)
		}
	}

	// 添加注册
	if !registrationExists {
		// 找到最后一个完整的注册块（including closing }）
		var registrationInsertIdx int = -1

		// 从后往前找最后一个 pluginMgr.Register(
		lastRegisterIdx := -1
		for i := len(lines) - 1; i >= 0; i-- {
			if strings.Contains(lines[i], "pluginMgr.Register(") {
				lastRegisterIdx = i
				break
			}
		}

		if lastRegisterIdx > -1 {
			// 找到这个注册块的闭合 }
			braceCount := 0
			foundOpening := false
			blockEndIdx := lastRegisterIdx

			for blockEndIdx < len(lines) {
				line := lines[blockEndIdx]
				if strings.Contains(line, "{") {
					foundOpening = true
					braceCount++
				}
				if strings.Contains(line, "}") {
					braceCount--
					if foundOpening && braceCount == 0 {
						// 找到了闭合的 }
						registrationInsertIdx = blockEndIdx + 1
						break
					}
				}
				blockEndIdx++
			}
		}

		if registrationInsertIdx > -1 {
			// 生成完整的注册块
			fullRegistration := []string{
				"",
				fmt.Sprintf("\t// 注册 %s 插件", pluginName),
				fmt.Sprintf("\tif err := pluginMgr.Register(%s.New()); err != nil {", pluginAlias),
				fmt.Sprintf("\t\tappLogger.Error(\"注册%s插件失败\", zap.Error(err))", pluginName),
				"\t}",
			}

			// 直接将所有行一次性插入
			lines = append(lines[:registrationInsertIdx], append(fullRegistration, lines[registrationInsertIdx:]...)...)

			appLogger.Info("后端插件注册添加成功",
				zap.String("plugin", pluginName),
				zap.String("pluginAlias", pluginAlias),
			)
		}
	}

	// 写回文件
	newContent := strings.Join(lines, "\n")
	if err := os.WriteFile(httpGoPath, []byte(newContent), 0o644); err != nil {
		return fmt.Errorf("写入 http.go 失败: %w", err)
	}

	return nil
}

// extractPlugin 解压插件包
func (s *UploadServer) extractPlugin(zipPath string) error {
	// 打开zip文件
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("打开zip文件失败: %w", err)
	}
	defer r.Close()

	// 获取项目根目录
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %w", err)
	}

	// 遍历zip中的文件
	for _, f := range r.File {
		// 跳过 __MACOSX 等系统文件
		if strings.Contains(f.Name, "__MACOSX") || strings.Contains(f.Name, ".DS_Store") {
			continue
		}

		// 去掉顶层目录（如 test-plugin/），只保留 web/ 或 backend/ 开头的路径
		parts := strings.Split(f.Name, "/")
		if len(parts) < 2 {
			continue
		}

		// 重新组合路径，去掉第一层目录
		relativePath := strings.Join(parts[1:], "/")
		if relativePath == "" {
			continue
		}

		var targetPath string
		if strings.HasPrefix(relativePath, "web/") {
			// 前端插件
			targetPath = filepath.Join(currentDir, relativePath)
		} else if strings.HasPrefix(relativePath, "backend/") {
			// 后端插件，将 backend/ 映射到 plugins/
			pluginPath := strings.TrimPrefix(relativePath, "backend/")
			targetPath = filepath.Join(currentDir, "plugins", pluginPath)
		} else {
			// 跳过不符合格式的文件
			continue
		}

		// 如果是目录，创建目录
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, os.ModePerm); err != nil {
				return fmt.Errorf("创建目录失败: %w", err)
			}
			continue
		}

		// 确保父目录存在
		if err := os.MkdirAll(filepath.Dir(targetPath), os.ModePerm); err != nil {
			return fmt.Errorf("创建父目录失败: %w", err)
		}

		// 提取文件
		if err := s.extractFile(f, targetPath); err != nil {
			return fmt.Errorf("提取文件 %s 失败: %w", f.Name, err)
		}

		appLogger.Info("文件提取成功",
			zap.String("source", f.Name),
			zap.String("target", targetPath),
		)
	}

	return nil
}

// removePluginImportFromFrontend 从前端 main.ts 中删除插件导入
func (s *UploadServer) removePluginImportFromFrontend(currentDir, pluginName string) error {
	mainTsPath := filepath.Join(currentDir, "web", "src", "main.ts")

	// 读取文件
	content, err := os.ReadFile(mainTsPath)
	if err != nil {
		appLogger.Warn("读取 main.ts 失败，可能已被删除",
			zap.String("plugin", pluginName),
			zap.String("path", mainTsPath),
			zap.Error(err),
		)
		return nil // 不作为错误处理
	}

	lines := strings.Split(string(content), "\n")
	var filteredLines []string
	removed := false

	// 删除对应的 import 行
	for _, line := range lines {
		// 查找 import '@/plugins/{pluginName}' 的行
		importPattern := fmt.Sprintf("import '@/plugins/%s'", pluginName)
		if strings.Contains(line, importPattern) {
			removed = true
			appLogger.Info("移除前端插件导入",
				zap.String("plugin", pluginName),
				zap.String("line", strings.TrimSpace(line)),
			)
			continue // 跳过这一行
		}
		filteredLines = append(filteredLines, line)
	}

	if !removed {
		appLogger.Warn("未在 main.ts 中找到插件导入",
			zap.String("plugin", pluginName),
		)
		return nil
	}

	// 写回文件
	newContent := strings.Join(filteredLines, "\n")
	if err := os.WriteFile(mainTsPath, []byte(newContent), 0o644); err != nil {
		appLogger.Error("写入 main.ts 失败",
			zap.String("plugin", pluginName),
			zap.Error(err),
		)
		return fmt.Errorf("写入 main.ts 失败: %w", err)
	}

	appLogger.Info("前端插件导入删除成功",
		zap.String("plugin", pluginName),
	)
	return nil
}

// removePluginImportFromBackend 从后端 http.go 中删除插件导入
func (s *UploadServer) removePluginImportFromBackend(currentDir, pluginName string) error {
	httpGoPath := filepath.Join(currentDir, "internal", "server", "http.go")

	// 读取文件
	content, err := os.ReadFile(httpGoPath)
	if err != nil {
		appLogger.Warn("读取 http.go 失败",
			zap.String("plugin", pluginName),
			zap.String("path", httpGoPath),
			zap.Error(err),
		)
		return nil
	}

	lines := strings.Split(string(content), "\n")
	var filteredLines []string
	importRemoved := false
	registrationRemoved := false
	i := 0

	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// 检查是否是导入行：例如 k8splugin "github.com/ydcloud-dy/opshub/plugins/kubernetes"
		importPattern := fmt.Sprintf("github.com/ydcloud-dy/opshub/plugins/%s", pluginName)
		if strings.Contains(line, importPattern) && strings.Contains(line, "\"") {
			importRemoved = true
			appLogger.Info("移除后端插件导入",
				zap.String("plugin", pluginName),
				zap.String("line", trimmed),
			)
			i++
			continue
		}

		// 检查是否是注册块的开始（包含注释行 // 注册 xxx 插件 或 if err := pluginMgr.Register()）
		pluginAlias := pluginName + "plugin"
		isRegistrationStart := strings.Contains(trimmed, fmt.Sprintf("// 注册 %s 插件", pluginName)) ||
			strings.Contains(trimmed, fmt.Sprintf("if err := pluginMgr.Register(%s.New())", pluginAlias))

		if isRegistrationStart {
			// 找到这个注册块的开始
			blockStart := i

			// 如果这一行是注释，下一行应该是 if 语句
			if strings.Contains(trimmed, "//") {
				i++
				if i < len(lines) {
					// 跳过 if 语句行
					blockStart = i
				}
			}

			// 现在 i 应该指向 if 语句
			// 找到这个块的末尾（找到闭合的 }）
			blockEnd := i
			braceCount := 0
			foundOpening := false

			for blockEnd < len(lines) {
				blockLine := lines[blockEnd]
				blockTrimmed := strings.TrimSpace(blockLine)

				if strings.Contains(blockTrimmed, "{") {
					foundOpening = true
					braceCount++
				}
				if strings.Contains(blockTrimmed, "}") {
					braceCount--
					if foundOpening && braceCount == 0 {
						// 找到了闭合的 }
						break
					}
				}
				blockEnd++
			}

			// 删除从注释行（如果有）到闭合 } 的所有行
			deleteStart := blockStart
			if blockStart > 0 && strings.Contains(strings.TrimSpace(lines[blockStart-1]), fmt.Sprintf("// 注册 %s 插件", pluginName)) {
				deleteStart = blockStart - 1
			}

			// 跳过删除的行，加上一个空行
			i = blockEnd + 1
			if i < len(lines) && strings.TrimSpace(lines[i]) == "" {
				i++ // 跳过空行
			}

			registrationRemoved = true
			appLogger.Info("移除后端插件注册块",
				zap.String("plugin", pluginName),
				zap.Int("startLine", deleteStart),
				zap.Int("endLine", blockEnd),
			)
			continue
		}

		filteredLines = append(filteredLines, line)
		i++
	}

	if !importRemoved && !registrationRemoved {
		appLogger.Warn("未在 http.go 中找到插件导入和注册",
			zap.String("plugin", pluginName),
		)
		return nil
	}

	// 写回文件
	newContent := strings.Join(filteredLines, "\n")
	if err := os.WriteFile(httpGoPath, []byte(newContent), 0o644); err != nil {
		appLogger.Error("写入 http.go 失败",
			zap.String("plugin", pluginName),
			zap.Error(err),
		)
		return fmt.Errorf("写入 http.go 失败: %w", err)
	}

	appLogger.Info("后端插件导入和注册删除成功",
		zap.String("plugin", pluginName),
		zap.Bool("import_removed", importRemoved),
		zap.Bool("registration_removed", registrationRemoved),
	)
	return nil
}

// UninstallPlugin 卸载插件
// @Summary 卸载插件
// @Description 卸载指定的插件，删除相关文件和配置
// @Tags 插件管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param name path string true "插件名称"
// @Success 200 {object} map[string]interface{} "卸载成功"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Router /api/v1/plugins/{name}/uninstall [delete]
func (s *UploadServer) UninstallPlugin(c *gin.Context) {
	pluginName := c.Param("name")
	if pluginName == "" {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "插件名称不能为空",
		})
		return
	}

	// 获取项目根目录
	currentDir, err := os.Getwd()
	if err != nil {
		appLogger.Error("获取当前目录失败", zap.Error(err))
		c.JSON(500, gin.H{
			"code":    500,
			"message": "获取当前目录失败",
		})
		return
	}

	// 第一步：删除前端插件导入（从 main.ts）
	if err := s.removePluginImportFromFrontend(currentDir, pluginName); err != nil {
		appLogger.Warn("删除前端插件导入失败，继续执行",
			zap.String("plugin", pluginName),
			zap.Error(err),
		)
		// 继续执行，不中断
	}

	// 第二步：删除后端插件导入和注册（从 http.go）
	if err := s.removePluginImportFromBackend(currentDir, pluginName); err != nil {
		appLogger.Warn("删除后端插件导入失败，继续执行",
			zap.String("plugin", pluginName),
			zap.Error(err),
		)
		// 继续执行，不中断
	}

	// 第三步：删除前端插件目录
	webPluginDir := filepath.Join(currentDir, "web", "src", "plugins", pluginName)
	if err := os.RemoveAll(webPluginDir); err != nil {
		appLogger.Error("删除前端插件目录失败",
			zap.String("plugin", pluginName),
			zap.String("path", webPluginDir),
			zap.Error(err),
		)
		// 继续执行，不中断
	} else {
		appLogger.Info("前端插件目录删除成功",
			zap.String("plugin", pluginName),
			zap.String("path", webPluginDir),
		)
	}

	// 第四步：删除后端插件目录
	backendPluginDir := filepath.Join(currentDir, "plugins", pluginName)
	if err := os.RemoveAll(backendPluginDir); err != nil {
		appLogger.Error("删除后端插件目录失败",
			zap.String("plugin", pluginName),
			zap.String("path", backendPluginDir),
			zap.Error(err),
		)
		// 继续执行，不中断
	} else {
		appLogger.Info("后端插件目录删除成功",
			zap.String("plugin", pluginName),
			zap.String("path", backendPluginDir),
		)
	}

	// 第五步：从数据库中删除插件状态
	if err := s.db.Exec("DELETE FROM plugin_states WHERE name = ?", pluginName).Error; err != nil {
		appLogger.Error("删除插件状态失败",
			zap.String("plugin", pluginName),
			zap.Error(err),
		)
	} else {
		appLogger.Info("插件状态删除成功", zap.String("plugin", pluginName))
	}

	appLogger.Info("插件卸载成功", zap.String("plugin", pluginName))

	c.JSON(200, gin.H{
		"code":    0,
		"message": "插件卸载成功，请刷新页面并重启服务以生效",
	})
}

// extractFile 提取单个文件
func (s *UploadServer) extractFile(f *zip.File, targetPath string) error {
	// 打开zip中的文件
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	// 创建目标文件
	outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer outFile.Close()

	// 复制内容
	_, err = io.Copy(outFile, rc)
	return err
}
