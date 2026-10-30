//go:build unix

package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"claimctl-cli/pkg/api"
	"github.com/stretchr/testify/require"
)

func TestExecRejectsRetiredIdempotencySnapshot(t *testing.T) {
	cleanup := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PATCH" {
			cleanup++
			_, _ = w.Write([]byte(`{}`))
			return
		}
		status := "active"
		if r.Method == "GET" {
			status = "completed"
		}
		lease := time.Now().Add(time.Minute)
		hard := time.Now().Add(time.Hour)
		require.NoError(t, json.NewEncoder(w).Encode(api.Reservation{ID: "claim", ResourceID: "gpu", Status: status, LeaseExpiresAt: &lease, HardEndTime: &hard}))
	}))
	defer server.Close()
	featureClientConfig(t, server.URL)
	cmd, out := featureCommand(context.Background())
	err := runClaimedCommand(cmd, []string{"sh", "-c", "echo unprotected"}, testExecOptions())
	require.ErrorContains(t, err, "completed")
	require.Empty(t, out.String())
	require.Zero(t, cleanup)
}

func TestExecActivationRefreshesHoldDeadline(t *testing.T) {
	var reads atomic.Int32
	var completed atomic.Int32
	activeEnd := time.Now().Add(200 * time.Millisecond)
	lease := time.Now().Add(3 * time.Second)
	hard := time.Now().Add(time.Hour)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PATCH" {
			completed.Add(1)
			_, _ = w.Write([]byte(`{}`))
			return
		}
		snapshot := api.Reservation{ID: "claim", ResourceID: "gpu", Status: "pending", LeaseExpiresAt: &lease, HardEndTime: &hard}
		if r.Method == "GET" && reads.Add(1) >= 2 {
			snapshot.Status = "active"
			snapshot.ScheduledEndTime = &activeEnd
		}
		require.NoError(t, json.NewEncoder(w).Encode(snapshot))
	}))
	defer server.Close()
	featureClientConfig(t, server.URL)
	cmd, out := featureCommand(context.Background())
	opts := testExecOptions()
	opts.heartbeat = 500 * time.Millisecond
	err := runClaimedCommand(cmd, []string{"sh", "-c", "sleep 0.4; echo ran-beyond-hold; sleep 30"}, opts)
	require.ErrorContains(t, err, "deadline")
	require.Empty(t, out.String())
	require.Equal(t, int32(1), completed.Load())
}
