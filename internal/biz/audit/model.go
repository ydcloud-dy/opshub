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

package audit

import (
	"time"
	"gorm.io/gorm"
)

// SysOperationLog 操作日志表
type SysOperationLog struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`

	// 用户信息
	UserID   uint   `gorm:"index;comment:用户ID" json:"userId"`
	Username string `gorm:"type:varchar(50);comment:用户名" json:"username"`
	RealName string `gorm:"type:varchar(50);comment:真实姓名" json:"realName"`

	// 操作信息
	Module      string `gorm:"type:varchar(50);comment:模块名称" json:"module"`         // 模块：用户管理、角色管理、主机管理等
	Action      string `gorm:"type:varchar(50);comment:操作类型" json:"action"`         // 操作：登录、查询、创建、更新、删除
	Description string `gorm:"type:varchar(200);comment:操作描述" json:"description"`   // 操作描述

	// 请求信息
	Method string `gorm:"type:varchar(10);comment:请求方法" json:"method"` // GET, POST, PUT, DELETE
	Path   string `gorm:"type:varchar(200);comment:请求路径" json:"path"` // /api/v1/users
	Params string `gorm:"type:text;comment:请求参数" json:"params"`       // JSON格式的请求参数

	// 响应信息
	Status   int    `gorm:"type:int;comment:状态码" json:"status"`                 // 200, 400, 500等
	ErrorMsg string `gorm:"type:text;comment:错误信息" json:"errorMsg"`            // 错误信息
	CostTime int64  `gorm:"type:bigint;comment:耗时(毫秒)" json:"costTime"`        // 请求耗时

	// 环境信息
	IP        string `gorm:"type:varchar(50);comment:IP地址" json:"ip"`
	UserAgent string `gorm:"type:varchar(500);comment:用户代理" json:"userAgent"`
}

// SysLoginLog 登录日志表
type SysLoginLog struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`

	// 用户信息
	UserID   uint   `gorm:"index;comment:用户ID" json:"userId"`
	Username string `gorm:"type:varchar(50);index;comment:用户名" json:"username"`
	RealName string `gorm:"type:varchar(50);comment:真实姓名" json:"realName"`

	// 登录信息
	LoginType   string    `gorm:"type:varchar(20);comment:登录类型" json:"loginType"`   // web, ssh, api
	LoginStatus string    `gorm:"type:varchar(20);comment:登录状态" json:"loginStatus"` // success, failed
	LoginTime   time.Time `gorm:"comment:登录时间" json:"loginTime"`
	LogoutTime  *time.Time `gorm:"comment:登出时间" json:"logoutTime,omitempty"`

	// 环境信息
	IP        string `gorm:"type:varchar(50);comment:IP地址" json:"ip"`
	Location  string `gorm:"type:varchar(100);comment:登录地点" json:"location"`
	UserAgent string `gorm:"type:varchar(500);comment:用户代理" json:"userAgent"`

	// 失败原因
	FailReason string `gorm:"type:varchar(200);comment:失败原因" json:"failReason,omitempty"`
}

// SysDataLog 数据日志表
type SysDataLog struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`

	// 用户信息
	UserID   uint   `gorm:"index;comment:用户ID" json:"userId"`
	Username string `gorm:"type:varchar(50);comment:用户名" json:"username"`
	RealName string `gorm:"type:varchar(50);comment:真实姓名" json:"realName"`

	// 数据信息
	TableName  string `gorm:"column:table_name;type:varchar(50);comment:表名" json:"tableName"`   // sys_user, sys_role等
	RecordID   uint   `gorm:"index;comment:记录ID" json:"recordId"`             // 数据记录的主键ID
	Action     string `gorm:"type:varchar(20);comment:操作类型" json:"action"`   // create, update, delete
	OldData    string `gorm:"type:longtext;comment:原始数据" json:"oldData"`     // JSON格式的原始数据
	NewData    string `gorm:"type:longtext;comment:新数据" json:"newData"`       // JSON格式的新数据
	DiffFields string `gorm:"type:text;comment:差异字段" json:"diffFields"`     // 变更的字段列表

	// 环境信息
	IP        string `gorm:"type:varchar(50);comment:IP地址" json:"ip"`
	UserAgent string `gorm:"type:varchar(500);comment:用户代理" json:"userAgent"`
}

// Table 指定表名
func (SysDataLog) Table() string {
	return "sys_data_log"
}

// TableName 指定表名
func (SysOperationLog) TableName() string {
	return "sys_operation_log"
}

func (SysLoginLog) TableName() string {
	return "sys_login_log"
}
