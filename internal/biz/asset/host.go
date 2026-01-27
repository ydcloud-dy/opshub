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

package asset

import (
	"time"

	"gorm.io/gorm"
)

// Host 主机模型
type Host struct {
	gorm.Model
	Name             string        `gorm:"type:varchar(100);not null;comment:主机名称" json:"name"`
	GroupID          uint          `gorm:"column:group_id;comment:分组ID" json:"groupId"`
	Group            *AssetGroup   `gorm:"-" json:"group,omitempty"`
	Type             string        `gorm:"type:varchar(20);not null;default:'self';comment:主机类型 self:自建 cloud:云主机" json:"type"`
	CloudProvider    string        `gorm:"type:varchar(50);comment:云厂商 aliyun/tencent/aws" json:"cloudProvider,omitempty"`
	CloudInstanceID  string        `gorm:"type:varchar(100);comment:云实例ID" json:"cloudInstanceId,omitempty"`
	CloudAccountID   uint          `gorm:"column:cloud_account_id;comment:云账号ID" json:"cloudAccountId,omitempty"`
	SSHUser          string        `gorm:"type:varchar(50);not null;comment:SSH用户名" json:"sshUser"`
	IP               string        `gorm:"type:varchar(50);not null;comment:IP地址" json:"ip"`
	Port             int           `gorm:"type:int;default:22;comment:SSH端口" json:"port"`
	CredentialID     uint          `gorm:"column:credential_id;comment:凭证ID" json:"credentialId"`
	Credential       *Credential   `gorm:"-" json:"credential,omitempty"`
	Tags             string        `gorm:"type:varchar(500);comment:主机标签(逗号分隔)" json:"tags"`
	Description      string        `gorm:"type:varchar(500);comment:备注" json:"description"`
	Status           int           `gorm:"type:tinyint;default:1;comment:状态 1:在线 0:离线 -1:未知" json:"status"`
	LastSeen         *time.Time    `gorm:"column:last_seen;comment:最后连接时间" json:"lastSeen,omitempty"`
	OS               string        `gorm:"type:varchar(100);comment:操作系统" json:"os"`
	Kernel           string        `gorm:"type:varchar(100);comment:内核版本" json:"kernel"`
	Arch             string        `gorm:"type:varchar(50);comment:架构" json:"arch"`
	// 扩展信息字段（JSON存储）
	CPUInfo          string        `gorm:"type:text;comment:CPU信息JSON" json:"-"`
	CPUCores         int           `gorm:"type:int;comment:CPU核心数" json:"cpuCores"`
	CPUUsage         float64       `gorm:"type:float;comment:CPU使用率" json:"cpuUsage"`
	MemoryTotal      uint64        `gorm:"type:bigint;comment:内存总容量(字节)" json:"memoryTotal"`
	MemoryUsed       uint64        `gorm:"type:bigint;comment:已用内存(字节)" json:"memoryUsed"`
	MemoryUsage      float64       `gorm:"type:float;comment:内存使用率" json:"memoryUsage"`
	DiskTotal        uint64        `gorm:"type:bigint;comment:磁盘总容量(字节)" json:"diskTotal"`
	DiskUsed         uint64        `gorm:"type:bigint;comment:已用磁盘(字节)" json:"diskUsed"`
	DiskUsage        float64       `gorm:"type:float;comment:磁盘使用率" json:"diskUsage"`
	Uptime           string        `gorm:"type:varchar(100);comment:运行时间" json:"uptime"`
	Hostname         string        `gorm:"type:varchar(100);comment:主机名" json:"hostname"`
}

