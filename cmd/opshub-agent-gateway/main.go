package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const defaultMaxBodyBytes = 10 << 20

type config struct {
	ListenAddr   string
	UpstreamURL  string
	TLSCert      string
	TLSKey       string
	AllowCIDRs   string
	MaxBodyBytes int64
}

type gateway struct {
	upstream *url.URL
	proxy    *httputil.ReverseProxy
	cidrs    []*net.IPNet
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func main() {
	cfg := loadConfig()
	if cfg.UpstreamURL == "" {
		log.Fatal("missing upstream OpsHub URL, set --upstream or OPSHUB_UPSTREAM_URL")
	}

	upstream, err := parseUpstream(cfg.UpstreamURL)
	if err != nil {
		log.Fatalf("invalid upstream URL: %v", err)
	}
	cidrs, err := parseCIDRs(cfg.AllowCIDRs)
	if err != nil {
		log.Fatalf("invalid allow CIDRs: %v", err)
	}

	gw := newGateway(upstream, cidrs)
	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           gw.withLogging(gw.withBodyLimit(cfg.MaxBodyBytes, http.HandlerFunc(gw.serveHTTP))),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("OpsHub Agent Gateway listening on %s, upstream=%s", cfg.ListenAddr, upstream.String())
		var err error
		if cfg.TLSCert != "" && cfg.TLSKey != "" {
			err = server.ListenAndServeTLS(cfg.TLSCert, cfg.TLSKey)
		} else {
			err = server.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("agent gateway stopped: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown failed: %v", err)
	}
}

func loadConfig() config {
	var cfg config
	flag.StringVar(&cfg.ListenAddr, "listen", getenv("OPSHUB_GATEWAY_LISTEN", ":9877"), "listen address")
	flag.StringVar(&cfg.UpstreamURL, "upstream", getenv("OPSHUB_UPSTREAM_URL", ""), "internal OpsHub base URL, e.g. http://10.0.0.10:9876")
	flag.StringVar(&cfg.TLSCert, "tls-cert", getenv("OPSHUB_GATEWAY_TLS_CERT", ""), "TLS certificate file")
	flag.StringVar(&cfg.TLSKey, "tls-key", getenv("OPSHUB_GATEWAY_TLS_KEY", ""), "TLS key file")
	flag.StringVar(&cfg.AllowCIDRs, "allow-cidrs", getenv("OPSHUB_GATEWAY_ALLOW_CIDRS", ""), "optional comma-separated source CIDRs")

	maxBodyMB := flag.Int("max-body-mb", getenvInt("OPSHUB_GATEWAY_MAX_BODY_MB", 10), "max POST body size in MiB")
	flag.Parse()

	cfg.MaxBodyBytes = int64(*maxBodyMB) << 20
	if cfg.MaxBodyBytes <= 0 {
		cfg.MaxBodyBytes = defaultMaxBodyBytes
	}
	return cfg
}

func newGateway(upstream *url.URL, cidrs []*net.IPNet) *gateway {
	proxy := httputil.NewSingleHostReverseProxy(upstream)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		externalHost := req.Host
		originalDirector(req)
		req.Host = upstream.Host
		req.Header.Set("X-Forwarded-Host", externalHost)
		req.Header.Set("X-Forwarded-Proto", forwardedProto(req))
		req.Header.Set("X-OpsHub-Agent-Gateway", "true")
	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Set("X-OpsHub-Agent-Gateway", "true")
		return nil
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("proxy error method=%s path=%s err=%v", r.Method, r.URL.Path, err)
		http.Error(w, "OpsHub upstream unavailable", http.StatusBadGateway)
	}
	return &gateway{upstream: upstream, proxy: proxy, cidrs: cidrs}
}

func (g *gateway) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
		return
	}

	if !g.sourceAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if !allowedAgentRoute(r.Method, r.URL.Path) {
		http.NotFound(w, r)
		return
	}

	g.proxy.ServeHTTP(w, r)
}

func (g *gateway) withBodyLimit(maxBodyBytes int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil && maxBodyBytes > 0 {
			if r.ContentLength > maxBodyBytes {
				http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		}
		next.ServeHTTP(w, r)
	})
}

func (g *gateway) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.Printf("%s %s %d %s remote=%s", r.Method, r.URL.RequestURI(), sw.status, time.Since(start).Round(time.Millisecond), r.RemoteAddr)
	})
}

func (g *gateway) sourceAllowed(r *http.Request) bool {
	if len(g.cidrs) == 0 {
		return true
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, cidr := range g.cidrs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func parseUpstream(raw string) (*url.URL, error) {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("scheme must be http or https")
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("host is required")
	}
	return parsed, nil
}

func parseCIDRs(raw string) ([]*net.IPNet, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	cidrs := make([]*net.IPNet, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if ip := net.ParseIP(part); ip != nil {
			mask := "/32"
			if ip.To4() == nil {
				mask = "/128"
			}
			part += mask
		}
		_, cidr, err := net.ParseCIDR(part)
		if err != nil {
			return nil, err
		}
		cidrs = append(cidrs, cidr)
	}
	return cidrs, nil
}

func allowedAgentRoute(method, path string) bool {
	switch {
	case path == "/api/v1/public/agents/install.sh":
		return method == http.MethodGet || method == http.MethodHead
	case strings.HasPrefix(path, "/api/v1/public/agents/binaries/"):
		return method == http.MethodGet || method == http.MethodHead
	case path == "/api/v1/public/agents/register":
		return method == http.MethodPost
	case path == "/api/v1/public/agents/heartbeat":
		return method == http.MethodPost
	case path == "/api/v1/public/agents/metrics":
		return method == http.MethodPost
	default:
		return false
	}
}

func forwardedProto(r *http.Request) string {
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		return proto
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
