package services

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/stretchr/testify/require"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
	"github.com/thetaqitahmid/claimctl/internal/types"
)

func TestReplicaRealtimeRetentionOverflowAndDisconnectRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	q := db.NewStore(pool)
	s := NewSharedRealtimeService(ctx, q).(*sharedRealtimeService)
	subscriber, cancel := context.WithCancel(ctx)
	defer cancel()
	events, err := s.Subscribe(subscriber)
	require.NoError(t, err)
	require.Equal(t, "resync", (<-events).Type)
	var id int64
	require.NoError(t, pool.QueryRow(ctx, "INSERT INTO claimctl.realtime_events(event_type) VALUES('resync') RETURNING id").Scan(&id))
	select {
	case event := <-events:
		require.Equal(t, id, event.ID)
	case <-time.After(time.Second):
		t.Fatal("event not delivered")
	}
	_, err = pool.Exec(ctx, "UPDATE claimctl.realtime_events SET created_at=clock_timestamp()-interval '2 hours'")
	require.NoError(t, err)
	require.NoError(t, q.CleanupReplicaState(ctx))
	select {
	case event := <-events:
		require.Equal(t, "resync", event.Type)
		require.Nil(t, event.Payload)
	case <-time.After(time.Second):
		t.Fatal("retention gap not signaled")
	}
	_, err = pool.Exec(ctx, "INSERT INTO claimctl.realtime_events(event_type) SELECT 'resync' FROM generate_series(1,10200)")
	require.NoError(t, err)
	var count int
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.realtime_events").Scan(&count))
	require.LessOrEqual(t, count, 10100)
	for i := 0; i < 201; i++ {
		s.publish(types.Event{Type: "reservation_update", ID: int64(i + 1)})
	}
	foundResync := false
	for len(events) > 0 {
		event := <-events
		if event.Type == "resync" {
			foundResync = true
			require.Nil(t, event.Payload)
		}
	}
	require.True(t, foundResync, "overflow must invalidate, never silently lose state")
	cancel()
	require.Eventually(t, func() bool {
		select {
		case _, ok := <-events:
			return !ok
		default:
			return false
		}
	}, time.Second, time.Millisecond)
	stop()
	_, err = s.Subscribe(context.Background())
	require.Error(t, err)
}

func TestReplicaRealtimeConcurrentOverflowRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := NewSharedRealtimeService(ctx, db.NewStore(pool)).(*sharedRealtimeService)
	events, err := s.Subscribe(ctx)
	require.NoError(t, err)
	reader := make(chan struct{})
	go func() {
		defer close(reader)
		for range events {
			time.Sleep(time.Microsecond)
		}
	}()
	writer := make(chan struct{})
	go func() {
		defer close(writer)
		for i := 0; i < 10000; i++ {
			s.publish(types.Event{Type: "resync"})
		}
	}()
	select {
	case <-writer:
	case <-time.After(3 * time.Second):
		t.Fatal("overflow drain blocked against concurrent receiver")
	}
	cancel()
	select {
	case <-reader:
	case <-time.After(time.Second):
		t.Fatal("subscriber did not disconnect")
	}
}

func TestReplicaUsageMetadataDoesNotEmitResyncRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	ctx := context.Background()
	q := db.NewStore(pool)
	actor, _ := seedRefreshAccount(t, pool)
	_, token, err := NewAPITokenService(q).GenerateToken(ctx, actor, "metadata", nil)
	require.NoError(t, err)
	var before, after int
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.realtime_events").Scan(&before))
	for i := 0; i < 20; i++ {
		require.NoError(t, q.UpdateAPITokenLastUsed(ctx, token.ID))
	}
	_, err = pool.Exec(ctx, "UPDATE claimctl.users SET failed_login_attempts=1,last_login=clock_timestamp() WHERE id=$1", actor)
	require.NoError(t, err)
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.realtime_events").Scan(&after))
	require.Equal(t, before, after, "read usage must not create a refetch/resync feedback loop")
	_, err = pool.Exec(ctx, "UPDATE claimctl.api_tokens SET scopes=ARRAY['read'] WHERE id=$1", token.ID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "UPDATE claimctl.api_tokens SET scopes=ARRAY['read'] WHERE id=$1", token.ID)
	require.NoError(t, err)
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.realtime_events").Scan(&after))
	require.Equal(t, before+1, after, "only actual access changes emit resync")
}

func TestReplicaMigrationPreservesBusinessDataRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	ctx := context.Background()
	actor, resource, _ := seedDelivery(t, pool, "https://example.com")
	_, err := insertDeliveryReservation(ctx, db.NewStore(pool), actor, resource)
	require.NoError(t, err)
	q := db.New(pool)
	require.NoError(t, q.SetSharedValue(ctx, db.SetSharedValueParams{Key: "synthetic", Value: []byte{1}, TtlSeconds: 3600}))
	_, err = q.IncrementRateLimit(ctx, db.IncrementRateLimitParams{Key: "synthetic", TtlSeconds: 60})
	require.NoError(t, err)
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	migration, err := migrate.New("file://"+filepath.Join(filepath.Dir(file), "../../migrations"), pool.Config().ConnString())
	require.NoError(t, err)
	defer migration.Close()
	require.NoError(t, migration.Migrate(24))
	var count int
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.reservations").Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, migration.Up())
	for _, table := range []string{"realtime_events", "shared_storage", "rate_limits"} {
		require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl."+table).Scan(&count))
		require.Zero(t, count)
	}
	_, err = pool.Exec(ctx, "UPDATE claimctl.resources SET is_under_maintenance=true WHERE id=$1", resource)
	require.NoError(t, err)
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.realtime_events").Scan(&count))
	require.Equal(t, 1, count)
}
