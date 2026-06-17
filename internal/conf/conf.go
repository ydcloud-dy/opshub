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

package conf

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync/atomic"

	"github.com/spf13/viper"
)

// Config 全局配置
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Log      LogConfig      `mapstructure:"log"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Mode         string `mapstructure:"mode"` // debug, release, test
	HttpPort     int    `mapstructure:"http_port"`
	RPCPort      int    `mapstructure:"rpc_port"`
	ReadTimeout  int    `mapstructure:"read_timeout"`  // 毫秒
	WriteTimeout int    `mapstructure:"write_timeout"` // 毫秒
	JWTSecret    string `mapstructure:"jwt_secret"`    // JWT密钥
	ExternalURL  string `mapstructure:"external_url"`  // 外部访问URL，用于OAuth2 issuer
	FrontendURL  string `mapstructure:"frontend_url"`  // 前端URL，用于OAuth2登录重定向
}

// GetOAuth2Issuer 获取OAuth2 issuer URL
// 本地开发时使用后端端口，因为 Jenkins 服务器端需要直接调用 token endpoint
func (c *ServerConfig) GetOAuth2Issuer() string {
	if c.ExternalURL != "" {
		return c.ExternalURL
	}
	// 本地开发默认使用后端端口
	return fmt.Sprintf("http://localhost:%d", c.HttpPort)
}

// GetFrontendURL 获取前端URL
func (c *ServerConfig) GetFrontendURL() string {
	if c.FrontendURL != "" {
		return c.FrontendURL
	}
	if c.ExternalURL != "" {
		return c.ExternalURL
	}
	// 本地开发默认使用 Vite 端口
	return "http://localhost:5173"
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver          string `mapstructure:"driver"`
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	Database        string `mapstructure:"database"`
	Username        string `mapstructure:"username"`
	Password        string `mapstructure:"password"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"` // 秒
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host        string `mapstructure:"host"`
	Port        int    `mapstructure:"port"`
	Password    string `mapstructure:"password"`
	DB          int    `mapstructure:"db"`
	PoolSize    int    `mapstructure:"pool_size"`
	MinIdleConn int    `mapstructure:"min_idle_conn"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"` // MB
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"` // days
	Compress   bool   `mapstructure:"compress"`
	Console    bool   `mapstructure:"console"`
}

var globalConfig *Config
var observedFrontendURL atomic.Value

// Load 加载配置
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// 设置配置文件
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// 环境变量前缀
	v.SetEnvPrefix("OPSHUB")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析配置
	config := &Config{}
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	globalConfig = config
	return config, nil
}

// Get 获取全局配置
func Get() *Config {
	return globalConfig
}

func RecordObservedFrontendURL(rawURL string) {
	rawURL = normalizeBaseURL(rawURL)
	if rawURL == "" {
		return
	}
	observedFrontendURL.Store(rawURL)
}

func GetObservedFrontendURL() string {
	if value, ok := observedFrontendURL.Load().(string); ok {
		return value
	}
	return ""
}

func GetNotificationFrontendURL() string {
	if globalConfig != nil {
		if frontendURL := normalizeBaseURL(globalConfig.Server.FrontendURL); frontendURL != "" {
			return frontendURL
		}
		if externalURL := normalizeBaseURL(globalConfig.Server.ExternalURL); externalURL != "" {
			return externalURL
		}
	}
	if observedURL := GetObservedFrontendURL(); observedURL != "" {
		return observedURL
	}
	return autoDetectedFrontendURL()
}

func normalizeBaseURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	rawURL = strings.TrimRight(rawURL, "/")
	if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		return rawURL
	}
	return "http://" + rawURL
}

func autoDetectedFrontendURL() string {
	if host := strings.TrimSpace(os.Getenv("OPSHUB_PUBLIC_HOST")); host != "" {
		scheme := strings.TrimSpace(os.Getenv("OPSHUB_PUBLIC_SCHEME"))
		if scheme == "" {
			scheme = "http"
		}
		if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
			return normalizeBaseURL(host)
		}
		port := strings.TrimSpace(os.Getenv("OPSHUB_PUBLIC_PORT"))
		if port != "" && !strings.Contains(host, ":") {
			host += ":" + port
		}
		return normalizeBaseURL(scheme + "://" + host)
	}
	ip := firstNonLoopbackIPv4()
	if ip == "" {
		return ""
	}
	port := strings.TrimSpace(os.Getenv("FRONTEND_PORT"))
	if port == "" {
		port = strings.TrimSpace(os.Getenv("OPSHUB_FRONTEND_PORT"))
	}
	if port == "" || port == "80" {
		return "http://" + ip
	}
	return "http://" + ip + ":" + port
}

func firstNonLoopbackIPv4() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue
			}
			return ip.String()
		}
	}
	return ""
}

// GetDSN 获取数据库连接字符串
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		c.Username,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
	)
}

// GetRedisAddr 获取Redis地址
func (c *RedisConfig) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
