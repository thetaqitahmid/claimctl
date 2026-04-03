package workers

import (
	"context"
	"log/slog"
	"time"

	"github.com/thetaqitahmid/claimctl/internal/services"
)

type ExpiryWorker struct {
	reservationService services.ReservationService
}

func NewExpiryWorker(reservationService services.ReservationService) *ExpiryWorker {
	return &ExpiryWorker{
		reservationService: reservationService,
	}
}

func (w *ExpiryWorker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := w.reservationService.ExpireReservations(ctx); err != nil {
					slog.Info("Error expiring reservations", "error", err)
				}
			case <-ctx.Done():
				ticker.Stop()
				slog.Info("Expiry worker stopped cleanly")
				return
			}
		}
	}()
}
