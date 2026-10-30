package services

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/thetaqitahmid/claimctl/internal/testutils"
)

func TestBackupService_CanonicalPopulatedRoundtripRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	ctx := context.Background()
	userID, _ := seedRefreshAccount(t, pool)
	spaceID, resourceID, reservationID, tokenID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	_, err := pool.Exec(ctx, "INSERT INTO claimctl.spaces (id,name) VALUES ($1,'Backup Private')", spaceID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "INSERT INTO claimctl.resources (id,name,type,space_id) VALUES ($1,'Backup Resource','test',$2)", resourceID, spaceID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "INSERT INTO claimctl.reservations (id,resource_id,user_id,status,start_time,duration,lease_duration,lease_expires_at,hard_end_time) VALUES ($1,$2,$3,'active',100,INTERVAL '1 month 2 days 03:04:05.123456',INTERVAL '10 minutes','2030-01-01 00:10:00+00','2030-01-02 00:00:00+00')", reservationID, resourceID, userID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "INSERT INTO claimctl.api_tokens (id,user_id,name,token_hash,expires_at,scopes,space_ids) VALUES ($1,$2,'Scoped Backup','synthetic-hash',NOW()+INTERVAL '1 day',ARRAY['read','reserve','users:read'],ARRAY[$3::uuid])", tokenID, userID, spaceID)
	require.NoError(t, err)
	service := NewBackupService(pool)
	backup, err := service.CreateBackup(ctx)
	require.NoError(t, err)
	require.Equal(t, reservationID.String(), backup.Reservations[0]["id"])
	require.Equal(t, "1 mon 2 days 03:04:05.123456", backup.Reservations[0]["duration"])
	require.Equal(t, "00:10:00", backup.Reservations[0]["lease_duration"])
	require.NotNil(t, backup.Reservations[0]["lease_expires_at"])
	require.NotNil(t, backup.Reservations[0]["hard_end_time"])
	require.Equal(t, []interface{}{spaceID.String()}, backup.APITokens[0]["space_ids"])
	encoded, err := json.Marshal(backup)
	require.NoError(t, err)
	require.NoError(t, service.RestoreBackup(ctx, encoded))
	restored, err := service.CreateBackup(ctx)
	require.NoError(t, err)
	require.Equal(t, backup.Users, restored.Users)
	require.Equal(t, backup.Spaces, restored.Spaces)
	require.Equal(t, backup.Resources, restored.Resources)
	require.Equal(t, backup.Reservations, restored.Reservations)
	require.Equal(t, backup.APITokens, restored.APITokens)

	for _, tc := range []struct {
		name   string
		mutate func(*BackupData)
	}{
		{"unknown column", func(b *BackupData) { b.Resources[0]["unexpected_column"] = "reject" }},
		{"foreign key", func(b *BackupData) { b.Reservations[0]["resource_id"] = uuid.NewString() }},
		{"unsupported version", func(b *BackupData) { b.Metadata.Version = "999" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var invalid BackupData
			require.NoError(t, json.Unmarshal(encoded, &invalid))
			tc.mutate(&invalid)
			raw, err := json.Marshal(invalid)
			require.NoError(t, err)
			require.Error(t, service.RestoreBackup(ctx, raw))
			after, err := service.CreateBackup(ctx)
			require.NoError(t, err)
			require.Equal(t, restored.Users, after.Users)
			require.Equal(t, restored.Resources, after.Resources)
			require.Equal(t, restored.Reservations, after.Reservations)
			require.Equal(t, restored.APITokens, after.APITokens)
		})
	}
}

func TestBackupService_ExportFailsFastRealDatabase(t *testing.T) {
	pool := testutils.SecurityTestDatabase(t)
	_, err := pool.Exec(context.Background(), "ALTER TABLE claimctl.api_tokens RENAME TO unavailable_api_tokens")
	require.NoError(t, err)
	backup, err := NewBackupService(pool).CreateBackup(context.Background())
	require.ErrorContains(t, err, "failed to export api_tokens")
	require.Nil(t, backup)
}
