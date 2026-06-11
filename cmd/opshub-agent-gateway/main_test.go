package main

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestAllowedAgentRoute(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		path    string
		allowed bool
	}{
		{name: "install script", method: http.MethodGet, path: "/api/v1/public/agents/install.sh", allowed: true},
		{name: "binary", method: http.MethodGet, path: "/api/v1/public/agents/binaries/opshub-agent-linux-amd64", allowed: true},
		{name: "register", method: http.MethodPost, path: "/api/v1/public/agents/register", allowed: true},
		{name: "heartbeat", method: http.MethodPost, path: "/api/v1/public/agents/heartbeat", allowed: true},
		{name: "metrics", method: http.MethodPost, path: "/api/v1/public/agents/metrics", allowed: true},
		{name: "wrong method", method: http.MethodGet, path: "/api/v1/public/agents/metrics", allowed: false},
		{name: "non agent route", method: http.MethodGet, path: "/api/v1/users", allowed: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := allowedAgentRoute(tt.method, tt.path); got != tt.allowed {
				t.Fatalf("allowedAgentRoute(%q, %q) = %v, want %v", tt.method, tt.path, got, tt.allowed)
			}
		})
	}
}

func TestGatewayProxiesOnlyAllowedRoutes(t *testing.T) {
	upstreamURL, err := url.Parse("http://opshub.internal:9876")
	if err != nil {
		t.Fatal(err)
	}
	gw := newGateway(upstreamURL, nil)

	proxyCalled := false
	gw.proxy.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		proxyCalled = true
		if r.Header.Get("X-OpsHub-Agent-Gateway") != "true" {
			t.Fatalf("missing gateway header")
		}
		if r.URL.Host != upstreamURL.Host {
			t.Fatalf("proxied host = %q, want %q", r.URL.Host, upstreamURL.Host)
		}
		return &http.Response{
			StatusCode: http.StatusAccepted,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("accepted")),
			Request:    r,
		}, nil
	})

	allowedReq := httptest.NewRequest(http.MethodPost, "/api/v1/public/agents/metrics", strings.NewReader(`{}`))
	allowedResp := httptest.NewRecorder()
	gw.serveHTTP(allowedResp, allowedReq)
	if allowedResp.Code != http.StatusAccepted {
		t.Fatalf("allowed route status = %d, want %d", allowedResp.Code, http.StatusAccepted)
	}
	if !proxyCalled {
		t.Fatalf("expected allowed route to be proxied")
	}

	proxyCalled = false
	blockedReq := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	blockedResp := httptest.NewRecorder()
	gw.serveHTTP(blockedResp, blockedReq)
	if blockedResp.Code != http.StatusNotFound {
		t.Fatalf("blocked route status = %d, want %d", blockedResp.Code, http.StatusNotFound)
	}
	if proxyCalled {
		t.Fatalf("blocked route should not be proxied")
	}
}

func TestGatewayCIDRAllowlist(t *testing.T) {
	_, cidr, err := net.ParseCIDR("203.0.113.0/24")
	if err != nil {
		t.Fatal(err)
	}
	gw := &gateway{cidrs: []*net.IPNet{cidr}}

	allowedReq := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	allowedReq.RemoteAddr = "203.0.113.10:12345"
	if !gw.sourceAllowed(allowedReq) {
		t.Fatalf("expected source to be allowed")
	}

	blockedReq := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	blockedReq.RemoteAddr = "198.51.100.10:12345"
	if gw.sourceAllowed(blockedReq) {
		t.Fatalf("expected source to be blocked")
	}
}

func TestBodyLimitRejectsLargeRequests(t *testing.T) {
	gw := &gateway{}
	nextCalled := false
	handler := gw.withBodyLimit(4, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/public/agents/metrics", strings.NewReader("12345"))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusRequestEntityTooLarge)
	}
	if nextCalled {
		t.Fatalf("next handler should not be called for oversized body")
	}
}
