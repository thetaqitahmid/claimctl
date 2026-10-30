package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

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
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	user, err := s.db.FindUserById(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to find user for notification: %w", err)
	}

	// Iterate over all available dispatchers
	for channel, dispatcher := range s.dispatchers {
		pref, err := s.prefs.GetPreference(ctx, userID, event, channel)
		if err != nil {
			// No row means the channel is disabled (opt-in default);
			// anything else is a real error worth surfacing.
			if !isNotFound(err) {
				slog.Error("Failed to load notification preference",
					"user_id", userID, "event", event, "channel", channel, "error", err)
			}
			continue
		}

		if !pref.Enabled {
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

		if err := dispatcher.Dispatch(ctx, recipient, payload); err != nil {
			slog.Error("Failed to dispatch notification", "channel", dispatcher.Type())
		}
	}

	return nil
}
