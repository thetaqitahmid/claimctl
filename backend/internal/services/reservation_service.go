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
	"github.com/thetaqitahmid/claimctl/internal/utils"
)

type QueueItem struct {
	ID uuid.UUID   `json:"id"`
	UserID uuid.UUID   `json:"userId"`
	Status        string  `json:"status"`
	QueuePosition int32   `json:"queuePosition"`
	StartTime     int64   `json:"startTime"`
	CreatedAt     int64   `json:"createdAt"`
	UserName      string  `json:"userName"`
	UserEmail     string  `json:"userEmail"`
	Duration      *string `json:"duration,omitempty"`
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
	store           db.Store
	historySvc      ReservationHistoryService
	webhookSvc      *WebhookService
	realtimeSvc     RealtimeService
	notificationSvc NotificationService
}

func NewReservationService(store db.Store, historySvc ReservationHistoryService, webhookSvc *WebhookService, realtimeSvc RealtimeService, notificationSvc NotificationService) ReservationService {
	return &reservationService{
		store:           store,
		historySvc:      historySvc,
		webhookSvc:      webhookSvc,
		realtimeSvc:     realtimeSvc,
		notificationSvc: notificationSvc,
	}
}

func (s *reservationService) CreateReservation(ctx context.Context, userID uuid.UUID, resourceID uuid.UUID, duration *string) (*db.ClaimctlReservation, error) {
	var reservation db.ClaimctlReservation

	err := s.store.ExecTx(ctx, func(q db.Querier) error {
		// Acquire exclusive lock on the resource
		_, err := q.AcquireResourceLock(ctx, resourceID)
		if err != nil {
			return fmt.Errorf("resource not found or failed to lock: %w", err)
		}

		// Check if resource is under maintenance
		maintenanceStatus, err := q.GetResourceMaintenanceStatus(ctx, resourceID)
		if err == nil && maintenanceStatus.Valid && maintenanceStatus.Bool {
			return fmt.Errorf("resource is currently under maintenance and cannot be reserved")
		}

		// Check if user already has an active/pending reservation for this resource
		existingReservation, err := q.FindUserReservationForResource(ctx, db.FindUserReservationForResourceParams{
			UserID:     userID,
			ResourceID: resourceID,
		})
		if err == nil {
			return fmt.Errorf("user already has a reservation for this resource (status: %s)", existingReservation.Status.String)
		}

		// Create the reservation with proper status and queue position
		var durationInterval pgtype.Interval
		if duration != nil {
			d, parseErr := utils.ParseDurationWithDays(*duration)
			if parseErr == nil {
				durationInterval = pgtype.Interval{Microseconds: int64(d.Microseconds()), Valid: true}
				slog.Info("Parsed duration", "duration", *duration, "microseconds", durationInterval.Microseconds)
			} else {
				slog.Error("Failed to parse duration", "duration", *duration, "error", parseErr)
			}
		} else {
			slog.Info("CreateReservation called with nil duration")
		}

		createdRes, createErr := q.CreateReservation(ctx, db.CreateReservationParams{
			ResourceID: resourceID,
			UserID:     userID,
			Duration:   durationInterval,
		})
		if createErr != nil {
			return fmt.Errorf("failed to create reservation: %w", createErr)
		}
		reservation = createdRes
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Trigger Webhook
	_ = s.webhookSvc.TriggerWebhooks(ctx, resourceID, "reservation.created", reservation)

	// Send Notification
	resource, _ := s.store.FindResourceById(ctx, resourceID)
	go s.notificationSvc.Notify(ctx, userID, "reservation_created", NotificationPayload{
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

	return &reservation, nil
}

func (s *reservationService) CancelAllForResource(ctx context.Context, resourceID uuid.UUID) error {
	timestamp := time.Now().Unix()
	err := s.store.CancelAllReservationsForResource(ctx, db.CancelAllReservationsForResourceParams{
		ResourceID: resourceID,
		EndTime:    pgtype.Int8{Int64: timestamp, Valid: true},
	})
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
			"action":        "created",
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
		timestamp := time.Now().Unix()
		q.UpdateQueuePositions(ctx, db.UpdateQueuePositionsParams{
			ResourceID:    res.ResourceID,
			UpdatedAt:     pgtype.Int8{Int64: timestamp, Valid: true},
			QueuePosition: pgtype.Int4{Int32: res.QueuePosition.Int32, Valid: true},
		})

		slog.Info("[ACTIVATE] Executing ActivateReservation SQL", "reservation_id", reservationID)
		err = q.ActivateReservation(ctx, db.ActivateReservationParams{
			ID:          reservationID,
			StartTime:   pgtype.Int8{Int64: timestamp, Valid: true},
			ToTimestamp: float64(timestamp),
			UpdatedAt:   pgtype.Int8{Int64: timestamp, Valid: true},
		})
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

	// Trigger Webhook
	_ = s.webhookSvc.TriggerWebhooks(ctx, reservation.ResourceID, "reservation.activated", reservation)

	// Send Notification
	activatedResource, _ := s.store.FindResourceById(ctx, reservation.ResourceID)
	go s.notificationSvc.Notify(ctx, reservation.UserID, "reservation_activated", NotificationPayload{
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
		timestamp := time.Now().Unix()
		err = q.CompleteReservation(ctx, db.CompleteReservationParams{
			ID:        reservationID,
			EndTime:   pgtype.Int8{Int64: timestamp, Valid: true},
			UpdatedAt: pgtype.Int8{Int64: timestamp, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to complete reservation: %w", err)
		}

		reservation = res
		return nil
	})

	if err != nil {
		return err
	}

	// Trigger Webhook
	_ = s.webhookSvc.TriggerWebhooks(ctx, reservation.ResourceID, "reservation.completed", reservation)

	// Send Notification
	completedResource, _ := s.store.FindResourceById(ctx, reservation.ResourceID)
	go s.notificationSvc.Notify(ctx, reservation.UserID, "reservation_completed", NotificationPayload{
		Subject: "Reservation Completed",
		Message: fmt.Sprintf("Your reservation for resource '%s' has been completed.", completedResource.Name),
	})

	slog.Info("[RESERVATION] Completed reservation", "reservation_id: ", reservationID, "resource_id: ", reservation.ResourceID, "user_id: ", reservation.UserID)

	// Process the queue after completion to activate the next reservation
	s.ProcessQueue(ctx, reservation.ResourceID)

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
		timestamp := time.Now().Unix()
		err = q.CancelReservation(ctx, db.CancelReservationParams{
			ID:        reservationID,
			EndTime:   pgtype.Int8{Int64: timestamp, Valid: true},
			UpdatedAt: pgtype.Int8{Int64: timestamp, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to cancel reservation: %w", err)
		}

		reservation = res
		return nil
	})

	if err != nil {
		return err
	}

	// Trigger Webhook
	_ = s.webhookSvc.TriggerWebhooks(ctx, reservation.ResourceID, "reservation.cancelled", reservation)

	// Send Notification
	cancelledRes, _ := s.store.FindResourceById(ctx, reservation.ResourceID)
	go s.notificationSvc.Notify(ctx, reservation.UserID, "reservation_cancelled", NotificationPayload{
		Subject: "Reservation Cancelled",
		Message: fmt.Sprintf("Your reservation for resource '%s' has been cancelled.", cancelledRes.Name),
	})

	slog.Info("[RESERVATION] Cancelled reservation", "reservation_id", reservationID, "resource_id", reservation.ResourceID, "user_id", reservation.UserID)

	// If this was an active reservation, process the queue to activate the next one
	if reservation.Status.String == "active" {
		s.ProcessQueue(ctx, reservation.ResourceID)
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
	return reservations, nil
}

func (s *reservationService) GetResourceReservations(ctx context.Context, resourceID uuid.UUID) ([]db.FindReservationsByResourceRow, error) {
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
	reservation, err := s.store.GetNextInQueue(ctx, resourceID)
	if err != nil {
		return nil, fmt.Errorf("no reservations in queue: %w", err)
	}
	return &reservation, nil
}

func (s *reservationService) GetUserQueuePosition(ctx context.Context, userID uuid.UUID, resourceID uuid.UUID) (int64, error) {
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
	nextReservation, err := s.store.GetNextInQueue(ctx, resourceID)
	if err != nil {
		return nil
	}

	_, err = s.ActivateReservation(ctx, nextReservation.ID)

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
	timestamp := time.Now().Unix()
	// Update queue positions for all reservations that were behind the given position
	s.store.UpdateQueuePositions(ctx, db.UpdateQueuePositionsParams{
		ResourceID:    resourceID,
		UpdatedAt:     pgtype.Int8{Int64: timestamp, Valid: true},
		QueuePosition: pgtype.Int4{Int32: position, Valid: true},
	})
}

func (s *reservationService) GetQueueForResource(ctx context.Context, resourceID uuid.UUID) ([]QueueItem, error) {
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
			ID:            row.ID,
			UserID:        row.UserID,
			Status:        row.Status.String,
			QueuePosition: queuePosition,
			StartTime:     startTime,
			CreatedAt:     createdAt,
			UserName:      row.UserName,
			UserEmail:     row.UserEmail,
			Duration:      duration,
		}
	}

	return queue, nil
}

func (s *reservationService) ExpireReservations(ctx context.Context) error {
	timestamp := time.Now().Unix()

	// Find expired reservations
	expired, err := s.store.FindExpiredActiveReservations(ctx)
	if err != nil {
		return fmt.Errorf("failed to find expired reservations: %w", err)
	}

	for _, res := range expired {
		err := s.store.ExpireReservation(ctx, db.ExpireReservationParams{
			ID:        res.ID,
			UpdatedAt: pgtype.Int8{Int64: timestamp, Valid: true},
		})
		if err != nil {
			slog.Info("Failed to expire reservation", "reservation_id: ", res.ID, "error: ", err)
			continue
		}

		// Trigger Webhook
		_ = s.webhookSvc.TriggerWebhooks(ctx, res.ResourceID, "reservation.expired", res)

		// Send Notification
		expiredResource, _ := s.store.FindResourceById(ctx, res.ResourceID)
		go s.notificationSvc.Notify(ctx, res.UserID, "reservation_expired", NotificationPayload{
			Subject: "Reservation Expired",
			Message: fmt.Sprintf("Your reservation for resource '%s' has expired.", expiredResource.Name),
		})

		slog.Info("[RESERVATION] Expired reservation", "reservation_id: ", res.ID, "resource_id: ", res.ResourceID, "user_id: ", res.UserID)

		// Process queue
		s.ProcessQueue(ctx, res.ResourceID)

		// Broadcast Realtime Event
		s.realtimeSvc.Broadcast(types.Event{
			Type: types.EventReservationUpdate,
			Payload: map[string]interface{}{
				"resource_id":   res.ResourceID,
				"resource_name": expiredResource.Name,
				"action":        "expired",
			},
		})
	}

	return nil
}
