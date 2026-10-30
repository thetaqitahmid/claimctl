package services

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

func TestDeliveryRestoreRuntimeRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	ctx := context.Background()
	store := db.NewStore(pool)
	actor, resource, _ := seedDelivery(t, pool, "https://example.com")
	_, err := insertDeliveryReservation(ctx, store, actor, resource)
	require.NoError(t, err)
	_, err = store.AcquireWorkerLease(ctx, db.AcquireWorkerLeaseParams{Name: "health:restore", Fence: uuid.New()})
	require.NoError(t, err)
	svc := NewBackupService(pool)
	backup, err := svc.CreateBackup(ctx)
	require.NoError(t, err)
	raw, err := json.Marshal(backup)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "webhook_outbox")
	var invalid BackupData
	require.NoError(t, json.Unmarshal(raw, &invalid))
	invalid.Reservations[0]["resource_id"] = uuid.NewString()
	bad, err := json.Marshal(invalid)
	require.NoError(t, err)
	require.Error(t, svc.RestoreBackup(ctx, bad))
	var count int
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.webhook_outbox").Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.worker_leases").Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, svc.RestoreBackup(ctx, raw))
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.webhook_outbox").Scan(&count))
	require.Zero(t, count)
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.worker_leases").Scan(&count))
	require.Zero(t, count)
	restored, err := svc.CreateBackup(ctx)
	require.NoError(t, err)
	require.Equal(t, backup.Reservations, restored.Reservations)
	reservationID, err := uuid.Parse(backup.Reservations[0]["id"].(string))
	require.NoError(t, err)
	require.NoError(t, store.CancelReservation(ctx, reservationID))
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM claimctl.webhook_outbox").Scan(&count))
	require.Equal(t, 1, count)
}
