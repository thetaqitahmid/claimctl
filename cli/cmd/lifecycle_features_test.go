package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"claimctl-cli/pkg/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func featureClientConfig(t *testing.T, address string) {
	t.Helper()
	keys := []string{"url", "token", "netrc", "json", "request-timeout", "token-file", "ca-cert", "allow-insecure-http"}
	previous := map[string]any{}
	for _, key := range keys {
		previous[key] = viper.Get(key)
	}
	t.Cleanup(func() {
		for _, key := range keys {
			viper.Set(key, previous[key])
		}
	})
	viper.Set("url", address)
	viper.Set("token", "secret")
	viper.Set("netrc", false)
	viper.Set("json", false)
	viper.Set("request-timeout", time.Second)
	viper.Set("token-file", "")
	viper.Set("ca-cert", "")
	viper.Set("allow-insecure-http", false)
}

func featureCommand(ctx context.Context) (*cobra.Command, *bytes.Buffer) {
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	return cmd, out
}

func TestReserveAtomicNoQueueAllOutputModes(t *testing.T) {
	for _, mode := range []string{"text", "json", "quiet"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				require.Equal(t, "/api/reservations/claim", r.URL.Path)
				var req api.ClaimRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
				require.True(t, req.NoQueue)
				require.Equal(t, "gpu", req.Type)
				require.True(t, req.RequireHealthy)
				w.WriteHeader(409)
				_, _ = w.Write([]byte(`{"error":"busy"}`))
			}))
			defer server.Close()
			featureClientConfig(t, server.URL)
			previousType, previousHealth, previousNoQueue, previousQuiet, previousWait := reserveType, reserveRequireHealthy, reserveNoQueue, reserveQuiet, reserveWait
			t.Cleanup(func() {
				reserveType, reserveRequireHealthy, reserveNoQueue, reserveQuiet, reserveWait = previousType, previousHealth, previousNoQueue, previousQuiet, previousWait
			})
			reserveType = "gpu"
			reserveRequireHealthy = true
			reserveNoQueue = true
			reserveQuiet = mode == "quiet"
			reserveWait = true
			viper.Set("json", mode == "json")
			cmd, out := featureCommand(context.Background())
			err := reserveCmd.RunE(cmd, nil)
			require.Equal(t, ExitResourceBusy, GetExitCode(err))
			require.Empty(t, out.String())
			require.Equal(t, 1, calls)
		})
	}
}

func TestReserveSelectorOutput(t *testing.T) {
	for _, mode := range []string{"json", "quiet"} {
		t.Run(mode, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/api/reservations/claim", r.URL.Path)
				_, _ = w.Write([]byte(`{"id":"claim","resourceId":"gpu","status":"active","scheduledEndTime":null}`))
			}))
			defer server.Close()
			featureClientConfig(t, server.URL)
			previousName, previousQuiet := reserveName, reserveQuiet
			t.Cleanup(func() { reserveName, reserveQuiet = previousName, previousQuiet })
			reserveName = "gpu"
			reserveQuiet = mode == "quiet"
			viper.Set("json", mode == "json")
			cmd, out := featureCommand(context.Background())
			require.NoError(t, reserveCmd.RunE(cmd, nil))
			if mode == "json" {
				require.True(t, json.Valid(out.Bytes()))
			} else {
				require.Equal(t, "claim\n", out.String())
			}
		})
	}
}

func TestSecureConfigWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte("old"), 0644))
	require.NoError(t, saveConfig(path, []byte(`{"token":"secret"}`)))
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.JSONEq(t, `{"token":"secret"}`, string(data))
	files, err := os.ReadDir(filepath.Dir(path))
	require.NoError(t, err)
	require.Len(t, files, 1)
}

func TestClientSecurityOptions(t *testing.T) {
	featureClientConfig(t, "http://example.com")
	cmd, _ := featureCommand(context.Background())
	_, err := newClient(cmd)
	require.ErrorContains(t, err, "HTTPS required")
	viper.Set("allow-insecure-http", true)
	_, err = newClient(cmd)
	require.NoError(t, err)
	viper.Set("request-timeout", 0)
	_, err = newClient(cmd)
	require.Error(t, err)
}

func TestClientCustomCAAndTokenFile(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer from-file", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()
	featureClientConfig(t, server.URL)
	directory := t.TempDir()
	caPath := filepath.Join(directory, "ca.pem")
	require.NoError(t, os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), 0600))
	tokenPath := filepath.Join(directory, "token")
	require.NoError(t, os.WriteFile(tokenPath, []byte("from-file\n"), 0644))
	viper.Set("ca-cert", caPath)
	viper.Set("token-file", tokenPath)
	cmd, _ := featureCommand(context.Background())
	_, err := newClient(cmd)
	require.ErrorContains(t, err, "owner-only")
	require.NoError(t, os.Chmod(tokenPath, 0600))
	client, err := newClient(cmd)
	require.NoError(t, err)
	resources, err := client.GetResources("")
	require.NoError(t, err)
	require.Empty(t, resources)
	cmd.Flags().String("token", "", "")
	require.NoError(t, cmd.Flags().Set("token", "explicit"))
	viper.Set("token", "explicit")
	client, err = newClient(cmd)
	require.NoError(t, err)
	require.Equal(t, "explicit", client.Token)
}

