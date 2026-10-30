package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLifecycleAPIContracts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/reservations/policy":
			require.Equal(t, "GET", r.Method)
			_, _ = w.Write([]byte(`{"defaultDuration":"1h","maxDuration":"24h","maxLiveClaims":8,"allowIndefinite":false}`))
		case "/api/reservations/bundle":
			require.Equal(t, "stable", r.Header.Get("Idempotency-Key"))
			var req BundleRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			require.Equal(t, []string{"a", "b"}, req.ResourceIDs)
			require.Equal(t, "2h", req.Duration)
			require.Equal(t, "1m", req.LeaseDuration)
			_, _ = w.Write([]byte(`[{"id":"one","status":"active"},{"id":"two","status":"active"}]`))
		case "/api/reservations/one/renew":
			require.Equal(t, "POST", r.Method)
			var req map[string]string
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			require.Empty(t, req)
			_, _ = w.Write([]byte(`{"id":"one","status":"active","leaseExpiresAt":"2030-01-01T00:00:00Z"}`))
		case "/api/tokens":
			var req map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			require.Equal(t, []any{"space"}, req["space_ids"])
			require.Equal(t, []any{"read", "reserve"}, req["scopes"])
			require.Equal(t, "30d", req["expires_in"])
			_, _ = w.Write([]byte(`{"token":"secret","scopes":["read","reserve"],"spaceIds":["space"]}`))
		case "/api/resources/a/metrics", "/api/admin/metrics":
			require.Equal(t, "100", r.URL.Query().Get("from"))
			require.Equal(t, "200", r.URL.Query().Get("to"))
			_, _ = w.Write([]byte(`{"utilization":0.5}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	c, err := NewClient(server.URL, "", false)
	require.NoError(t, err)
	policy, err := c.Policy()
	require.NoError(t, err)
	require.Equal(t, int64(8), policy.MaxLiveClaims)
	bundle, err := c.Bundle(BundleRequest{ResourceIDs: []string{"a", "b"}, Duration: "2h", LeaseDuration: "1m"}, "stable")
	require.NoError(t, err)
	require.Len(t, bundle, 2)
	renewed, err := c.Renew("one", "")
	require.NoError(t, err)
	require.NotNil(t, renewed.LeaseExpiresAt)
	token, err := c.GenerateScopedToken("worker", "30d", []string{"read", "reserve"}, []string{"space"})
	require.NoError(t, err)
	require.Equal(t, []string{"space"}, token.SpaceIDs)
	for _, id := range []string{"a", ""} {
		result, err := c.Metrics(id, 100, 200)
		require.NoError(t, err)
		require.JSONEq(t, `{"utilization":0.5}`, string(result))
	}
}

func TestReservationJSONPreservesNullFields(t *testing.T) {
	var r Reservation
	require.NoError(t, json.Unmarshal([]byte(`{"id":"claim","startTime":null,"endTime":null,"queuePosition":null,"scheduledEndTime":null,"leaseExpiresAt":null,"hardEndTime":null}`), &r))
	data, err := json.Marshal(r)
	require.NoError(t, err)
	var fields map[string]any
	require.NoError(t, json.Unmarshal(data, &fields))
	for _, key := range []string{"startTime", "endTime", "queuePosition", "scheduledEndTime", "leaseExpiresAt", "hardEndTime"} {
		require.Contains(t, fields, key)
		require.Nil(t, fields[key], key)
	}
}
