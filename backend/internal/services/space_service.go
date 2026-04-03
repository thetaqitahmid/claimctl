package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/thetaqitahmid/claimctl/internal/db"
)

type SpaceService interface {
	CreateSpace(ctx context.Context, name, description string) (*db.ClaimctlSpace, error)
	GetSpace(ctx context.Context, id uuid.UUID, userID uuid.UUID, isAdmin bool) (*db.ClaimctlSpace, error)
	GetAllSpaces(ctx context.Context) ([]db.ClaimctlSpace, error)
	GetAllSpacesForUser(ctx context.Context, userID uuid.UUID, isAdmin bool) ([]db.ClaimctlSpace, error)
	UpdateSpace(ctx context.Context, id uuid.UUID, name, description string) (*db.ClaimctlSpace, error)
	DeleteSpace(ctx context.Context, id uuid.UUID) error
	EnsureDefaultSpaceExists(ctx context.Context) error

	// Permissions
	GetSpacePermissions(ctx context.Context, spaceID uuid.UUID) ([]db.GetSpacePermissionsRow, error)
	AddPermission(ctx context.Context, spaceID uuid.UUID, groupID, userID *uuid.UUID) error
	RemovePermission(ctx context.Context, spaceID uuid.UUID, groupID, userID *uuid.UUID) error
}

type spaceService struct {
	db db.Querier
}

func NewSpaceService(db db.Querier) SpaceService {
	return &spaceService{db: db}
}

func (s *spaceService) CreateSpace(ctx context.Context, name, description string) (*db.ClaimctlSpace, error) {
	if name == "" {
		return nil, fmt.Errorf("space name cannot be empty")
	}
	// Check for uniqueness
	_, err := s.db.GetSpaceByName(ctx, name)
	if err == nil {
		return nil, fmt.Errorf("space with name '%s' already exists", name)
	}

	descriptionText := pgtype.Text{String: description, Valid: description != ""}
	space, err := s.db.CreateSpace(ctx, db.CreateSpaceParams{
		Name:        name,
		Description: descriptionText,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create space: %w", err)
	}
	return &space, nil
}

func (s *spaceService) GetSpace(ctx context.Context, id uuid.UUID, userID uuid.UUID, isAdmin bool) (*db.ClaimctlSpace, error) {
	if isAdmin {
		space, err := s.db.GetSpace(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("space not found: %w", err)
		}
		return &space, nil
	}

	hasPerm, err := s.db.HasSpacePermission(ctx, db.HasSpacePermissionParams{
		ID:     id,
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
	})
	if err != nil || !hasPerm {
		return nil, fmt.Errorf("access denied or space not found")
	}

	space, err := s.db.GetSpace(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("space not found: %w", err)
	}
	return &space, nil
}

func (s *spaceService) GetAllSpaces(ctx context.Context) ([]db.ClaimctlSpace, error) {
	spaces, err := s.db.ListSpaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list spaces: %w", err)
	}
	return spaces, nil
}

func (s *spaceService) GetAllSpacesForUser(ctx context.Context, userID uuid.UUID, isAdmin bool) ([]db.ClaimctlSpace, error) {
	if isAdmin {
		return s.GetAllSpaces(ctx)
	}

	spaces, err := s.db.ListSpacesForUser(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list spaces for user: %w", err)
	}
	return spaces, nil
}

func (s *spaceService) UpdateSpace(ctx context.Context, id uuid.UUID, name, description string) (*db.ClaimctlSpace, error) {
	// Check if exists
	_, err := s.db.GetSpace(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("space not found: %w", err)
	}

	// Check name uniqueness if changed
	existing, err := s.db.GetSpaceByName(ctx, name)
	if err == nil && existing.ID != id {
		return nil, fmt.Errorf("space name '%s' already taken", name)
	}

	descriptionText := pgtype.Text{String: description, Valid: description != ""}
	space, err := s.db.UpdateSpace(ctx, db.UpdateSpaceParams{
		ID:          id,
		Name:        name,
		Description: descriptionText,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update space: %w", err)
	}
	return &space, nil
}

func (s *spaceService) DeleteSpace(ctx context.Context, id uuid.UUID) error {
	space, err := s.db.GetSpace(ctx, id)
	if err != nil {
		return fmt.Errorf("space not found: %w", err)
	}

	if space.Name == "Default Space" {
		return fmt.Errorf("cannot delete the Default Space")
	}

	if err := s.db.DeleteSpace(ctx, id); err != nil {
		return fmt.Errorf("failed to delete space: %w", err)
	}
	return nil
}

func (s *spaceService) EnsureDefaultSpaceExists(ctx context.Context) error {
	_, err := s.db.GetSpaceByName(ctx, "Default Space")
	if err == nil {
		// Default space already exists
		fmt.Println("Default Space already exists, skipping creation")
		return nil
	}

	// Create the default space
	fmt.Println("Default Space not found, creating it")
	_, err = s.CreateSpace(ctx, "Default Space", "The default space for resources")
	if err != nil {
		return fmt.Errorf("failed to create Default Space: %w", err)
	}

	fmt.Println("Successfully created Default Space")
	return nil
}

// Permissions

func (s *spaceService) GetSpacePermissions(ctx context.Context, spaceID uuid.UUID) ([]db.GetSpacePermissionsRow, error) {
	permissions, err := s.db.GetSpacePermissions(ctx, pgtype.UUID{Bytes: spaceID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get space permissions: %w", err)
	}
	return permissions, nil
}

func (s *spaceService) AddPermission(ctx context.Context, spaceID uuid.UUID, groupID, userID *uuid.UUID) error {
	if groupID == nil && userID == nil {
		return fmt.Errorf("either groupID or userID must be provided")
	}
	if groupID != nil && userID != nil {
		return fmt.Errorf("cannot provide both groupID and userID")
	}

	params := db.AddSpacePermissionParams{
		SpaceID: pgtype.UUID{Bytes: spaceID, Valid: true},
	}
	if groupID != nil {
		params.GroupID = pgtype.UUID{Bytes: *groupID, Valid: true}
	} else {
		params.UserID = pgtype.UUID{Bytes: *userID, Valid: true}
	}

	if err := s.db.AddSpacePermission(ctx, params); err != nil {
		return fmt.Errorf("failed to add permission: %w", err)
	}
	return nil
}

func (s *spaceService) RemovePermission(ctx context.Context, spaceID uuid.UUID, groupID, userID *uuid.UUID) error {
	if groupID == nil && userID == nil {
		return fmt.Errorf("either groupID or userID must be provided")
	}

	params := db.RemoveSpacePermissionParams{
		SpaceID: pgtype.UUID{Bytes: spaceID, Valid: true},
	}
	if groupID != nil {
		params.GroupID = pgtype.UUID{Bytes: *groupID, Valid: true}
	} else {
		params.UserID = pgtype.UUID{Bytes: *userID, Valid: true}
	}

	if err := s.db.RemoveSpacePermission(ctx, params); err != nil {
		return fmt.Errorf("failed to remove permission: %w", err)
	}
	return nil
}
