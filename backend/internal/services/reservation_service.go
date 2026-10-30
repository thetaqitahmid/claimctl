package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/types"
)

type QueueItem struct {
	ID             uuid.UUID          `json:"id"`
	UserID         uuid.UUID          `json:"userId"`
	Status         string             `json:"status"`
	QueuePosition  int32              `json:"queuePosition"`
	StartTime      int64              `json:"startTime"`
	CreatedAt      int64              `json:"createdAt"`
	UserName       string             `json:"userName"`
	UserEmail      string             `json:"userEmail"`
	Duration       *string            `json:"duration,omitempty"`
	LeaseExpiresAt pgtype.Timestamptz `json:"leaseExpiresAt"`
	HardEndTime    pgtype.Timestamptz `json:"hardEndTime"`
}

type ReservationService interface {
	CreateReservation(ctx context.Context, userID uuid.UUID, resourceID uuid.UUID, duration *string) (*db.ClaimctlReservation, error)
	CancelAllForResource(ctx context.Context, resourceID uuid.UUID) error
	ActivateReservation(ctx context.Context, reservationID uuid.UUID) (*db.ClaimctlReservation, error)
	CompleteReservation(ctx context.Context, reservationID uuid.UUID) error
	CancelReservation(ctx context.Context, reservationID uuid.UUID, userID uuid.UUID) error
	GetReservation(ctx context.Context, reservationID uuid.UUID) (*db.ClaimctlReservation, error)
	GetUserReservations(ctx context.Context, userID uuid.UUID) ([]db.FindUserActiveReservationsRow, error)
	GetResourceReservations(ctx context.Context, resourceID uuid.UUID) ([]db.FindReservationsByResourceRow, error)
	GetAllReservations(ctx context.Context) ([]db.FindAllReservationsRow, error)
	GetNextInQueue(ctx context.Context, resourceID uuid.UUID) (*db.ClaimctlReservation, error)
	GetUserQueuePosition(ctx context.Context, userID uuid.UUID, resourceID uuid.UUID) (int64, error)
	ProcessQueue(ctx context.Context, resourceID uuid.UUID) error
	GetQueueForResource(ctx context.Context, resourceID uuid.UUID) ([]QueueItem, error)
	ExpireReservations(ctx context.Context) error
}

type reservationService struct {
	policy          ReservationPolicy
	store           db.Store
	historySvc      ReservationHistoryService
	webhookSvc      *WebhookService
	realtimeSvc     RealtimeService
	notificationSvc NotificationService
}

func NewReservationService(store db.Store, historySvc ReservationHistoryService, webhookSvc *WebhookService, realtimeSvc RealtimeService, notificationSvc NotificationService) ReservationService {
	policy, err := LoadReservationPolicy()
	if err != nil {
		panic(err)
	}
	return &reservationService{
		policy:          policy,
		store:           store,
		historySvc:      historySvc,
		webhookSvc:      webhookSvc,
		realtimeSvc:     realtimeSvc,
		notificationSvc: notificationSvc,
	}
}

func (s *reservationService) CreateReservation(ctx context.Context, userID uuid.UUID, resourceID uuid.UUID, duration *string) (*db.ClaimctlReservation, error) {
	return s.Claim(ctx, userID, ClaimRequest{ResourceID: resourceID, Duration: duration}, "")
}

func (s *reservationService) notify(ctx context.Context, userID uuid.UUID, event string, payload NotificationPayload) {
	if s.notificationSvc == nil {
		return
	}
	notifyCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.notificationSvc.Notify(notifyCtx, userID, event, payload); err != nil {
		slog.Warn("Reservation notification failed", "event", event)
	}
}

func (s *reservationService) publishCreated(ctx context.Context, reservation db.ClaimctlReservation) {
	resourceID, userID := reservation.ResourceID, reservation.UserID

	// Send Notification
	resource, _ := s.store.FindResourceById(ctx, resourceID)
	s.notify(ctx, userID, "reservation_created", NotificationPayload{
		Subject: "Reservation Created",
		Message: fmt.Sprintf("Your reservation for resource '%s' has been created.", resource.Name),
	})

	// Broadcast Realtime Event
	s.realtimeSvc.Broadcast(types.Event{
		Type: types.EventReservationUpdate,
		Payload: map[string]interface{}{
			"resource_id": resourceID,
			"action":      "created",
		},
	})

}

