package services

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

// withFastRetries shortens the retry backoff so retry-path tests stay fast.
func withFastRetries(t *testing.T) {
	t.Helper()
	orig := webhookRetryBaseDelay
	webhookRetryBaseDelay = time.Millisecond
	t.Cleanup(func() { webhookRetryBaseDelay = orig })
}

// withStubbedDNS replaces the hostname resolver with a static records map so
// validation tests stay hermetic.
func withStubbedDNS(t *testing.T, records map[string][]net.IP) {
	t.Helper()
	orig := resolveHostIPs
	resolveHostIPs = func(ctx context.Context, host string) ([]net.IP, error) {
		if ips, ok := records[host]; ok {
			return ips, nil
		}
		return nil, fmt.Errorf("no such host %q", host)
	}
	t.Cleanup(func() { resolveHostIPs = orig })
}

func TestValidateWebhookURL(t *testing.T) {
	withStubbedDNS(t, map[string][]net.IP{
		"example.com":              {net.ParseIP("93.184.216.34")},
		"internal.corp":            {net.ParseIP("10.0.0.5")},
		"mixed.corp":               {net.ParseIP("93.184.216.34"), net.ParseIP("10.0.0.5")},
		"metadata.google.internal": {net.ParseIP("169.254.169.254")},
		"rebind.attacker.test":     {net.ParseIP("127.0.0.1")},
	})

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"https url", "https://example.com/hook", false},
		{"http url", "http://example.com/hook", false},
		{"url with port", "http://example.com:8080/hook", false},
		{"uppercase scheme", "HTTPS://example.com/hook", false},
		{"empty url", "", true},
		{"ftp scheme", "ftp://example.com/hook", true},
		{"no host", "http:///hook", true},
		{"localhost", "http://localhost:8080/hook", true},
		{"loopback ipv4", "http://127.0.0.1/hook", true},
		{"loopback ipv6", "http://[::1]/hook", true},
		{"unspecified address", "http://0.0.0.0/hook", true},
		{"private ipv4", "http://192.168.1.10/hook", true},
		{"private cidr 10", "http://10.0.0.5/hook", true},
		{"private cidr 172", "http://172.16.0.1/hook", true},
		{"link local", "http://169.254.169.254/latest/meta-data", true},
		{"hostname resolves to private address", "http://internal.corp/hook", true},
		{"hostname resolves to public and private addresses", "http://mixed.corp/hook", true},
		{"hostname resolves to link-local address", "http://metadata.google.internal/", true},
		{"hostname resolves to loopback", "http://rebind.attacker.test/hook", true},
		{"unresolvable hostname", "http://nonexistent.test/hook", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWebhookURL(context.Background(), tt.url)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateWebhookURL_RespectsContextDeadline(t *testing.T) {
	orig := resolveHostIPs
	resolveHostIPs = func(ctx context.Context, host string) ([]net.IP, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return []net.IP{net.ParseIP("93.184.216.34")}, nil
	}
	t.Cleanup(func() { resolveHostIPs = orig })

	// A canceled context must abort DNS resolution instead of hanging.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := ValidateWebhookURL(ctx, "http://example.com/hook")
	assert.Error(t, err)
}

func TestReadResponseBody(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "short body is returned as-is",
			body:     "ok",
			expected: "ok",
		},
		{
			name:     "body at the limit is not truncated",
			body:     strings.Repeat("a", maxResponseLogBytes),
			expected: strings.Repeat("a", maxResponseLogBytes),
		},
		{
			name:     "body beyond the limit is truncated",
			body:     strings.Repeat("a", maxResponseLogBytes+500),
			expected: strings.Repeat("a", maxResponseLogBytes) + "... (truncated)",
		},
		{
			name: "truncation does not split a multi-byte rune",
			// 999 'a' + the 2-byte rune 'é' exceeds the limit by one byte,
			// so the cut lands mid-rune and must back off to the rune start.
			body:     strings.Repeat("a", maxResponseLogBytes-1) + "é",
			expected: strings.Repeat("a", maxResponseLogBytes-1) + "... (truncated)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{Body: io.NopCloser(strings.NewReader(tt.body))}
			got := readResponseBody(resp)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func newTestWebhookService(t *testing.T) (*WebhookService, *testutils.MockQuerier) {
	t.Helper()
	mockDB := &testutils.MockQuerier{}
	secretSvc := NewSecretService(mockDB, "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=")
	svc := NewWebhookService(mockDB, secretSvc, "")
	// httptest servers bind to loopback, which the SSRF-safe dialer blocks;
	// swap in a plain client so execution tests exercise real round trips.
	svc.httpClient = &http.Client{Timeout: 10 * time.Second}
	return svc, mockDB
}

func testWebhook(url string) db.ClaimctlWebhook {
	return db.ClaimctlWebhook{
		ID:            testutils.TestUUID(1),
		Name:          "test webhook",
		Url:           url,
		Method:        "POST",
		Headers:       []byte(`{}`),
		Template:      pgtype.Text{},
		SigningSecret: "secret",
	}
}

func TestExecuteWebhook_RetriesOnServerError(t *testing.T) {
	withFastRetries(t)
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	svc, mockDB := newTestWebhookService(t)
	var logged db.CreateWebhookLogParams
	mockDB.On("CreateWebhookLog", mock.Anything, mock.AnythingOfType("db.CreateWebhookLogParams")).
		Run(func(args mock.Arguments) {
			logged = args.Get(1).(db.CreateWebhookLogParams)
		}).
		Return(db.ClaimctlWebhookLog{}, nil).Once()

	svc.executeWebhook(context.Background(), testWebhook(srv.URL), WebhookPayload{Event: "reservation.created"})

	assert.EqualValues(t, 2, calls.Load(), "a 500 response should be retried")
	assert.EqualValues(t, http.StatusOK, logged.StatusCode)
	assert.NotEmpty(t, logged.ResponseBody)
	mockDB.AssertExpectations(t)
}

func TestExecuteWebhook_NoRetryOnClientError(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	svc, mockDB := newTestWebhookService(t)
	var logged db.CreateWebhookLogParams
	mockDB.On("CreateWebhookLog", mock.Anything, mock.AnythingOfType("db.CreateWebhookLogParams")).
		Run(func(args mock.Arguments) {
			logged = args.Get(1).(db.CreateWebhookLogParams)
		}).
		Return(db.ClaimctlWebhookLog{}, nil).Once()

	svc.executeWebhook(context.Background(), testWebhook(srv.URL), WebhookPayload{Event: "reservation.created"})

	assert.EqualValues(t, 1, calls.Load(), "a 400 response must not be retried")
	assert.EqualValues(t, http.StatusBadRequest, logged.StatusCode)
	mockDB.AssertExpectations(t)
}

func TestExecuteWebhook_SignatureHeader(t *testing.T) {
	var gotSignature atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSignature.Store(r.Header.Get("X-claimctl-Signature"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	svc, mockDB := newTestWebhookService(t)
	mockDB.On("CreateWebhookLog", mock.Anything, mock.AnythingOfType("db.CreateWebhookLogParams")).
		Return(db.ClaimctlWebhookLog{}, nil).Once()

	svc.executeWebhook(context.Background(), testWebhook(srv.URL), WebhookPayload{Event: "reservation.created"})

	sig, ok := gotSignature.Load().(string)
	require.True(t, ok, "signature header should have been sent")
	assert.True(t, strings.HasPrefix(sig, "sha256="), "signature must be sha256-prefixed, got %q", sig)
	assert.Len(t, strings.TrimPrefix(sig, "sha256="), 64, "signature must be a hex-encoded sha256 hmac")
}

func TestExecuteWebhook_NetworkErrorLogsStatusZero(t *testing.T) {
	withFastRetries(t)
	// Point at a closed port on loopback to force a transport error.
	svc, mockDB := newTestWebhookService(t)
	hook := testWebhook("http://127.0.0.1:1/hook")

	var logged db.CreateWebhookLogParams
	mockDB.On("CreateWebhookLog", mock.Anything, mock.AnythingOfType("db.CreateWebhookLogParams")).
		Run(func(args mock.Arguments) {
			logged = args.Get(1).(db.CreateWebhookLogParams)
		}).
		Return(db.ClaimctlWebhookLog{}, nil).Once()

	svc.executeWebhook(context.Background(), hook, WebhookPayload{Event: "reservation.created"})

	assert.EqualValues(t, 0, logged.StatusCode)
	assert.Contains(t, logged.ResponseBody, "Network Error")
	mockDB.AssertExpectations(t)
}

func TestSafeHTTPClient_BlocksLoopback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := newSafeHTTPClient(5 * time.Second)
	resp, err := client.Get(srv.URL)
	if err == nil {
		resp.Body.Close()
		t.Fatal("expected connection to a loopback address to be blocked by the safe client")
	}
}

func TestSafeDialContext_BlocksPrivateHostname(t *testing.T) {
	withStubbedDNS(t, map[string][]net.IP{
		"internal.corp": {net.ParseIP("10.0.0.5")},
	})

	_, err := safeDialContext(context.Background(), "tcp", "internal.corp:80")
	assert.Error(t, err, "expected dial to a hostname resolving to a private address to be blocked")
}

func TestSafeDialContext_FallsBackAcrossAddresses(t *testing.T) {
	withStubbedDNS(t, map[string][]net.IP{
		"multi.example": {net.ParseIP("93.184.216.34"), net.ParseIP("93.184.216.35")},
	})

	origDial := outboundDial
	var dialed []string
	outboundDial = func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialed = append(dialed, addr)
		if len(dialed) == 1 {
			return nil, fmt.Errorf("connection refused")
		}
		server, client := net.Pipe()
		_ = server // endpoint kept for the caller; closed with the client
		return client, nil
	}
	t.Cleanup(func() { outboundDial = origDial })

	conn, err := safeDialContext(context.Background(), "tcp", "multi.example:80")
	require.NoError(t, err, "expected fallback to the second resolved address to succeed")
	conn.Close()
	assert.Len(t, dialed, 2, "expected both resolved addresses to be tried in order")
	assert.Contains(t, dialed[0], "93.184.216.34")
	assert.Contains(t, dialed[1], "93.184.216.35")
}

func TestSafeDialContext_BlocksUnspecifiedAddress(t *testing.T) {
	_, err := safeDialContext(context.Background(), "tcp", "0.0.0.0:80")
	assert.Error(t, err, "expected dial to the unspecified address to be blocked")
}

func TestResolveSecrets(t *testing.T) {
	secrets := map[string]string{"API_KEY": "abc123", "SLACK_URL": "https://hooks.example/x"}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no placeholders", "https://example.com", "https://example.com"},
		{"single placeholder", "https://example.com?key={{Secret.API_KEY}}", "https://example.com?key=abc123"},
		{"multiple placeholders", "{{Secret.API_KEY}}@{{Secret.SLACK_URL}}", "abc123@https://hooks.example/x"},
		{"unknown placeholder untouched", "{{Secret.MISSING}}", "{{Secret.MISSING}}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestWebhookService(t)
			assert.Equal(t, tt.want, svc.resolveSecrets(tt.input, secrets))
		})
	}
}
