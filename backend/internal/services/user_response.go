package services

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/thetaqitahmid/claimctl/internal/db"
)

type PublicUserResponse struct {
	ID           uuid.UUID          `json:"id"`
	Email        string             `json:"email"`
	Name         string             `json:"name"`
	Role         string             `json:"role"`
	Status       string             `json:"status"`
	AuthProvider string             `json:"authProvider"`
	CreatedAt    pgtype.Timestamptz `json:"createdAt"`
	UpdatedAt    pgtype.Timestamptz `json:"updatedAt"`
}

type AdminUserResponse struct {
	PublicUserResponse
	LastLogin           pgtype.Timestamptz `json:"lastLogin"`
	FailedLoginAttempts int32              `json:"failedLoginAttempts"`
	LockedUntil         pgtype.Timestamptz `json:"lockedUntil"`
	SlackDestination    pgtype.Text        `json:"slackDestination"`
	TeamsWebhookURL     pgtype.Text        `json:"teamsWebhookUrl"`
	NotificationEmail   pgtype.Text        `json:"notificationEmail"`
	OIDCSubject         pgtype.Text        `json:"oidcSubject"`
}

func adminUserResponse(user db.ClaimctlUser) *AdminUserResponse {
	return &AdminUserResponse{
		PublicUserResponse: PublicUserResponse{ID: user.ID, Email: user.Email, Name: user.Name, Role: user.Role, Status: user.Status, AuthProvider: user.AuthProvider, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt},
		LastLogin:          user.LastLogin, FailedLoginAttempts: user.FailedLoginAttempts, LockedUntil: user.LockedUntil,
		SlackDestination: user.SlackDestination, TeamsWebhookURL: user.TeamsWebhookUrl, NotificationEmail: user.NotificationEmail, OIDCSubject: user.OidcSubject,
	}
}