func (s *reservationService) CancelAllForResource(ctx context.Context, resourceID uuid.UUID) error {
	if err := AuthorizeRequestResource(ctx, s.store, resourceID, true); err != nil {
		return err
	}
	err := s.store.CancelAllReservationsForResource(ctx, resourceID)
	if err != nil {
		return fmt.Errorf("failed to cancel all reservations: %w", err)
	}
	slog.Info("[RESERVATION] Cancelled all reservations for resource", "resource_id: ", resourceID)

	// Broadcast Realtime Event
	resource, _ := s.store.FindResourceById(ctx, resourceID)
	s.realtimeSvc.Broadcast(types.Event{
		Type: types.EventReservationUpdate,
		Payload: map[string]interface{}{
			"resource_id":   resourceID,
			"resource_name": resource.Name,
			"action":        "cancel_all",
		},
	})
	return nil
}

func (s *reservationService) activateReservationInternal(ctx context.Context, reservationID uuid.UUID) error {
	slog.Info("[ACTIVATE] Starting activation", "reservation_id", reservationID)

	var reservation db.ClaimctlReservation

	err := s.store.ExecTx(ctx, func(q db.Querier) error {
		res, err := q.FindReservationById(ctx, reservationID)
		if err != nil {
			slog.Error("[ACTIVATE] Reservation not found", "reservation_id", reservationID, "error", err)
			return fmt.Errorf("reservation not found: %w", err)
		}

		// Acquire exclusive lock on the resource
		_, err = q.AcquireResourceLock(ctx, res.ResourceID)
		if err != nil {
			return fmt.Errorf("resource not found or failed to lock: %w", err)
		}

		res, err = q.FindReservationById(ctx, reservationID)
		if err != nil {
			return err
		}
		if res.LeaseExpiresAt.Valid && !res.LeaseExpiresAt.Time.After(time.Now()) || res.HardEndTime.Valid && !res.HardEndTime.Time.After(time.Now()) {
			return ErrClaimConflict
		}

		slog.Info("[ACTIVATE] Found reservation", "reservation_id", reservationID, "status", res.Status.String, "resource_id", res.ResourceID, "user_id", res.UserID, "queue_position", res.QueuePosition.Int32)

		if res.Status.String != "pending" {
			slog.Error("[ACTIVATE] Reservation not pending", "reservation_id", reservationID, "status", res.Status.String)
			return fmt.Errorf("reservation cannot be activated, current status: %s", res.Status.String)
		}

		// Check if there's already an active reservation for this resource
		activeReservation, err := q.FindActiveReservationByResource(ctx, res.ResourceID)
		slog.Info("[ACTIVATE] Checked for active reservation", "resource_id", res.ResourceID, "found_active", err == nil, "active_id", activeReservation.ID, "error", err)

		if err == nil && activeReservation.ID != reservationID {
			slog.Error("[ACTIVATE] Resource already has active reservation", "active_id", activeReservation.ID, "trying_to_activate", reservationID)
			return fmt.Errorf("resource already has an active reservation")
		}

		slog.Info("[ACTIVATE] Promoting queue positions", "resource_id", res.ResourceID, "position", res.QueuePosition.Int32)
		// Update queue positions for all reservations that were behind the given position internally within the transaction
		if err = q.UpdateQueuePositions(ctx, db.UpdateQueuePositionsParams{
			ResourceID:    res.ResourceID,
			QueuePosition: pgtype.Int4{Int32: res.QueuePosition.Int32, Valid: true},
		}); err != nil {
			return err
		}

		slog.Info("[ACTIVATE] Executing ActivateReservation SQL", "reservation_id", reservationID)
		activated, activationErr := q.ActivateReservation(ctx, reservationID)
		err = activationErr
		if err == nil && activated != 1 {
			return ErrClaimConflict
		}
		if err != nil {
			slog.Error("[ACTIVATE] ActivateReservation SQL failed", "reservation_id", reservationID, "error", err)
			return fmt.Errorf("failed to activate reservation: %w", err)
		}

		reservation = res
		return nil
	})

	if err != nil {
		return err
	}

	slog.Info("[ACTIVATE] Reservation activated successfully", "reservation_id", reservationID)

	// Send Notification
	activatedResource, _ := s.store.FindResourceById(ctx, reservation.ResourceID)
	s.notify(ctx, reservation.UserID, "reservation_activated", NotificationPayload{
		Subject: "Reservation Activated",
		Message: fmt.Sprintf("Your reservation for resource '%s' is now active.", activatedResource.Name),
	})

	// Broadcast Realtime Event
	s.realtimeSvc.Broadcast(types.Event{
		Type: types.EventReservationUpdate,
		Payload: map[string]interface{}{
			"resource_id":   reservation.ResourceID,
			"resource_name": activatedResource.Name,
			"action":        "activated",
		},
	})

	return nil
}

