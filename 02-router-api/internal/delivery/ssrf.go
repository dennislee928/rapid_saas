package delivery

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"
)

func DefaultHTTPClient() *http.Client {
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           guardedDialContext((&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext),
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		IdleConnTimeout:       60 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
	}
	return &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func guardedDialContext(next func(context.Context, string, string) (net.Conn, error)) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		for _, ip := range ips {
			if blockedIP(ip.IP) {
				return nil, errors.New("destination resolves to blocked IP range")
			}
		}
		if len(ips) == 0 {
			return nil, errors.New("destination did not resolve")
		}
		return next(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
	}
}

func blockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	privateBlocks := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"100.64.0.0/10",
		"169.254.0.0/16",
		"fc00::/7",
		"fe80::/10",
	}
	for _, block := range privateBlocks {
		_, network, err := net.ParseCIDR(block)
		if err == nil && network.Contains(ip) {
			return true
		}
	}
	return false
}
