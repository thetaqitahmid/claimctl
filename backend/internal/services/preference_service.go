package services

import (
	"context"

	"github.com/google/uuid"

	"github.com/thetaqitahmid/claimctl/internal/db"
)

type PreferenceService interface {
	GetPreferences(ctx context.Context, userID uuid.UUID) ([]db.ClaimctlUserNotificationPreference, error)
	UpsertPreference(ctx context.Context, userID uuid.UUID, eventType string, channel string, enabled bool) (db.ClaimctlUserNotificationPreference, error)
	GetPreference(ctx context.Context, userID uuid.UUID, eventType string, channel string) (db.ClaimctlUserNotificationPreference, error)
}

type preferenceService struct {
	db db.Querier
}

func NewPreferenceService(db db.Querier) PreferenceService {
	return &preferenceService{db: db}
}

func (s *preferenceService) GetPreferences(ctx context.Context, userID uuid.UUID) ([]db.ClaimctlUserNotificationPreference, error) {
	return s.db.GetUserPreferences(ctx, userID)
}

func (s *preferenceService) UpsertPreference(ctx context.Context, userID uuid.UUID, eventType string, channel string, enabled bool) (db.ClaimctlUserNotificationPreference, error) {
	return s.db.UpsertPreference(ctx, db.UpsertPreferenceParams{
		UserID:    userID,
		EventType: eventType,
		Channel:   channel,
		Enabled:   enabled,
	})
}

func (s *preferenceService) GetPreference(ctx context.Context, userID uuid.UUID, eventType string, channel string) (db.ClaimctlUserNotificationPreference, error) {
	return s.db.GetUserPreference(ctx, db.GetUserPreferenceParams{
		UserID:    userID,
		EventType: eventType,
		Channel:   channel,
	})
}