func (s *reservationService) ActivateReservation(ctx context.Context, reservationID uuid.UUID) (*db.ClaimctlReservation, error) {
	if err := AuthorizeRequestReservation(ctx, s.store, reservationID, true); err != nil {
		return nil, err
	}
	err := s.activateReservationInternal(ctx, reservationID)
	if err != nil {
		return nil, err
	}

	// Get the updated reservation
	updatedReservation, err := s.store.FindReservationById(ctx, reservationID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve updated reservation: %w", err)
	}

	return &updatedReservation, nil
}

func (s *reservationService) CompleteReservation(ctx context.Context, reservationID uuid.UUID) error {
	if err := AuthorizeRequestReservation(ctx, s.store, reservationID, false); err != nil {
		return err
	}
	var reservation db.ClaimctlReservation

	err := s.store.ExecTx(ctx, func(q db.Querier) error {
		res, err := q.FindReservationById(ctx, reservationID)
		if err != nil {
			return fmt.Errorf("reservation not found: %w", err)
		}

		// Acquire exclusive lock on the resource
		_, err = q.AcquireResourceLock(ctx, res.ResourceID)
		if err != nil {
			return fmt.Errorf("resource not found or failed to lock: %w", err)
		}

		if res.Status.String != "active" {
			return fmt.Errorf("reservation cannot be completed, current status: %s", res.Status.String)
		}

		// Complete the reservation
		err = q.CompleteReservation(ctx, reservationID)
		if err != nil {
			return fmt.Errorf("failed to complete reservation: %w", err)
		}

		reservation = res
		return nil
	})

	if err != nil {
		return err
	}

	// Send Notification
	completedResource, _ := s.store.FindResourceById(ctx, reservation.ResourceID)
	s.notify(ctx, reservation.UserID, "reservation_completed", NotificationPayload{
		Subject: "Reservation Completed",
		Message: fmt.Sprintf("Your reservation for resource '%s' has been completed.", completedResource.Name),
	})

	slog.Info("[RESERVATION] Completed reservation", "reservation_id: ", reservationID, "resource_id: ", reservation.ResourceID, "user_id: ", reservation.UserID)

	// Process the queue after completion to activate the next reservation
	s.processQueueInternal(ctx, reservation.ResourceID)

	// Broadcast Realtime Event
	s.realtimeSvc.Broadcast(types.Event{
		Type: types.EventReservationUpdate,
		Payload: map[string]interface{}{
			"resource_id":   reservation.ResourceID,
			"resource_name": completedResource.Name,
			"action":        "completed",
		},
	})

	return nil
}

func (s *reservationService) CancelReservation(ctx context.Context, reservationID uuid.UUID, userID uuid.UUID) error {
	if err := AuthorizeRequestReservation(ctx, s.store, reservationID, false); err != nil {
		return err
	}
	var reservation db.ClaimctlReservation

	err := s.store.ExecTx(ctx, func(q db.Querier) error {
		res, err := q.FindReservationById(ctx, reservationID)
		if err != nil {
			return fmt.Errorf("reservation not found: %w", err)
		}

		// Acquire exclusive lock on the resource
		_, err = q.AcquireResourceLock(ctx, res.ResourceID)
		if err != nil {
			return fmt.Errorf("resource not found or failed to lock: %w", err)
		}

		// Check if user owns this reservation or is admin
		if res.UserID != userID {
			return fmt.Errorf("user does not own this reservation")
		}

		if res.Status.String != "pending" && res.Status.String != "active" {
			return fmt.Errorf("reservation cannot be cancelled, current status: %s", res.Status.String)
		}

		// Cancel the reservation
		err = q.CancelReservation(ctx, reservationID)
		if err != nil {
			return fmt.Errorf("failed to cancel reservation: %w", err)
		}

		reservation = res
		return nil
	})

	if err != nil {
		return err
	}

	// Send Notification
	cancelledRes, _ := s.store.FindResourceById(ctx, reservation.ResourceID)
	s.notify(ctx, reservation.UserID, "reservation_cancelled", NotificationPayload{
		Subject: "Reservation Cancelled",
		Message: fmt.Sprintf("Your reservation for resource '%s' has been cancelled.", cancelledRes.Name),
	})

	slog.Info("[RESERVATION] Cancelled reservation", "reservation_id", reservationID, "resource_id", reservation.ResourceID, "user_id", reservation.UserID)

	// If this was an active reservation, process the queue to activate the next one
	if reservation.Status.String == "active" {
		s.processQueueInternal(ctx, reservation.ResourceID)
	} else {
		// If this was a pending reservation, promote others in the queue
		s.PromoteQueuePositions(ctx, reservation.ResourceID, reservation.QueuePosition.Int32)
	}

	// Broadcast Realtime Event
	s.realtimeSvc.Broadcast(types.Event{
		Type: types.EventReservationUpdate,
		Payload: map[string]interface{}{
			"resource_id":   reservation.ResourceID,
			"resource_name": cancelledRes.Name,
			"action":        "cancelled",
		},
	})

	return nil
}