// HostRequest 主机请求
type HostRequest struct {
	ID            uint   `json:"id"`
	Name          string `json:"name" binding:"required,min=2,max=100"`
	GroupID       uint   `json:"groupId"`
	Type          string `json:"type" binding:"required,oneof=self cloud"`
	CloudProvider string `json:"cloudProvider,omitempty"`
	CloudInstanceID string `json:"cloudInstanceId,omitempty"`
	CloudAccountID  uint   `json:"cloudAccountId,omitempty"`
	SSHUser       string `json:"sshUser" binding:"required"`
	IP            string `json:"ip" binding:"required,ip"`
	Port          int    `json:"port" binding:"required,min=1,max=65535"`
	CredentialID  uint   `json:"credentialId"`
	Tags          string `json:"tags"`
	Description   string `json:"description"`
}

// HostInfoVO 主机信息VO
type HostInfoVO struct {
	ID               uint           `json:"id"`
	Name             string         `json:"name"`
	GroupName        string         `json:"groupName"`
	GroupID          uint           `json:"groupId"`
	Type             string         `json:"type"`
	TypeText         string         `json:"typeText"`
	CloudProvider    string         `json:"cloudProvider,omitempty"`
	CloudProviderText string        `json:"cloudProviderText,omitempty"`
	CloudInstanceID  string         `json:"cloudInstanceId,omitempty"`
	SSHUser          string         `json:"sshUser"`
	IP               string         `json:"ip"`
	Port             int            `json:"port"`
	CredentialID     uint           `json:"credentialId"`
	Credential       *CredentialVO  `json:"credential,omitempty"`
	Tags             []string       `json:"tags"`
	Description      string         `json:"description"`
	Status           int            `json:"status"`
	StatusText       string         `json:"statusText"`
	LastSeen         string         `json:"lastSeen,omitempty"`
	OS               string         `json:"os"`
	Kernel           string         `json:"kernel"`
	Arch             string         `json:"arch"`
	CreateTime       string         `json:"createTime"`
	UpdateTime       string         `json:"updateTime"`
	// 扩展信息
	CPUCores         int     `json:"cpuCores"`
	CPUUsage         float64 `json:"cpuUsage"`
	MemoryTotal      uint64  `json:"memoryTotal"`
	MemoryUsed       uint64  `json:"memoryUsed"`
	MemoryUsage      float64 `json:"memoryUsage"`
	DiskTotal        uint64  `json:"diskTotal"`
	DiskUsed         uint64  `json:"diskUsed"`
	DiskUsage        float64 `json:"diskUsage"`
	Uptime           string  `json:"uptime"`
	Hostname         string  `json:"hostname"`
}

// HostListVO 主机列表VO（用于分组下的主机列表）
type HostListVO struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	IP      string `json:"ip"`
	Status  int    `json:"status"`
	Port    int    `json:"port"`
	SSHUser string `json:"sshUser"`
	OS      string `json:"os"`
}

// ToModel 转换为模型
func (req *HostRequest) ToModel() *Host {
	return &Host{
		Name:            req.Name,
		GroupID:         req.GroupID,
		Type:            req.Type,
		CloudProvider:   req.CloudProvider,
		CloudInstanceID: req.CloudInstanceID,
		CloudAccountID:  req.CloudAccountID,
		SSHUser:         req.SSHUser,
		IP:              req.IP,
		Port:            req.Port,
		CredentialID:    req.CredentialID,
		Tags:            req.Tags,
		Description:     req.Description,
		Status:          -1, // 初始状态未知
	}
}

// Credential 凭证模型
type Credential struct {
	ID          uint   `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`
	Name        string `gorm:"type:varchar(100);not null;comment:凭证名称" json:"name"`
	Type        string `gorm:"type:varchar(20);not null;comment:认证方式 password/key" json:"type"`
	Username    string `gorm:"type:varchar(100);comment:用户名" json:"username"`
	Password    string `gorm:"type:varchar(500);comment:密码(加密)" json:"password,omitempty"`
	PrivateKey  string `gorm:"type:text;comment:私钥(加密)" json:"privateKey,omitempty"`
	Passphrase  string `gorm:"type:varchar(500);comment:私钥密码(加密)" json:"passphrase,omitempty"`
	Description string `gorm:"type:varchar(500);comment:备注" json:"description"`
}

