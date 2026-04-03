package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/thetaqitahmid/claimctl/internal/db"
)

type GroupService interface {
	CreateGroup(ctx context.Context, name, description string) (*db.ClaimctlGroup, error)
	GetGroup(ctx context.Context, id uuid.UUID) (*db.ClaimctlGroup, error)
	ListGroups(ctx context.Context) ([]db.ClaimctlGroup, error)
	UpdateGroup(ctx context.Context, id uuid.UUID, name, description string) (*db.ClaimctlGroup, error)
	DeleteGroup(ctx context.Context, id uuid.UUID) error

	// Member Management
	AddUserToGroup(ctx context.Context, groupID, userID uuid.UUID) error
	RemoveUserFromGroup(ctx context.Context, groupID, userID uuid.UUID) error
	ListGroupMembers(ctx context.Context, groupID uuid.UUID) ([]db.ListGroupMembersRow, error)
	GetUserGroups(ctx context.Context, userID uuid.UUID) ([]db.ClaimctlGroup, error)
}

type groupService struct {
	db db.Querier
}

func NewGroupService(db db.Querier) GroupService {
	return &groupService{db: db}
}

func (s *groupService) CreateGroup(ctx context.Context, name, description string) (*db.ClaimctlGroup, error) {
	if name == "" {
		return nil, fmt.Errorf("group name cannot be empty")
	}

	existing, _ := s.db.GetGroupByName(ctx, name)
	if existing.ID != uuid.Nil {
		return nil, fmt.Errorf("group with name '%s' already exists", name)
	}

	descriptionText := pgtype.Text{String: description, Valid: description != ""}

	group, err := s.db.CreateGroup(ctx, db.CreateGroupParams{
		Name:        name,
		Description: descriptionText,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}
	return &group, nil
}

func (s *groupService) GetGroup(ctx context.Context, id uuid.UUID) (*db.ClaimctlGroup, error) {
	group, err := s.db.GetGroup(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("group not found: %w", err)
	}
	return &group, nil
}

func (s *groupService) ListGroups(ctx context.Context) ([]db.ClaimctlGroup, error) {
	groups, err := s.db.ListGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}
	return groups, nil
}

func (s *groupService) UpdateGroup(ctx context.Context, id uuid.UUID, name, description string) (*db.ClaimctlGroup, error) {
	current, err := s.db.GetGroup(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("group not found: %w", err)
	}

	if name != "" && name != current.Name {
		existing, _ := s.db.GetGroupByName(ctx, name)
		if existing.ID != uuid.Nil {
			return nil, fmt.Errorf("group name '%s' already taken", name)
		}
	} else {
		name = current.Name
	}

	descriptionText := pgtype.Text{String: description, Valid: description != ""}

	group, err := s.db.UpdateGroup(ctx, db.UpdateGroupParams{
		ID:          id,
		Name:        name,
		Description: descriptionText,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update group: %w", err)
	}
	return &group, nil
}

func (s *groupService) DeleteGroup(ctx context.Context, id uuid.UUID) error {
	if err := s.db.DeleteGroup(ctx, id); err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}
	return nil
}

func (s *groupService) AddUserToGroup(ctx context.Context, groupID, userID uuid.UUID) error {
	err := s.db.AddUserToGroup(ctx, db.AddUserToGroupParams{
		GroupID: groupID,
		UserID:  userID,
	})
	if err != nil {
		return fmt.Errorf("failed to add user to group: %w", err)
	}
	return nil
}

func (s *groupService) RemoveUserFromGroup(ctx context.Context, groupID, userID uuid.UUID) error {
	err := s.db.RemoveUserFromGroup(ctx, db.RemoveUserFromGroupParams{
		GroupID: groupID,
		UserID:  userID,
	})
	if err != nil {
		return fmt.Errorf("failed to remove user from group: %w", err)
	}
	return nil
}

func (s *groupService) ListGroupMembers(ctx context.Context, groupID uuid.UUID) ([]db.ListGroupMembersRow, error) {
	members, err := s.db.ListGroupMembers(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to list group members: %w", err)
	}
	return members, nil
}

func (s *groupService) GetUserGroups(ctx context.Context, userID uuid.UUID) ([]db.ClaimctlGroup, error) {
	groups, err := s.db.GetUserGroups(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user groups: %w", err)
	}
	return groups, nil
}