func (s *reservationService) GetReservation(ctx context.Context, reservationID uuid.UUID) (*db.ClaimctlReservation, error) {
	if err := AuthorizeRequestReservation(ctx, s.store, reservationID, false); err != nil {
		return nil, err
	}
	reservation, err := s.store.FindReservationById(ctx, reservationID)
	if err != nil {
		return nil, fmt.Errorf("reservation not found: %w", err)
	}
	return &reservation, nil
}

func (s *reservationService) GetUserReservations(ctx context.Context, userID uuid.UUID) ([]db.FindUserActiveReservationsRow, error) {
	reservations, err := s.store.FindUserActiveReservations(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user reservations: %w", err)
	}
	if auth, ok := AuthorizationFromContext(ctx); ok {
		if userID != auth.User.ID {
			return nil, ErrAccessDenied
		}
		visible := make([]db.FindUserActiveReservationsRow, 0, len(reservations))
		for _, row := range reservations {
			if AuthorizeRequestResource(ctx, s.store, row.ResourceID, false) == nil {
				visible = append(visible, row)
			}
		}
		reservations = visible
	}
	return reservations, nil
}

func (s *reservationService) GetResourceReservations(ctx context.Context, resourceID uuid.UUID) ([]db.FindReservationsByResourceRow, error) {
	if err := AuthorizeRequestResource(ctx, s.store, resourceID, false); err != nil {
		return nil, err
	}
	reservations, err := s.store.FindReservationsByResource(ctx, resourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource reservations: %w", err)
	}
	return reservations, nil
}

func (s *reservationService) GetAllReservations(ctx context.Context) ([]db.FindAllReservationsRow, error) {
	reservations, err := s.store.FindAllReservations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all reservations: %w", err)
	}
	return reservations, nil
}

func (s *reservationService) GetNextInQueue(ctx context.Context, resourceID uuid.UUID) (*db.ClaimctlReservation, error) {
	if err := AuthorizeRequestResource(ctx, s.store, resourceID, false); err != nil {
		return nil, err
	}
	reservation, err := s.store.GetNextInQueue(ctx, resourceID)
	if err != nil {
		return nil, fmt.Errorf("no reservations in queue: %w", err)
	}
	return &reservation, nil
}

func (s *reservationService) GetUserQueuePosition(ctx context.Context, userID uuid.UUID, resourceID uuid.UUID) (int64, error) {
	if err := AuthorizeRequestResource(ctx, s.store, resourceID, false); err != nil {
		return 0, err
	}
	position, err := s.store.GetUserQueuePosition(ctx, db.GetUserQueuePositionParams{
		UserID:     userID,
		ResourceID: resourceID,
	})
	if err != nil {
		return 0, fmt.Errorf("user not in queue for this resource: %w", err)
	}
	return int64(position.Int32), nil
}

func (s *reservationService) ProcessQueue(ctx context.Context, resourceID uuid.UUID) error {
	if err := AuthorizeRequestResource(ctx, s.store, resourceID, true); err != nil {
		return err
	}
	return s.processQueueInternal(ctx, resourceID)
}

func (s *reservationService) processQueueInternal(ctx context.Context, resourceID uuid.UUID) error {
	nextReservation, err := s.store.GetNextInQueue(ctx, resourceID)
	if err != nil {
		return nil
	}

	err = s.activateReservationInternal(ctx, nextReservation.ID)

	if err == nil {
		// Broadcast Queue Update
		queueResource, _ := s.store.FindResourceById(ctx, resourceID)
		s.realtimeSvc.Broadcast(types.Event{
			Type: types.EventQueueUpdate,
			Payload: map[string]interface{}{
				"resource_id":   resourceID,
				"resource_name": queueResource.Name,
			},
		})
	}

	return err
}

