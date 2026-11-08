package services

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"

	probing "github.com/prometheus-community/pro-bing"
	"github.com/thetaqitahmid/claimctl/internal/utils"
)

type healthEgressPolicy struct {
	networks []netip.Prefix
	ports    map[string]bool
	allPorts bool
	resolve  func(context.Context, string) ([]netip.Addr, error)
}

func newHealthEgressPolicy() (*healthEgressPolicy, error) {
	p := &healthEgressPolicy{ports: map[string]bool{}, resolve: func(ctx context.Context, host string) ([]netip.Addr, error) {
		return net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	}}
	for _, raw := range strings.Split(utils.GetEnv("HEALTH_CHECK_ALLOWED_CIDRS", ""), ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid health CIDR: %w", err)
		}
		p.networks = append(p.networks, prefix)
	}
	for _, raw := range strings.Split(utils.GetEnv("HEALTH_CHECK_ALLOWED_PORTS", ""), ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if raw == "*" {
			p.allPorts = true
			continue
		}
		port, err := strconv.Atoi(raw)
		if err != nil || port < 1 || port > 65535 {
			return nil, fmt.Errorf("invalid health port")
		}
		p.ports[strconv.Itoa(port)] = true
	}
	return p, nil
}

func (p *healthEgressPolicy) allows(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return false
	}
	for _, cidr := range []string{"100.100.100.200/32", "168.63.129.16/32", "169.254.0.0/16", "fe80::/10", "fd00:ec2::254/128", "fd20:ce::254/128"} {
		if netip.MustParsePrefix(cidr).Contains(ip) {
			return false
		}
	}
	for _, network := range p.networks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func healthTarget(checkType HealthCheckType, target string) (host, port string, err error) {
	switch checkType {
	case HealthCheckTypeHTTP:
		var u *url.URL
		u, err = url.Parse(target)
		if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.Hostname() == "" || u.Fragment != "" {
			return "", "", fmt.Errorf("invalid HTTP target")
		}
		host, port = u.Hostname(), u.Port()
		if port == "" {
			port = "80"
			if u.Scheme == "https" {
				port = "443"
			}
		}
	case HealthCheckTypeTCP:
		host, port, err = net.SplitHostPort(target)
	case HealthCheckTypePing:
		host = target
	default:
		err = fmt.Errorf("invalid check type")
	}
	if host == "" {
		err = fmt.Errorf("empty health target")
	}
	if port != "" {
		n, e := strconv.Atoi(port)
		if e != nil || n < 1 || n > 65535 {
			err = fmt.Errorf("invalid target port")
		}
	}
	return
}

func (p *healthEgressPolicy) addresses(ctx context.Context, host, port string) ([]netip.Addr, error) {
	if port != "" && !p.allPorts && !p.ports[port] {
		return nil, fmt.Errorf("health target port denied")
	}
	var ips []netip.Addr
	if ip, err := netip.ParseAddr(host); err == nil {
		ips = []netip.Addr{ip}
	} else {
		var err error
		ips, err = p.resolve(ctx, host)
		if err != nil {
			return nil, err
		}
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("health target has no addresses")
	}
	for _, ip := range ips {
		if !p.allows(ip) {
			return nil, fmt.Errorf("health target address denied")
		}
	}
	return ips, nil
}

func (p *healthEgressPolicy) dial(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	ips, err := p.addresses(ctx, host, port)
	if err != nil {
		return nil, err
	}
	return (&net.Dialer{}).DialContext(ctx, network, net.JoinHostPort(ips[0].Unmap().String(), port))
}

func newHealthHTTPClient(transport http.RoundTripper) *http.Client {
	return &http.Client{Transport: transport, CheckRedirect: func(req *http.Request, via []*http.Request) error { return fmt.Errorf("health redirects denied") }}
}

func executeHealthProbe(ctx context.Context, checkType HealthCheckType, target string, timeoutSeconds, retryCount int) error {
	if timeoutSeconds < 1 || timeoutSeconds > 30 || retryCount < 0 || retryCount > 3 {
		return fmt.Errorf("health timeout/retries out of bounds")
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()
	policy, err := newHealthEgressPolicy()
	if err != nil {
		return err
	}
	host, port, err := healthTarget(checkType, target)
	if err != nil {
		return err
	}
	ips, err := policy.addresses(ctx, host, port)
	if err != nil {
		return err
	}
	if checkType == HealthCheckTypePing {
		pinger, err := probing.NewPinger(ips[0].Unmap().String())
		if err != nil {
			return err
		}
		pinger.Count = retryCount + 1
		pinger.Timeout = time.Duration(timeoutSeconds) * time.Second
		pinger.SetPrivileged(false)
		if err := pinger.RunWithContext(ctx); err != nil {
			return err
		}
		if pinger.Statistics().PacketsRecv == 0 {
			return fmt.Errorf("no packets received")
		}
		return nil
	}
	transport := &http.Transport{DialContext: policy.dial, DisableKeepAlives: true}
	defer transport.CloseIdleConnections()
	client := newHealthHTTPClient(transport)
	for attempt := 0; attempt <= retryCount; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if checkType == HealthCheckTypeTCP {
			var conn net.Conn
			conn, err = policy.dial(ctx, "tcp", target)
			if err == nil {
				conn.Close()
				return nil
			}
		} else {
			var req *http.Request
			req, err = http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
			if err != nil {
				return err
			}
			var resp *http.Response
			resp, err = client.Do(req)
			if resp != nil {
				resp.Body.Close()
				if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
					return nil
				}
				if err == nil {
					err = fmt.Errorf("HTTP status %d", resp.StatusCode)
				}
			}
		}
	}
	return err
}
