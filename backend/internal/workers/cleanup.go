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
	store        db.Store
	daysToKeep   int
	pollInterval time.Duration
}

func NewCleanupWorker(store db.Store) *CleanupWorker {
	return &CleanupWorker{store: store,
		daysToKeep:   max(1, utils.GetEnvAsInt("DB_CLEANUP_DAYS", 7)),
		pollInterval: time.Duration(max(1, utils.GetEnvAsInt("DB_CLEANUP_INTERVAL_HOURS", 1))) * time.Hour,
	}
}

func (w *CleanupWorker) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(w.pollInterval)
		defer ticker.Stop()
		w.RunCleanup(ctx)
		for {
			select {
			case <-ticker.C:
				w.RunCleanup(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (w *CleanupWorker) RunCleanup(ctx context.Context) {
	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	err := w.store.ExecTx(runCtx, func(q db.Querier) error {
		locked, err := q.TryWorkerTransactionLock(runCtx, "cleanup")
		if err != nil || !locked {
			return err
		}
		cutoff := time.Now().AddDate(0, 0, -w.daysToKeep)
		if err := q.CleanupOldHealthStatus(runCtx, cutoff.Unix()); err != nil {
			return err
		}
		if err := q.CleanupOldWebhookLogs(runCtx, pgtype.Timestamp{Time: cutoff, Valid: true}); err != nil {
			return err
		}
		if err := q.CleanupDeliveredWebhooks(runCtx, pgtype.Timestamptz{Time: cutoff, Valid: true}); err != nil {
			return err
		}
		return q.DeleteExpiredRefreshTokens(runCtx)
	})
	if err != nil {
		slog.Error("Database cleanup failed", "error", err)
	}
}
