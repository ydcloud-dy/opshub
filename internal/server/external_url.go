package server

import (
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/conf"
)

func captureExternalURL() gin.HandlerFunc {
	return func(c *gin.Context) {
		if baseURL := requestExternalBaseURL(c.Request); baseURL != "" {
			conf.RecordObservedFrontendURL(baseURL)
		}
		c.Next()
	}
}

func requestExternalBaseURL(r *http.Request) string {
	if r == nil {
		return ""
	}
	host := firstHeaderValue(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = strings.TrimSpace(r.Host)
	}
	if !isPublicFrontendHost(host) {
		return ""
	}
	proto := firstHeaderValue(r.Header.Get("X-Forwarded-Proto"))
	if proto == "" {
		proto = firstHeaderValue(r.Header.Get("X-Forwarded-Scheme"))
	}
	if proto == "" && r.TLS != nil {
		proto = "https"
	}
	if proto == "" {
		proto = "http"
	}
	proto = strings.ToLower(strings.TrimSpace(proto))
	if proto != "http" && proto != "https" {
		proto = "http"
	}
	return proto + "://" + strings.TrimRight(host, "/")
}

func firstHeaderValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.Split(value, ",")
	return strings.TrimSpace(parts[0])
}

func isPublicFrontendHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return false
	}
	withoutPort := host
	if h, _, err := net.SplitHostPort(host); err == nil {
		withoutPort = h
	}
	withoutPort = strings.Trim(withoutPort, "[]")
	if withoutPort == "" ||
		withoutPort == "localhost" ||
		withoutPort == "127.0.0.1" ||
		withoutPort == "::1" ||
		withoutPort == "backend" ||
		withoutPort == "opshub-backend" ||
		strings.HasSuffix(withoutPort, ".svc") ||
		strings.Contains(withoutPort, ".svc.") {
		return false
	}
	if strings.HasSuffix(host, ":9876") {
		return false
	}
	return true
}
