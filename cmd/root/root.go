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

package root

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// Cmd represents the base command when called without any subcommands
var Cmd = &cobra.Command{
	Use:   "opshub",
	Short: "运维管理平台",
	Long:  `OpsHub 是一个基于 Gin 的运维管理平台后端服务`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return Cmd.Execute()
}

func init() {
	// 持久化标志(Persistent Flags): 对所有子命令都可用
	Cmd.PersistentFlags().StringVar(&cfgFile, "config", "", "配置文件路径 (默认为 config/config.yaml)")
	Cmd.PersistentFlags().StringP("mode", "m", "", "运行模式: debug, release, test")
	Cmd.PersistentFlags().StringP("log-level", "l", "", "日志级别: debug, info, warn, error")

	// 绑定到 viper
	Cmd.PersistentFlags().String("server.http-addr", "", "HTTP 服务监听地址")
	Cmd.PersistentFlags().Int("server.http-port", 0, "HTTP 服务监听端口")
	Cmd.PersistentFlags().String("database.host", "", "数据库主机地址")
	Cmd.PersistentFlags().Int("database.port", 0, "数据库端口")
	Cmd.PersistentFlags().String("database.username", "", "数据库用户名")
	Cmd.PersistentFlags().String("database.password", "", "数据库密码")
	Cmd.PersistentFlags().String("database.database", "", "数据库名称")
	Cmd.PersistentFlags().String("redis.host", "", "Redis 主机地址")
	Cmd.PersistentFlags().Int("redis.port", 0, "Redis 端口")
	Cmd.PersistentFlags().String("redis.password", "", "Redis 密码")

	// 绑定标志到 viper
	if err := viper.BindPFlags(Cmd.PersistentFlags()); err != nil {
		panic(err)
	}

	if err := viper.BindPFlags(Cmd.Flags()); err != nil {
		panic(err)
	}

	// Cobra 也支持 shell 自动补全
	// 当用户输入 <program> completion [bash|zsh|fish|powershell] 时会生成补全脚本
	Cmd.CompletionOptions.DisableDefaultCmd = true
}

// GetConfigFile 获取配置文件路径
func GetConfigFile() string {
	if cfgFile != "" {
		return cfgFile
	}
	return "config/config.yaml"
}
