package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/thetaqitahmid/claimctl/internal/db"
)

type ReservationHistoryService interface {
	AddManualHistoryLog(ctx context.Context, req AddHistoryLogRequest) (*db.ClaimctlReservationHistory, error)
	GetRecentHistoryByAction(ctx context.Context, action string, limit int32) (*[]db.GetRecentHistoryByActionRow, error)
	GetUserHistory(ctx context.Context, userID uuid.UUID) (*[]db.GetUserReservationHistoryRow, error)
	GetResourceHistory(ctx context.Context, resourceID uuid.UUID) (*[]db.GetResourceReservationHistoryRow, error)
}

type AddHistoryLogRequest = db.AddReservationHistoryLogParams

type reservationHistoryService struct {
	db db.Querier
}

func NewReservationHistoryService(db db.Querier) ReservationHistoryService {
	return &reservationHistoryService{db: db}
}

func (s *reservationHistoryService) AddManualHistoryLog(ctx context.Context, req AddHistoryLogRequest) (*db.ClaimctlReservationHistory, error) {
	// Validate that the resource exists
	_, err := s.db.FindResourceById(ctx, req.ResourceID)
	if err != nil {
		return nil, fmt.Errorf("resource with ID %s not found: %w", req.ResourceID, err)
	}

	// Validate that the user exists
	_, err = s.db.FindUserById(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("user with ID %s not found: %w", req.UserID, err)
	}

	history, err := s.db.AddReservationHistoryLog(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to add reservation history log: %w", err)
	}

	return &history, nil
}

func (s *reservationHistoryService) GetRecentHistoryByAction(ctx context.Context, action string, limit int32) (*[]db.GetRecentHistoryByActionRow, error) {
	params := db.GetRecentHistoryByActionParams{
		Action: action,
		Limit:  limit,
	}
	history, err := s.db.GetRecentHistoryByAction(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent history for action %s: %w", action, err)
	}
	return &history, nil
}

func (s *reservationHistoryService) GetUserHistory(ctx context.Context, userID uuid.UUID) (*[]db.GetUserReservationHistoryRow, error) {
	history, err := s.db.GetUserReservationHistory(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user history: %w", err)
	}
	return &history, nil
}

func (s *reservationHistoryService) GetResourceHistory(ctx context.Context, resourceID uuid.UUID) (*[]db.GetResourceReservationHistoryRow, error) {
	history, err := s.db.GetResourceReservationHistory(ctx, resourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource history: %w", err)
	}
	if history == nil {
		history = []db.GetResourceReservationHistoryRow{}
	}
	return &history, nil
}