func (s *reservationService) PromoteQueuePositions(ctx context.Context, resourceID uuid.UUID, position int32) {
	// Update queue positions for all reservations that were behind the given position
	s.store.UpdateQueuePositions(ctx, db.UpdateQueuePositionsParams{
		ResourceID:    resourceID,
		QueuePosition: pgtype.Int4{Int32: position, Valid: true},
	})
}

func (s *reservationService) GetQueueForResource(ctx context.Context, resourceID uuid.UUID) ([]QueueItem, error) {
	if err := AuthorizeRequestResource(ctx, s.store, resourceID, false); err != nil {
		return nil, err
	}
	rows, err := s.store.GetQueueForResource(ctx, resourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue for resource: %w", err)
	}

	queue := make([]QueueItem, len(rows))
	for i, row := range rows {
		var duration *string
		if row.Duration.Valid {
			micro := row.Duration.Microseconds
			days := row.Duration.Days
			months := row.Duration.Months
			totalMicros := micro + int64(days)*24*3600*1000000 + int64(months)*30*24*3600*1000000
			d := time.Duration(totalMicros * 1000)
			s := d.String()
			duration = &s
		}

		var startTime int64
		if row.StartTime.Valid {
			startTime = row.StartTime.Int64
		}

		var createdAt int64
		if row.CreatedAt.Valid {
			createdAt = row.CreatedAt.Int64
		}

		var queuePosition int32
		if row.QueuePosition.Valid {
			queuePosition = row.QueuePosition.Int32
		}

		queue[i] = QueueItem{
			ID:             row.ID,
			UserID:         row.UserID,
			Status:         row.Status.String,
			QueuePosition:  queuePosition,
			StartTime:      startTime,
			CreatedAt:      createdAt,
			UserName:       row.UserName,
			UserEmail:      row.UserEmail,
			Duration:       duration,
			LeaseExpiresAt: row.LeaseExpiresAt,
			HardEndTime:    row.HardEndTime,
		}
	}

	return queue, nil
}

func (s *reservationService) ExpireReservations(ctx context.Context) error {
	expired, err := s.store.FindExpiredActiveReservations(ctx)
	if err != nil {
		return err
	}
	for _, res := range expired {
		var reclaimed []db.ClaimctlReservation
		var promoted *db.ClaimctlReservation
		err = s.store.ExecTx(ctx, func(q db.Querier) error {
			locked, e := q.TryWorkerTransactionLock(ctx, "expiry:"+res.ResourceID.String())
			if e != nil || !locked {
				return e
			}
			if _, e := q.AcquireResourceLock(ctx, res.ResourceID); e != nil {
				return e
			}
			reclaimed, e = q.ReclaimResourceReservations(ctx, res.ResourceID)
			if e != nil || len(reclaimed) == 0 {
				return e
			}
			prior, priorErr := q.FindActiveReservationByResource(ctx, res.ResourceID)
			if e = promoteLiveQueue(ctx, q, res.ResourceID); e != nil {
				return e
			}
			after, afterErr := q.FindActiveReservationByResource(ctx, res.ResourceID)
			if afterErr == nil && (priorErr != nil || prior.ID != after.ID) {
				promoted = &after
			}
			return nil
		})
		if err != nil {
			return err
		}
		if promoted != nil {
			if s.notificationSvc != nil {
				s.notify(ctx, promoted.UserID, "reservation_activated", NotificationPayload{Subject: "Reservation Activated", Message: "Your reservation is now active."})
			}
			if s.realtimeSvc != nil {
				s.realtimeSvc.Broadcast(types.Event{Type: types.EventQueueUpdate, Payload: map[string]interface{}{"resource_id": promoted.ResourceID}})
			}
		}
		for _, r := range reclaimed {
			if s.notificationSvc != nil {
				s.notify(ctx, r.UserID, "reservation_expired", NotificationPayload{Subject: "Reservation Expired", Message: "Your reservation has expired."})
			}
			if s.realtimeSvc != nil {
				s.realtimeSvc.Broadcast(types.Event{Type: types.EventReservationUpdate, Payload: map[string]interface{}{"resource_id": r.ResourceID, "action": "expired"}})
			}
		}
	}
	return nil
}
