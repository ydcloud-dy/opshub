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

package service

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/oschwald/geoip2-golang"
	"github.com/phuslu/iploc"
)

// GeoInfo 地理位置信息
type GeoInfo struct {
	Country  string `json:"country"`
	Province string `json:"province"`
	City     string `json:"city"`
	ISP      string `json:"isp"`
}

// GeolocationService 地理位置服务
type GeolocationService struct {
	cache   *lru.Cache[string, *GeoInfo]
	cityDB  *geoip2.Reader // GeoLite2-City.mmdb
	asnDB   *geoip2.Reader // GeoLite2-ASN.mmdb (可选，用于 ISP)
	hasMMDB bool
}

// mmdb 文件搜索路径
var mmdbSearchPaths = []string{
	"plugins/nginx/data",
	"data",
	"./data",
	"/usr/share/GeoIP",
	"/var/lib/GeoIP",
}

// NewGeolocationService 创建地理位置服务
func NewGeolocationService() *GeolocationService {
	cache, _ := lru.New[string, *GeoInfo](10000)

	svc := &GeolocationService{
		cache:   cache,
		hasMMDB: false,
	}

	// 尝试加载 mmdb 文件
	svc.loadMMDB()

	return svc
}

// loadMMDB 加载 mmdb 数据库文件
func (s *GeolocationService) loadMMDB() {
	// 查找 City 数据库
	cityFiles := []string{"GeoLite2-City.mmdb", "GeoIP2-City.mmdb", "city.mmdb"}
	for _, searchPath := range mmdbSearchPaths {
		for _, filename := range cityFiles {
			path := filepath.Join(searchPath, filename)
			// 检查文件是否存在
			if _, err := os.Stat(path); err != nil {
				continue // 文件不存在，继续查找
			}
			// 尝试打开数据库
			db, err := geoip2.Open(path)
			if err == nil {
				s.cityDB = db
				s.hasMMDB = true
				fmt.Printf("[Nginx] Loaded GeoIP City database: %s\n", path)
				break
			} else {
				fmt.Printf("[Nginx] Failed to open GeoIP City database %s: %v\n", path, err)
			}
		}
		if s.cityDB != nil {
			break
		}
	}

	// 查找 ASN 数据库 (可选)
	asnFiles := []string{"GeoLite2-ASN.mmdb", "GeoIP2-ASN.mmdb", "asn.mmdb"}
	for _, searchPath := range mmdbSearchPaths {
		for _, filename := range asnFiles {
			path := filepath.Join(searchPath, filename)
			if _, err := os.Stat(path); err != nil {
				continue
			}
			db, err := geoip2.Open(path)
			if err == nil {
				s.asnDB = db
				fmt.Printf("[Nginx] Loaded GeoIP ASN database: %s\n", path)
				break
			}
		}
		if s.asnDB != nil {
			break
		}
	}

	if !s.hasMMDB {
		fmt.Println("[Nginx] Warning: No mmdb database found, geolocation will be limited (country only)")
		fmt.Println("[Nginx] Searched paths:", strings.Join(mmdbSearchPaths, ", "))
		fmt.Println("[Nginx] Please download GeoLite2-City.mmdb from https://dev.maxmind.com/geoip/geolite2-free-geolocation-data")
	}
}

// Close 关闭数据库连接
func (s *GeolocationService) Close() {
	if s.cityDB != nil {
		s.cityDB.Close()
	}
	if s.asnDB != nil {
		s.asnDB.Close()
	}
}

// Lookup 查询 IP 地理位置
func (s *GeolocationService) Lookup(ip string) (*GeoInfo, error) {
	// 检查缓存
	if info, ok := s.cache.Get(ip); ok {
		return info, nil
	}

	// 检查是否为内网 IP
	if isPrivateIP(ip) {
		info := &GeoInfo{
			Country:  "内网",
			Province: "局域网",
			City:     "局域网",
			ISP:      "内网",
		}
		s.cache.Add(ip, info)
		return info, nil
	}

	var info *GeoInfo

	// 优先使用 mmdb 数据库
	if s.hasMMDB && s.cityDB != nil {
		info = s.lookupMMDB(ip)
	}

	// 如果 mmdb 没有结果，使用 phuslu/iploc 作为后备
	if info == nil || info.Country == "" {
		info = s.lookupIploc(ip)
	}

	if info != nil && info.Country != "" {
		s.cache.Add(ip, info)
		return info, nil
	}

	// 返回默认值
	info = &GeoInfo{
		Country:  "未知",
		Province: "",
		City:     "",
		ISP:      "",
	}
	s.cache.Add(ip, info)
	return info, nil
}

