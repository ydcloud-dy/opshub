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

package version

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ydcloud-dy/opshub/cmd/root"
)

var (
	// Version 版本号
	Version = "1.0.0"
	// GitCommit Git提交哈希
	GitCommit = "unknown"
	// BuildTime 构建时间
	BuildTime = "unknown"
	// GoVersion Go版本
	GoVersion = "unknown"
)

var Cmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Long:  `显示 OpsHub 的版本信息`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("========================================")
		fmt.Println("           OpsHub 运维管理平台")
		fmt.Println("========================================")
		fmt.Printf("版本号:     %s\n", Version)
		fmt.Printf("Git提交:    %s\n", GitCommit)
		fmt.Printf("构建时间:   %s\n", BuildTime)
		fmt.Printf("Go版本:     %s\n", GoVersion)
		fmt.Println("========================================")
	},
}

func init() {
	root.Cmd.AddCommand(Cmd)
}
