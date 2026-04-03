package testutils

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/mock"

	"github.com/thetaqitahmid/claimctl/internal/db"
	types "github.com/thetaqitahmid/claimctl/internal/types"
)

// MockQuerier implements the db.Querier interface for testing
type MockQuerier struct {
	mock.Mock
}

func (m *MockQuerier) HasSpacePermission(ctx context.Context, arg db.HasSpacePermissionParams) (bool, error) {
	args := m.Called(ctx, arg)
	return args.Bool(0), args.Error(1)
}

func (m *MockQuerier) ExecTx(ctx context.Context, fn func(db.Querier) error) error {
	return fn(m)
}

// MockReservationHistoryService implements the ReservationHistoryService interface for testing
type MockReservationHistoryService struct {
	mock.Mock
}

func (m *MockReservationHistoryService) AddManualHistoryLog(ctx context.Context, req db.AddReservationHistoryLogParams) (*db.ClaimctlReservationHistory, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.ClaimctlReservationHistory), args.Error(1)
}

func (m *MockReservationHistoryService) GetRecentHistoryByAction(ctx context.Context, action string, limit int32) (*[]db.GetRecentHistoryByActionRow, error) {
	args := m.Called(ctx, action, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]db.GetRecentHistoryByActionRow), args.Error(1)
}

func (m *MockReservationHistoryService) GetUserHistory(ctx context.Context, userID uuid.UUID) (*[]db.GetUserReservationHistoryRow, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]db.GetUserReservationHistoryRow), args.Error(1)
}

func (m *MockReservationHistoryService) GetResourceHistory(ctx context.Context, resourceID uuid.UUID) (*[]db.GetResourceReservationHistoryRow, error) {
	args := m.Called(ctx, resourceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]db.GetResourceReservationHistoryRow), args.Error(1)
}

// Test fixtures
func CreateTestUser(id uuid.UUID, email, name string, admin bool) db.ClaimctlUser {
	role := "user"
	if admin {
		role = "admin"
	}
	return db.ClaimctlUser{
		ID:        id,
		Email:     email,
		Name:      name,
		Password:  "$2a$10$hashedPassword",
		Role:      role,
		Status:    "active",
		LastLogin: pgtype.Timestamptz{Valid: false},
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
}

func CreateTestAdminUser(id uuid.UUID) db.ClaimctlUser {
	return db.ClaimctlUser{
		ID:       id,
		Email:    "admin@test.com",
		Name:     "Admin User",
		Password: "$2a$10$hashedPassword",

		Role:      "admin",
		Status:    "active",
		LastLogin: pgtype.Timestamptz{Valid: false},
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
}

func CreateTestResource(id uuid.UUID, name, resourceType string, spaceID uuid.UUID) db.ClaimctlResource {
	return db.ClaimctlResource{
		ID:        id,
		Name:      name,
		Type:      resourceType,
		Labels:    types.JSONBArray{`{"tag": "test"}`},
		CreatedAt: pgtype.Int8{Int64: time.Now().Unix(), Valid: true},
		UpdatedAt: pgtype.Int8{Int64: time.Now().Unix(), Valid: true},
		SpaceID:   spaceID,
	}
}

func CreateTestSpace(id uuid.UUID, name, description string) db.ClaimctlSpace {
	return db.ClaimctlSpace{
		ID:          id,
		Name:        name,
		Description: pgtype.Text{String: description, Valid: description != ""},
		CreatedAt:   pgtype.Int8{Int64: time.Now().Unix(), Valid: true},
		UpdatedAt:   pgtype.Int8{Int64: time.Now().Unix(), Valid: true},
	}
}

func CreateTestReservation(id, resourceID, userID uuid.UUID, status string, queuePos int32) db.ClaimctlReservation {
	return db.ClaimctlReservation{
		ID:            id,
		ResourceID:    resourceID,
		UserID:        userID,
		Status:        pgtype.Text{String: status, Valid: true},
		QueuePosition: pgtype.Int4{Int32: queuePos, Valid: queuePos > 0},
		StartTime:     pgtype.Int8{Valid: false},
		EndTime:       pgtype.Int8{Valid: false},
		CreatedAt:     pgtype.Int8{Int64: time.Now().Unix(), Valid: true},
		UpdatedAt:     pgtype.Int8{Int64: time.Now().Unix(), Valid: true},
	}
}

func CreateTestReservationHistory(id, resourceID, reservationID, userID uuid.UUID, action string) db.ClaimctlReservationHistory {
	return db.ClaimctlReservationHistory{
		ID:            id,
		ResourceID:    resourceID,
		ReservationID: pgtype.UUID{Bytes: reservationID, Valid: true},
		Action:        action,
		UserID:        userID,
		Timestamp:     pgtype.Int8{Int64: time.Now().Unix(), Valid: true},
		Details:       types.JSONB{"test": "data"},
	}
}

// Helper functions for testing
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func AssertError(t *testing.T, err error, expectedMsg string) {
	t.Helper()
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if expectedMsg != "" && err.Error() != expectedMsg {
		t.Fatalf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func AssertEqual[T comparable](t *testing.T, expected, actual T) {
	t.Helper()
	if expected != actual {
		t.Fatalf("Expected %v, got %v", expected, actual)
	}
}

func AssertNotNil[T any](t *testing.T, value *T) {
	t.Helper()
	if value == nil {
		t.Fatal("Expected non-nil value")
	}
}

// Time helpers
func FixedTime() time.Time {
	return time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
}

func FixedTimestamp() int64 {
	return FixedTime().Unix()
}

// Context helper
func TestContext() context.Context {
	return context.Background()
}

// Helper for creating valid pgtype.Int4
func ValidInt4(value int32) pgtype.Int4 {
	return pgtype.Int4{Int32: value, Valid: true}
}

// Helper for generating deterministic testing UUIDs from small ints
func TestUUID(id int32) uuid.UUID {
	var u uuid.UUID
	u[12] = byte(id >> 24)
	u[13] = byte(id >> 16)
	u[14] = byte(id >> 8)
	u[15] = byte(id)
	return u
}
