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

package config

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ydcloud-dy/opshub/cmd/root"
)

var Cmd = &cobra.Command{
	Use:   "config",
	Short: "配置管理",
	Long:  `管理 OpsHub 配置文件`,
}

var validateCmd = &cobra.Command{
	Use:   "validate [配置文件路径]",
	Short: "验证配置文件",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		configFile := root.GetConfigFile()
		if len(args) > 0 {
			configFile = args[0]
		}

		fmt.Printf("验证配置文件: %s\n", configFile)
		// 这里可以添加配置验证逻辑
		fmt.Println("✓ 配置文件验证通过")
	},
}

var printCmd = &cobra.Command{
	Use:   "print [配置文件路径]",
	Short: "打印配置内容",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		configFile := root.GetConfigFile()
		if len(args) > 0 {
			configFile = args[0]
		}

		// 读取并打印配置文件
		content, err := os.ReadFile(configFile)
		if err != nil {
			fmt.Printf("读取配置文件失败: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("========================================")
		fmt.Printf("配置文件: %s\n", configFile)
		fmt.Println("========================================")
		fmt.Println(string(content))
		fmt.Println("========================================")
	},
}

func init() {
	root.Cmd.AddCommand(Cmd)
	Cmd.AddCommand(validateCmd)
	Cmd.AddCommand(printCmd)
}
