package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestURLValidation(t *testing.T) {
	for _, raw := range []string{"", "example.com", "ftp://example.com", "https://user:pass@example.com", "https://example.com?token=x", "https://example.com#x"} {
		_, err := NewClient(raw, "", false)
		require.Error(t, err, raw)
	}
	c, err := NewClient("https://example.com/prefix/", "", false)
	require.NoError(t, err)
	require.Equal(t, "https://example.com/prefix", c.BaseURL)
}

func TestTransportMutationIsNotRetried(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(503) }))
	defer server.Close()
	c, err := NewClient(server.URL, "", false)
	require.NoError(t, err)
	c.RetryWait = time.Millisecond
	_, err = c.CreateResource(CreateResourceRequest{Name: "one"})
	require.Error(t, err)
	require.Equal(t, 1, calls)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, 503, apiErr.StatusCode)
}

func TestClaimRetryUsesOneKeyAndBody(t *testing.T) {
	calls := 0
	key := ""
	var original ClaimRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/reservations/claim", r.URL.Path)
		require.Equal(t, "Bearer secret", r.Header.Get("Authorization"))
		var body ClaimRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		calls++
		if calls == 1 {
			key = r.Header.Get("Idempotency-Key")
			original = body
			w.WriteHeader(503)
			return
		}
		require.NotEmpty(t, key)
		require.Equal(t, key, r.Header.Get("Idempotency-Key"))
		require.Equal(t, original, body)
		_, _ = w.Write([]byte(`{"id":"claim","status":"active","scheduledEndTime":null,"leaseExpiresAt":null,"hardEndTime":null}`))
	}))
	defer server.Close()
	c, err := NewClient(server.URL, "secret", false)
	require.NoError(t, err)
	c.RetryWait = time.Millisecond
	result, err := c.Claim(ClaimRequest{Name: "gpu", NoQueue: true, RequireHealthy: true}, "")
	require.NoError(t, err)
	require.Equal(t, "claim", result.ID)
	require.Equal(t, 2, calls)
	require.Nil(t, result.ScheduledEndTime)
}

func TestRedirectDoesNotForwardCredentials(t *testing.T) {
	reached := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached = true }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 307) }))
	defer server.Close()
	c, err := NewClient(server.URL, "secret", false)
	require.NoError(t, err)
	_, err = c.GetResources("")
	require.Error(t, err)
	require.False(t, reached)
}

func TestRetryBackoffCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) }))
	defer server.Close()
	c, err := NewClient(server.URL, "", false)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	c.Context = ctx
	started := time.Now()
	_, err = c.GetResources("")
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Less(t, time.Since(started), time.Second)
}

func TestNetrcPermissions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".netrc")
	require.NoError(t, os.WriteFile(path, []byte("machine example.com login token password secret\n"), 0644))
	_, err := NewClient("https://example.com", "", true)
	require.Error(t, err)
	require.NoError(t, os.Chmod(path, 0600))
	c, err := NewClient("https://example.com", "", true)
	require.NoError(t, err)
	require.Equal(t, "secret", c.Token)
}

func TestWaitValidationCancellationAndTerminal(t *testing.T) {
	c, err := NewClient("http://127.0.0.1", "", false)
	require.NoError(t, err)
	require.Error(t, c.WaitForReservation("id", 10, 0, nil))
	require.Error(t, c.WaitForReservation("id", 0, 1, nil))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"id":"id","status":"expired"}`)) }))
	defer server.Close()
	c.BaseURL = server.URL
	require.ErrorContains(t, c.WaitForReservation("id", 10, 1, nil), "expired")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c.Context = ctx
	require.True(t, errors.Is(c.WaitForReservation("id", 10, 1, nil), context.Canceled))
}
