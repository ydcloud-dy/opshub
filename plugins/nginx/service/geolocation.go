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
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
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
	cache       *lru.Cache[string, *GeoInfo]
	httpClient  *http.Client
	rateLimiter *time.Ticker
	rateMutex   sync.Mutex
}

// NewGeolocationService 创建地理位置服务
func NewGeolocationService() *GeolocationService {
	cache, _ := lru.New[string, *GeoInfo](10000)

	return &GeolocationService{
		cache: cache,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		rateLimiter: time.NewTicker(100 * time.Millisecond), // 10 req/sec for ip-api.com free tier
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

	// 尝试在线查询 (ip-api.com)
	info, err := s.lookupOnline(ip)
	if err == nil && info != nil {
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

// LookupBatch 批量查询 IP 地理位置
func (s *GeolocationService) LookupBatch(ips []string) map[string]*GeoInfo {
	results := make(map[string]*GeoInfo)

	for _, ip := range ips {
		info, _ := s.Lookup(ip)
		results[ip] = info
	}

	return results
}

// ipAPIResponse ip-api.com 响应
type ipAPIResponse struct {
	Status      string `json:"status"`
	Country     string `json:"country"`
	CountryCode string `json:"countryCode"`
	Region      string `json:"region"`
	RegionName  string `json:"regionName"`
	City        string `json:"city"`
	ISP         string `json:"isp"`
	Message     string `json:"message"`
}

// lookupOnline 在线查询 IP 地理位置
func (s *GeolocationService) lookupOnline(ip string) (*GeoInfo, error) {
	// 速率限制
	s.rateMutex.Lock()
	<-s.rateLimiter.C
	s.rateMutex.Unlock()

	url := fmt.Sprintf("http://ip-api.com/json/%s?fields=status,message,country,countryCode,region,regionName,city,isp&lang=zh-CN", ip)

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result ipAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("ip-api error: %s", result.Message)
	}

	info := &GeoInfo{
		Country:  result.Country,
		Province: result.RegionName,
		City:     result.City,
		ISP:      result.ISP,
	}

	// 中国地址转换
	if result.CountryCode == "CN" {
		info.Country = "中国"
	}

	return info, nil
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

// ParseIP2RegionResult 解析 ip2region 结果格式
// 格式: 国家|区域|省份|城市|ISP
func ParseIP2RegionResult(result string) *GeoInfo {
	parts := strings.Split(result, "|")

	info := &GeoInfo{}

	if len(parts) >= 1 && parts[0] != "0" {
		info.Country = parts[0]
	}
	// parts[1] is area/region, usually not needed
	if len(parts) >= 3 && parts[2] != "0" {
		info.Province = parts[2]
	}
	if len(parts) >= 4 && parts[3] != "0" {
		info.City = parts[3]
	}
	if len(parts) >= 5 && parts[4] != "0" {
		info.ISP = parts[4]
	}

	return info
}

// ClearCache 清除缓存
func (s *GeolocationService) ClearCache() {
	s.cache.Purge()
}

// CacheSize 获取缓存大小
func (s *GeolocationService) CacheSize() int {
	return s.cache.Len()
}
