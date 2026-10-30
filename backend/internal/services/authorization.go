package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/thetaqitahmid/claimctl/internal/db"
)

var ErrAccessDenied = errors.New("access denied")

type Authorization struct {
	User      db.ClaimctlUser
	TokenID   uuid.UUID
	Scopes    []string
	SpaceIDs  []uuid.UUID
	ExpiresAt time.Time
}

type authorizationKey struct{}

func BindAuthorization(ctx interface{ SetUserValue(any, any) }, auth Authorization) {
	ctx.SetUserValue(authorizationKey{}, auth)
}

func WithAuthorization(ctx context.Context, auth Authorization) context.Context {
	return context.WithValue(ctx, authorizationKey{}, auth)
}

func AuthorizationFromContext(ctx context.Context) (Authorization, bool) {
	auth, ok := ctx.Value(authorizationKey{}).(Authorization)
	return auth, ok
}

func CredentialVersion(passwordHash string) string {
	digest := sha256.Sum256([]byte(passwordHash))
	return hex.EncodeToString(digest[:])
}

func (a Authorization) AllowsSpace(spaceID uuid.UUID) bool {
	if a.TokenID == uuid.Nil || len(a.SpaceIDs) == 0 {
		return true
	}
	for _, id := range a.SpaceIDs {
		if id == spaceID {
			return true
		}
	}
	return false
}

func AuthorizeSpace(ctx context.Context, q db.Querier, spaceID, userID uuid.UUID, isAdmin bool) error {
	if userID == uuid.Nil || spaceID == uuid.Nil {
		return ErrAccessDenied
	}
	if auth, ok := AuthorizationFromContext(ctx); ok {
		if auth.User.ID != userID || auth.User.Status != "active" || !auth.AllowsSpace(spaceID) {
			return ErrAccessDenied
		}
		isAdmin = auth.User.Role == "admin"
	}
	if isAdmin {
		return nil
	}
	allowed, err := q.HasSpacePermission(ctx, db.HasSpacePermissionParams{ID: spaceID, UserID: pgtype.UUID{Bytes: userID, Valid: true}})
	if err != nil || !allowed {
		return ErrAccessDenied
	}
	return nil
}

func AuthorizeRequestReservation(ctx context.Context, q db.Querier, reservationID uuid.UUID, adminOnly bool) error {
	auth, ok := AuthorizationFromContext(ctx)
	if !ok {
		return nil
	}
	if adminOnly && auth.User.Role != "admin" {
		return ErrAccessDenied
	}
	reservation, err := q.FindReservationById(ctx, reservationID)
	if err != nil || (reservation.UserID != auth.User.ID && auth.User.Role != "admin") {
		return ErrAccessDenied
	}
	return AuthorizeRequestResource(ctx, q, reservation.ResourceID, false)
}

// Request principals are required at the HTTP boundary; background workers use
// trusted contexts without a principal to execute scheduled work.
func AuthorizeRequestResource(ctx context.Context, q db.Querier, resourceID uuid.UUID, adminOnly bool) error {
	auth, ok := AuthorizationFromContext(ctx)
	if !ok {
		return nil
	}
	if adminOnly && auth.User.Role != "admin" {
		return ErrAccessDenied
	}
	_, err := AuthorizeResource(ctx, q, resourceID, auth.User.ID, auth.User.Role == "admin")
	return err
}

func AuthorizeResource(ctx context.Context, q db.Querier, resourceID, userID uuid.UUID, isAdmin bool) (*db.ClaimctlResource, error) {
	resource, err := q.FindResourceById(ctx, resourceID)
	if err != nil {
		return nil, ErrAccessDenied
	}
	if err := AuthorizeSpace(ctx, q, resource.SpaceID, userID, isAdmin); err != nil {
		return nil, err
	}
	return &resource, nil
}
