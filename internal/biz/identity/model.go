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

package identity

import (
	"time"

	"gorm.io/gorm"
)

// IdentitySource 身份源表
type IdentitySource struct {
	gorm.Model
	Name           string `gorm:"type:varchar(50);not null;comment:身份源名称" json:"name"`
	Type           string `gorm:"type:varchar(30);not null;comment:类型(wechat/dingtalk/feishu/qq/github等)" json:"type"`
	Icon           string `gorm:"type:varchar(255);comment:图标URL" json:"icon"`
	Config         string `gorm:"type:text;comment:配置JSON" json:"config"`
	UserMapping    string `gorm:"type:text;comment:用户属性映射" json:"userMapping"`
	AutoCreateUser bool   `gorm:"default:false;comment:自动创建用户" json:"autoCreateUser"`
	DefaultRoleID  uint   `gorm:"default:0;comment:默认角色ID" json:"defaultRoleId"`
	Enabled        bool   `gorm:"default:true;comment:是否启用" json:"enabled"`
	Sort           int    `gorm:"default:0;comment:排序" json:"sort"`
}

// SSOApplication SSO应用表
type SSOApplication struct {
	gorm.Model
	Name        string `gorm:"type:varchar(100);not null;comment:应用名称" json:"name"`
	Code        string `gorm:"type:varchar(50);uniqueIndex;comment:应用编码" json:"code"`
	Icon        string `gorm:"type:varchar(255);comment:图标URL" json:"icon"`
	Description string `gorm:"type:varchar(500);comment:应用描述" json:"description"`
	Category    string `gorm:"type:varchar(50);comment:分类(cicd/code/monitor/registry)" json:"category"`
	URL         string `gorm:"type:varchar(500);not null;comment:应用URL" json:"url"`
	SSOType     string `gorm:"type:varchar(30);comment:SSO类型(oauth2/saml/form/token)" json:"ssoType"`
	SSOConfig   string `gorm:"type:text;comment:SSO配置JSON" json:"ssoConfig"`
	Enabled     bool   `gorm:"default:true;comment:是否启用" json:"enabled"`
	Sort        int    `gorm:"default:0;comment:排序" json:"sort"`
}

// UserCredential 用户凭证表
type UserCredential struct {
	gorm.Model
	UserID    uint   `gorm:"index;not null;comment:用户ID" json:"userId"`
	AppID     uint   `gorm:"index;not null;comment:应用ID" json:"appId"`
	Username  string `gorm:"type:varchar(100);comment:应用账号" json:"username"`
	Password  string `gorm:"type:varchar(500);comment:应用密码(加密存储)" json:"-"`
	ExtraData string `gorm:"type:text;comment:额外数据JSON" json:"extraData"`
}

// AppPermission 应用权限表
type AppPermission struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	AppID       uint      `gorm:"index;not null;comment:应用ID" json:"appId"`
	SubjectType string    `gorm:"type:varchar(20);not null;comment:主体类型(user/role/dept)" json:"subjectType"`
	SubjectID   uint      `gorm:"index;not null;comment:主体ID" json:"subjectId"`
	Permission  string    `gorm:"type:varchar(20);default:access;comment:权限类型" json:"permission"`
	CreatedAt   time.Time `json:"createdAt"`
}

// UserOAuthBinding 用户第三方绑定表
type UserOAuthBinding struct {
	gorm.Model
	UserID     uint   `gorm:"index;not null;comment:用户ID" json:"userId"`
	SourceID   uint   `gorm:"index;not null;comment:身份源ID" json:"sourceId"`
	SourceType string `gorm:"type:varchar(30);not null;comment:身份源类型" json:"sourceType"`
	OpenID     string `gorm:"type:varchar(255);index;comment:OpenID" json:"openId"`
	UnionID    string `gorm:"type:varchar(255);comment:UnionID" json:"unionId"`
	Nickname   string `gorm:"type:varchar(100);comment:昵称" json:"nickname"`
	Avatar     string `gorm:"type:varchar(500);comment:头像URL" json:"avatar"`
	ExtraInfo  string `gorm:"type:text;comment:额外信息JSON" json:"extraInfo"`
}

