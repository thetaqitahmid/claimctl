package services

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"testing"
	"time"
)

func TestCalculateMetricsExactWindow(t *testing.T) {
	epoch := func(n int64) pgtype.Int8 { return pgtype.Int8{Int64: n, Valid: true} }
	deadline := func(n int64) pgtype.Timestamptz { return pgtype.Timestamptz{Time: time.Unix(n, 0), Valid: true} }
	rows := []db.ClaimctlReservation{
		{CreatedAt: epoch(80), StartTime: epoch(90), EndTime: epoch(140), Status: pgtype.Text{String: "completed", Valid: true}},
		{CreatedAt: epoch(100), StartTime: epoch(120), LeaseExpiresAt: deadline(160), Status: pgtype.Text{String: "active", Valid: true}},
		{CreatedAt: epoch(180), StartTime: epoch(190), ScheduledEndTime: deadline(205), Status: pgtype.Text{String: "active", Valid: true}},
		{CreatedAt: epoch(170), StartTime: epoch(220), EndTime: epoch(250), Status: pgtype.Text{String: "completed", Valid: true}},
		{CreatedAt: epoch(200), StartTime: epoch(210), Status: pgtype.Text{String: "active", Valid: true}},
		{CreatedAt: epoch(110), EndTime: epoch(150), Status: pgtype.Text{String: "cancelled", Valid: true}},
	}
	m := calculateMetrics(uuid.New(), rows, 100, 200)
	require.Equal(t, int64(70), m.OccupiedSeconds)
	require.Equal(t, 0.7, m.Utilization)
	require.Equal(t, int64(4), m.ClaimCount)
	require.Equal(t, int64(2), m.OpenClaims)
	require.Equal(t, int64(2), m.QueueWaitSamples)
	require.Equal(t, float64(15), m.MeanQueueWaitSeconds)
	require.Equal(t, int64(1), m.OpenPendingSamples)
	require.Equal(t, int64(30), m.OpenPendingWaitSeconds)
	empty := calculateMetrics(uuid.New(), nil, 100, 200)
	require.Zero(t, empty.Utilization)
	require.Zero(t, empty.MeanQueueWaitSeconds)
}
