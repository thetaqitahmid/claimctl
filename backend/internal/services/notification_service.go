package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/thetaqitahmid/claimctl/internal/db"
)

type NotificationService interface {
	Notify(ctx context.Context, userID uuid.UUID, event string, payload NotificationPayload) error
}

type notificationService struct {
	db          db.Querier
	dispatchers map[string]NotificationDispatcher
	prefs       PreferenceService
}

func NewNotificationService(db db.Querier, dispatchers []NotificationDispatcher, prefs PreferenceService) NotificationService {
	dMap := make(map[string]NotificationDispatcher)
	for _, d := range dispatchers {
		dMap[d.Type()] = d
	}
	return &notificationService{
		db:          db,
		dispatchers: dMap,
		prefs:       prefs,
	}
}

func (s *notificationService) Notify(ctx context.Context, userID uuid.UUID, event string, payload NotificationPayload) error {
	user, err := s.db.FindUserById(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to find user for notification: %w", err)
	}

	// Iterate over all available dispatchers
	for channel, dispatcher := range s.dispatchers {
		pref, err := s.prefs.GetPreference(ctx, userID, event, channel)
		enabled := false

		if err == nil {
			enabled = pref.Enabled
		}

		if !enabled {
			continue
		}

		// Determine Recipient based on Channel
		var recipient string
		switch channel {
		case "email":
			if user.NotificationEmail.Valid && user.NotificationEmail.String != "" {
				recipient = user.NotificationEmail.String
			} else {
				recipient = user.Email
			}
		case "slack":
			if user.SlackDestination.Valid {
				recipient = user.SlackDestination.String
			}
		case "teams":
			if user.TeamsWebhookUrl.Valid {
				recipient = user.TeamsWebhookUrl.String
			}
		}

		if recipient == "" {
			// Channel enabled but no destination configured
			slog.Warn("Notification channel enabled but no destination found", "user_id", userID, "channel", channel)
			continue
		}

		// Dispatch
		go func(d NotificationDispatcher, r string) {
			err := d.Dispatch(context.Background(), r, payload)
			if err != nil {
				slog.Error("Failed to dispatch notification", "channel", d.Type(), "recipient", r, "error", err)
			} else {
				slog.Info("Sent notification", "channel", d.Type(), "recipient", r)
			}
		}(dispatcher, recipient)
	}

	return nil
}
