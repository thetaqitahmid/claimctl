package services

import (
	"context"
	"io"
	"net/http"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHealthEgress_ExplicitAllowlistAndMetadataDeny(t *testing.T) {
	t.Setenv("HEALTH_CHECK_ALLOWED_CIDRS", "10.20.0.0/16,0.0.0.0/0,::/0")
	t.Setenv("HEALTH_CHECK_ALLOWED_PORTS", "80,443,8080")
	policy, err := newHealthEgressPolicy()
	require.NoError(t, err)
	for _, target := range []string{"127.0.0.1", "169.254.169.254", "100.100.100.200", "168.63.129.16", "::1", "fe80::1", "fd00:ec2::254", "fd20:ce::254", "::ffff:169.254.169.254", "0.0.0.0", "224.0.0.1"} {
		require.False(t, policy.allows(netip.MustParseAddr(target)), target)
	}
	require.True(t, policy.allows(netip.MustParseAddr("10.20.0.2")))
	_, err = policy.addresses(context.Background(), "10.20.0.2", "22")
	require.ErrorContains(t, err, "port denied")
	t.Setenv("HEALTH_CHECK_ALLOWED_CIDRS", "")
	policy, err = newHealthEgressPolicy()
	require.NoError(t, err)
	_, err = policy.addresses(context.Background(), "10.20.0.2", "80")
	require.ErrorContains(t, err, "address denied")
}

func TestHealthEgress_WildcardPorts(t *testing.T) {
	t.Setenv("HEALTH_CHECK_ALLOWED_CIDRS", "10.20.0.0/16")
	t.Setenv("HEALTH_CHECK_ALLOWED_PORTS", "*")
	policy, err := newHealthEgressPolicy()
	require.NoError(t, err)
	for _, port := range []string{"22", "80", "5432", "65535"} {
		_, err = policy.addresses(context.Background(), "10.20.0.2", port)
		require.NoError(t, err, port)
	}
	_, err = policy.addresses(context.Background(), "169.254.169.254", "80")
	require.ErrorContains(t, err, "address denied")

	t.Setenv("HEALTH_CHECK_ALLOWED_PORTS", "80,*")
	policy, err = newHealthEgressPolicy()
	require.NoError(t, err)
	_, err = policy.addresses(context.Background(), "10.20.0.2", "5432")
	require.NoError(t, err)
}

func TestHealthEgress_DNSRebindingAndMixedAnswers(t *testing.T) {
	t.Setenv("HEALTH_CHECK_ALLOWED_CIDRS", "10.20.0.0/16")
	t.Setenv("HEALTH_CHECK_ALLOWED_PORTS", "80")
	policy, err := newHealthEgressPolicy()
	require.NoError(t, err)
	calls := 0
	policy.resolve = func(ctx context.Context, host string) ([]netip.Addr, error) {
		calls++
		if calls == 1 {
			return []netip.Addr{netip.MustParseAddr("10.20.0.2")}, nil
		}
		return []netip.Addr{netip.MustParseAddr("169.254.169.254")}, nil
	}
	_, err = policy.addresses(context.Background(), "lab.example", "80")
	require.NoError(t, err)
	_, err = policy.dial(context.Background(), "tcp", "lab.example:80")
	require.ErrorContains(t, err, "address denied")
	require.Equal(t, 2, calls)
	policy.resolve = func(ctx context.Context, host string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("10.20.0.2"), netip.MustParseAddr("127.0.0.1")}, nil
	}
	_, err = policy.addresses(context.Background(), "lab.example", "80")
	require.Error(t, err)
}

type healthRoundTripper func(*http.Request) (*http.Response, error)

func (f healthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestHealthEgress_HTTPRedirectNeverFollowed(t *testing.T) {
	calls := 0
	client := newHealthHTTPClient(healthRoundTripper(func(req *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{"http://169.254.169.254/latest/meta-data"}}, Body: io.NopCloser(strings.NewReader("")), Request: req}, nil
	}))
	resp, err := client.Get("http://lab.example/status")
	if resp != nil {
		resp.Body.Close()
	}
	require.ErrorContains(t, err, "redirects denied")
	require.Equal(t, 1, calls)
}

func TestHealthEgress_BoundsAndCancellation(t *testing.T) {
	t.Setenv("HEALTH_CHECK_ALLOWED_CIDRS", "10.20.0.0/16")
	t.Setenv("HEALTH_CHECK_ALLOWED_PORTS", "80")
	require.Error(t, executeHealthProbe(context.Background(), HealthCheckTypeHTTP, "http://10.20.0.2", 31, 0))
	require.Error(t, executeHealthProbe(context.Background(), HealthCheckTypeHTTP, "http://10.20.0.2", 1, 4))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	require.ErrorIs(t, executeHealthProbe(ctx, HealthCheckTypeHTTP, "http://10.20.0.2", 30, 3), context.Canceled)
	require.Less(t, time.Since(start), time.Second)
	_, _, err := healthTarget(HealthCheckTypeHTTP, "http://user:pass@10.20.0.2")
	require.Error(t, err)
	_, _, err = healthTarget(HealthCheckTypeHTTP, "file:///etc/passwd")
	require.Error(t, err)
}