// AuthLog 认证日志表
type AuthLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"index;comment:用户ID" json:"userId"`
	Username   string    `gorm:"type:varchar(50);comment:用户名" json:"username"`
	Action     string    `gorm:"type:varchar(30);comment:动作(login/logout/access_app)" json:"action"`
	AppID      uint      `json:"appId"`
	AppName    string    `gorm:"type:varchar(100);comment:应用名称" json:"appName"`
	LoginType  string    `gorm:"type:varchar(30);comment:登录类型" json:"loginType"`
	IP         string    `gorm:"type:varchar(50);comment:IP地址" json:"ip"`
	Location   string    `gorm:"type:varchar(100);comment:地理位置" json:"location"`
	UserAgent  string    `gorm:"type:varchar(500);comment:UserAgent" json:"userAgent"`
	Result     string    `gorm:"type:varchar(20);comment:结果(success/failed)" json:"result"`
	FailReason string    `gorm:"type:varchar(255);comment:失败原因" json:"failReason"`
	CreatedAt  time.Time `gorm:"index;comment:创建时间" json:"createdAt"`
}

// UserFavoriteApp 用户收藏应用表
type UserFavoriteApp struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null;comment:用户ID" json:"userId"`
	AppID     uint      `gorm:"index;not null;comment:应用ID" json:"appId"`
	CreatedAt time.Time `json:"createdAt"`
}

// TableName 指定表名
func (IdentitySource) TableName() string {
	return "identity_sources"
}

func (SSOApplication) TableName() string {
	return "sso_applications"
}

func (UserCredential) TableName() string {
	return "user_credentials"
}

func (AppPermission) TableName() string {
	return "app_permissions"
}

func (UserOAuthBinding) TableName() string {
	return "user_oauth_bindings"
}

func (AuthLog) TableName() string {
	return "auth_logs"
}

func (UserFavoriteApp) TableName() string {
	return "user_favorite_apps"
}

// IdentitySourceConfig 身份源配置
type IdentitySourceConfig struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	RedirectURI  string `json:"redirectUri"`
	Scopes       string `json:"scopes"`
	AuthURL      string `json:"authUrl"`
	TokenURL     string `json:"tokenUrl"`
	UserInfoURL  string `json:"userInfoUrl"`
}

// SSOConfig SSO配置
type SSOConfig struct {
	// OAuth2配置
	ClientID     string `json:"clientId,omitempty"`
	ClientSecret string `json:"clientSecret,omitempty"`
	RedirectURI  string `json:"redirectUri,omitempty"`
	Scopes       string `json:"scopes,omitempty"`

	// 表单代填配置
	LoginURL         string `json:"loginUrl,omitempty"`
	UsernameField    string `json:"usernameField,omitempty"`
	PasswordField    string `json:"passwordField,omitempty"`
	SubmitButton     string `json:"submitButton,omitempty"`
	AdditionalFields string `json:"additionalFields,omitempty"`

	// Token配置
	TokenHeader string `json:"tokenHeader,omitempty"`
	TokenPrefix string `json:"tokenPrefix,omitempty"`
}

// AppTemplate 预置应用模板
type AppTemplate struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Icon        string `json:"icon"`
	Category    string `json:"category"`
	Description string `json:"description"`
	SSOType     string `json:"ssoType"`
	URLTemplate string `json:"urlTemplate"`
}

// PortalApp 门户应用视图
type PortalApp struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
	Category    string `json:"category"`
	URL         string `json:"url"`
	IsFavorite  bool   `json:"isFavorite"`
}

// AuthLogStats 认证日志统计
type AuthLogStats struct {
	TotalLogins    int64         `json:"totalLogins"`
	TodayLogins    int64         `json:"todayLogins"`
	FailedLogins   int64         `json:"failedLogins"`
	UniqueUsers    int64         `json:"uniqueUsers"`
	AppAccessCount int64         `json:"appAccessCount"`
	LoginTrend     []TrendPoint  `json:"loginTrend"`
	TopApps        []TopAppStat  `json:"topApps"`
	TopUsers       []TopUserStat `json:"topUsers"`
}

// TrendPoint 趋势数据点
type TrendPoint struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// TopAppStat 应用访问排行
type TopAppStat struct {
	AppID   uint   `json:"appId"`
	AppName string `json:"appName"`
	Count   int64  `json:"count"`
}

// TopUserStat 用户活跃排行
type TopUserStat struct {
	UserID   uint   `json:"userId"`
	Username string `json:"username"`
	Count    int64  `json:"count"`
}
