package workers

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/utils"
)

type CleanupWorker struct {
	db           db.Querier
	daysToKeep   int
	pollInterval time.Duration
}

func NewCleanupWorker(database db.Querier) *CleanupWorker {
	// Default to keeping 7 days of data
	daysToKeep := utils.GetEnvAsInt("DB_CLEANUP_DAYS", 7)
	// Default to running cleanup every hour
	pollInterval := time.Duration(utils.GetEnvAsInt("DB_CLEANUP_INTERVAL_HOURS", 1)) * time.Hour

	return &CleanupWorker{
		db:           database,
		daysToKeep:   daysToKeep,
		pollInterval: pollInterval,
	}
}

func (w *CleanupWorker) Start(ctx context.Context) {
	slog.Info("Starting database cleanup worker", "daysToKeep", w.daysToKeep, "interval", w.pollInterval)
	ticker := time.NewTicker(w.pollInterval)

	// Run once immediately on startup
	go func() {
		w.RunCleanup(ctx)

		for {
			select {
			case <-ticker.C:
				w.RunCleanup(ctx)
			case <-ctx.Done():
				ticker.Stop()
				slog.Info("Database cleanup worker stopped cleanly")
				return
			}
		}
	}()
}

func (w *CleanupWorker) RunCleanup(ctx context.Context) {
	slog.Info("Running database cleanup")

	cutoffTime := time.Now().AddDate(0, 0, -w.daysToKeep)
	cutoffEpoch := cutoffTime.Unix()

	// 1. Cleanup Health Check Status
	err := w.db.CleanupOldHealthStatus(ctx, cutoffEpoch)
	if err != nil {
		slog.Error("Failed to cleanup old health status", "error", err)
	} else {
		slog.Debug("Cleaned up old health status records", "olderThan", cutoffTime)
	}

	// 2. Cleanup Webhook Logs
	cutoffTimestamp := pgtype.Timestamp{Time: cutoffTime, Valid: true}
	err = w.db.CleanupOldWebhookLogs(ctx, cutoffTimestamp)
	if err != nil {
		slog.Error("Failed to cleanup old webhook logs", "error", err)
	} else {
		slog.Debug("Cleaned up old webhook logs", "olderThan", cutoffTime)
	}
}