// CredentialRequest 凭证请求
type CredentialRequest struct {
	ID          uint   `json:"id"`
	Name        string `json:"name" binding:"required,min=2,max=100"`
	Type        string `json:"type" binding:"required,oneof=password key"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	PrivateKey  string `json:"privateKey"`
	Passphrase  string `json:"passphrase"`
	Description string `json:"description"`
}

// CredentialVO 凭证VO
type CredentialVO struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	TypeText    string `json:"typeText"`
	Username    string `json:"username"`
	Description string `json:"description"`
	CreateTime  string `json:"createTime"`
	HostCount   int64  `json:"hostCount"` // 使用该凭证的主机数量
}

// ToModel 转换为模型
func (req *CredentialRequest) ToModel() *Credential {
	return &Credential{
		Name:        req.Name,
		Type:        req.Type,
		Username:    req.Username,
		Password:    req.Password,
		PrivateKey:  req.PrivateKey,
		Passphrase:  req.Passphrase,
		Description: req.Description,
	}
}

// CloudAccount 云平台账号模型
type CloudAccount struct {
	gorm.Model
	Name        string `gorm:"type:varchar(100);not null;comment:账号名称" json:"name"`
	Provider    string `gorm:"type:varchar(50);not null;comment:云厂商 aliyun/tencent/aws/huawei" json:"provider"`
	AccessKey   string `gorm:"type:varchar(200);not null;comment:AccessKey" json:"-"`
	SecretKey   string `gorm:"type:varchar(500);not null;comment:SecretKey" json:"-"`
	Region      string `gorm:"type:varchar(100);comment:默认区域" json:"region"`
	Description string `gorm:"type:varchar(500);comment:备注" json:"description"`
	Status      int    `gorm:"type:tinyint;default:1;comment:状态 1:启用 0:禁用" json:"status"`
}

// CloudAccountRequest 云平台账号请求
type CloudAccountRequest struct {
	ID          uint   `json:"id"`
	Name        string `json:"name" binding:"required,min=2,max=100"`
	Provider    string `json:"provider" binding:"required,oneof=aliyun tencent aws huawei jdcloud"`
	AccessKey   string `json:"accessKey"`
	SecretKey   string `json:"secretKey"`
	Region      string `json:"region"`
	Description string `json:"description"`
	Status      int    `json:"status"`
}

// CloudAccountVO 云平台账号VO
type CloudAccountVO struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	Provider     string `json:"provider"`
	ProviderText string `json:"providerText"`
	Region       string `json:"region"`
	Description  string `json:"description"`
	Status       int    `json:"status"`
	CreateTime   string `json:"createTime"`
}

// ToModel 转换为模型
func (req *CloudAccountRequest) ToModel() *CloudAccount {
	return &CloudAccount{
		Name:        req.Name,
		Provider:    req.Provider,
		AccessKey:   req.AccessKey,
		SecretKey:   req.SecretKey,
		Region:      req.Region,
		Description: req.Description,
		Status:      req.Status,
	}
}

// CloudImportRequest 云主机导入请求
type CloudImportRequest struct {
	AccountID   uint     `json:"accountId" binding:"required"`
	AccountName string   `json:"accountName"`
	Region      string   `json:"region"`
	GroupID     uint     `json:"groupId"`
	InstanceIDs []string `json:"instanceIds"` // 要导入的实例ID列表
}

// CloudInstanceVO 云主机实例VO（用于前端展示）
type CloudInstanceVO struct {
	InstanceID string `json:"instanceId"`
	Name       string `json:"name"`
	PublicIP   string `json:"publicIp"`
	PrivateIP  string `json:"privateIp"`
	OS         string `json:"os"`
	Status     string `json:"status"`
}

// CloudRegionVO 云区域VO
type CloudRegionVO struct {
	Value string `json:"value"`
	Label string `json:"label"`
}