func execFixture(t *testing.T, status string, renewStatus int) (string, *atomic.Int32, *atomic.Int32, *atomic.Int32) {
	t.Helper()
	renewals := new(atomic.Int32)
	completed := new(atomic.Int32)
	cancelled := new(atomic.Int32)
	snapshot := func(w http.ResponseWriter) {
		expires := time.Now().Add(2 * time.Second)
		hard := time.Now().Add(5 * time.Second)
		require.NoError(t, json.NewEncoder(w).Encode(api.Reservation{ID: "claim", ResourceID: "gpu", Status: status, LeaseExpiresAt: &expires, HardEndTime: &hard}))
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/reservations/claim":
			snapshot(w)
		case "/api/reservations/claim/renew":
			renewals.Add(1)
			if renewStatus != 200 {
				w.WriteHeader(renewStatus)
			} else {
				snapshot(w)
			}
		case "/api/reservations/claim/complete":
			completed.Add(1)
			_, _ = w.Write([]byte(`{}`))
		case "/api/reservations/claim/cancel":
			cancelled.Add(1)
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(server.Close)
	return server.URL, renewals, completed, cancelled
}

func testExecOptions() execOptions {
	return execOptions{claim: api.ClaimRequest{ResourceID: "gpu", LeaseDuration: "2s"}, heartbeat: 100 * time.Millisecond, timeout: 1, poll: 1, grace: 100 * time.Millisecond}
}

func TestExecHeartbeatsAndPreservesExitCode(t *testing.T) {
	address, renewed, completed, cancelled := execFixture(t, "active", 200)
	featureClientConfig(t, address)
	cmd, out := featureCommand(context.Background())
	err := runClaimedCommand(cmd, []string{"sh", "-c", `sleep 0.35; printf '%s:%s' "$CLAIMCTL_RESERVATION_ID" "$CLAIMCTL_RESOURCE_ID"; exit 17`}, testExecOptions())
	require.Equal(t, 17, GetExitCode(err))
	require.Equal(t, "claim:gpu", out.String())
	require.GreaterOrEqual(t, renewed.Load(), int32(1))
	require.Equal(t, int32(1), completed.Load())
	require.Zero(t, cancelled.Load())
}

func TestExecRenewalFailureStopsWorkAndCleansUp(t *testing.T) {
	address, renewed, completed, _ := execFixture(t, "active", 403)
	featureClientConfig(t, address)
	cmd, _ := featureCommand(context.Background())
	started := time.Now()
	err := runClaimedCommand(cmd, []string{"sh", "-c", "sleep 30"}, testExecOptions())
	require.ErrorContains(t, err, "lease renewal failed")
	require.Less(t, time.Since(started), 2*time.Second)
	require.Equal(t, int32(1), renewed.Load())
	require.Equal(t, int32(1), completed.Load())
}

func TestExecQueueTimeoutCancelsWithoutStartingChild(t *testing.T) {
	address, _, completed, cancelled := execFixture(t, "pending", 200)
	featureClientConfig(t, address)
	cmd, out := featureCommand(context.Background())
	err := runClaimedCommand(cmd, []string{"sh", "-c", "echo should-not-run"}, testExecOptions())
	require.Equal(t, ExitTimeout, GetExitCode(err))
	require.Empty(t, out.String())
	require.Zero(t, completed.Load())
	require.Equal(t, int32(1), cancelled.Load())
}

func TestExecStartFailureReleases(t *testing.T) {
	address, _, completed, _ := execFixture(t, "active", 200)
	featureClientConfig(t, address)
	cmd, _ := featureCommand(context.Background())
	err := runClaimedCommand(cmd, []string{filepath.Join(t.TempDir(), "missing")}, testExecOptions())
	require.Error(t, err)
	require.Equal(t, int32(1), completed.Load())
}

func TestWrappedExitCodes(t *testing.T) {
	require.Equal(t, 27, GetExitCode(fmt.Errorf("wrapped: %w", &CLIError{Message: "child", ExitCode: 27})))
	require.Equal(t, ExitResourceBusy, GetExitCode(fmt.Errorf("wrapped: %w", &api.APIError{StatusCode: 409, Body: "conflict"})))
}
