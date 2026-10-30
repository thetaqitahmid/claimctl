package services

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/types"
)

func RefreshAuthorization(ctx context.Context, q db.Querier, auth Authorization) (Authorization, error) {
	if !auth.ExpiresAt.IsZero() && !auth.ExpiresAt.After(time.Now()) {
		return Authorization{}, ErrAccessDenied
	}
	user, err := q.FindUserById(ctx, auth.User.ID)
	if err != nil || user.Status != "active" || CredentialVersion(user.Password) != CredentialVersion(auth.User.Password) {
		return Authorization{}, ErrAccessDenied
	}
	auth.User = user
	if auth.TokenID != uuid.Nil {
		tokens, err := q.ListAPITokens(ctx, user.ID)
		if err != nil {
			return Authorization{}, ErrAccessDenied
		}
		found := false
		for _, token := range tokens {
			if token.ID == auth.TokenID && token.ExpiresAt.Valid && token.ExpiresAt.Time.After(time.Now()) {
				for _, scope := range token.Scopes {
					if scope == "read" {
						found = true
					}
				}
				auth.Scopes = token.Scopes
				auth.SpaceIDs = token.SpaceIds
				auth.ExpiresAt = token.ExpiresAt.Time
			}
		}
		if !found {
			return Authorization{}, ErrAccessDenied
		}
	}
	return auth, nil
}

func CanDeliverEvent(ctx context.Context, q db.Querier, auth Authorization, event types.Event) bool {
	if event.Type == "resync" {
		return event.Payload == nil
	}
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return false
	}
	var target struct {
		ResourceID uuid.UUID `json:"resource_id"`
		SpaceID    uuid.UUID `json:"space_id"`
	}
	if json.Unmarshal(payload, &target) != nil || target.ResourceID == uuid.Nil {
		return false
	}
	ctx = WithAuthorization(ctx, auth)
	if target.SpaceID == uuid.Nil {
		_, err = AuthorizeResource(ctx, q, target.ResourceID, auth.User.ID, auth.User.Role == "admin")
		return err == nil
	}
	resource, err := q.FindResourceById(ctx, target.ResourceID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return false
	}
	if err == nil && resource.SpaceID != target.SpaceID {
		return false
	}
	return AuthorizeSpace(ctx, q, target.SpaceID, auth.User.ID, auth.User.Role == "admin") == nil
}