// lookupMMDB 使用 mmdb 数据库查询
func (s *GeolocationService) lookupMMDB(ip string) *GeoInfo {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return nil
	}

	info := &GeoInfo{}

	// 查询城市信息
	if s.cityDB != nil {
		record, err := s.cityDB.City(parsedIP)
		if err == nil {
			// 国家 - 优先使用中文
			if name, ok := record.Country.Names["zh-CN"]; ok {
				info.Country = name
			} else if name, ok := record.Country.Names["en"]; ok {
				info.Country = name
			}

			// 省份/地区 - 优先使用中文
			if len(record.Subdivisions) > 0 {
				if name, ok := record.Subdivisions[0].Names["zh-CN"]; ok {
					info.Province = name
				} else if name, ok := record.Subdivisions[0].Names["en"]; ok {
					info.Province = name
				}
			}

			// 城市 - 优先使用中文
			if name, ok := record.City.Names["zh-CN"]; ok {
				info.City = name
			} else if name, ok := record.City.Names["en"]; ok {
				info.City = name
			}
		}
	}

	// 查询 ASN/ISP 信息
	if s.asnDB != nil {
		record, err := s.asnDB.ASN(parsedIP)
		if err == nil && record.AutonomousSystemOrganization != "" {
			info.ISP = simplifyISP(record.AutonomousSystemOrganization)
		}
	}

	return info
}

// lookupIploc 使用 phuslu/iploc 查询 (后备方案)
func (s *GeolocationService) lookupIploc(ip string) *GeoInfo {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return nil
	}

	countryCode := iploc.Country(parsedIP)
	if countryCode == "" {
		return nil
	}

	return &GeoInfo{
		Country: countryCodeToName(countryCode),
	}
}

// simplifyISP 简化 ISP 名称
func simplifyISP(org string) string {
	// 常见 ISP 简化
	ispMap := map[string]string{
		"China Telecom": "中国电信",
		"China Unicom":  "中国联通",
		"China Mobile":  "中国移动",
		"CHINA TELECOM": "中国电信",
		"CHINA UNICOM":  "中国联通",
		"CHINA MOBILE":  "中国移动",
		"Alibaba":       "阿里云",
		"Tencent":       "腾讯云",
		"TENCENT":       "腾讯云",
		"Huawei":        "华为云",
		"Amazon":        "AWS",
		"Google":        "Google Cloud",
		"Microsoft":     "Azure",
		"Cloudflare":    "Cloudflare",
	}

	for key, value := range ispMap {
		if strings.Contains(org, key) {
			return value
		}
	}

	// 截断过长的名称
	if len(org) > 30 {
		return org[:30] + "..."
	}
	return org
}

// countryCodeToName 将国家代码转换为中文国家名
func countryCodeToName(code string) string {
	countryNames := map[string]string{
		"CN": "中国",
		"US": "美国",
		"JP": "日本",
		"KR": "韩国",
		"DE": "德国",
		"FR": "法国",
		"GB": "英国",
		"RU": "俄罗斯",
		"CA": "加拿大",
		"AU": "澳大利亚",
		"IN": "印度",
		"BR": "巴西",
		"SG": "新加坡",
		"HK": "香港",
		"TW": "台湾",
		"MO": "澳门",
		"NL": "荷兰",
		"IT": "意大利",
		"ES": "西班牙",
		"SE": "瑞典",
		"CH": "瑞士",
		"NO": "挪威",
		"FI": "芬兰",
		"DK": "丹麦",
		"PL": "波兰",
		"AT": "奥地利",
		"BE": "比利时",
		"IE": "爱尔兰",
		"NZ": "新西兰",
		"MX": "墨西哥",
		"AR": "阿根廷",
		"TH": "泰国",
		"VN": "越南",
		"MY": "马来西亚",
		"ID": "印度尼西亚",
		"PH": "菲律宾",
		"ZA": "南非",
		"EG": "埃及",
		"AE": "阿联酋",
		"SA": "沙特阿拉伯",
		"IL": "以色列",
		"TR": "土耳其",
		"UA": "乌克兰",
		"CZ": "捷克",
		"RO": "罗马尼亚",
		"HU": "匈牙利",
		"GR": "希腊",
		"PT": "葡萄牙",
	}

	if name, ok := countryNames[code]; ok {
		return name
	}
	return code
}

// LookupBatch 批量查询 IP 地理位置
func (s *GeolocationService) LookupBatch(ips []string) map[string]*GeoInfo {
	results := make(map[string]*GeoInfo)

	for _, ip := range ips {
		info, _ := s.Lookup(ip)
		results[ip] = info
	}

	return results
}

// isPrivateIP 判断是否为内网 IP
func isPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// IPv4 内网地址段
	privateBlocks := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
	}

	for _, cidr := range privateBlocks {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if block.Contains(ip) {
			return true
		}
	}

	// IPv6 本地地址
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsPrivate() {
		return true
	}

	return false
}

// ClearCache 清除缓存
func (s *GeolocationService) ClearCache() {
	s.cache.Purge()
}

// CacheSize 获取缓存大小
func (s *GeolocationService) CacheSize() int {
	return s.cache.Len()
}

// HasMMDB 检查是否有 mmdb 数据库
func (s *GeolocationService) HasMMDB() bool {
	return s.hasMMDB
}
