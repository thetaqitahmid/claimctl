package testutils

import (
	"context"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/thetaqitahmid/claimctl/internal/db"
)

// Mock implementations for all db.Querier methods
// Add methods as needed for testing

func (m *MockQuerier) CreateUser(ctx context.Context, arg db.CreateUserParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) CleanupOldWebhookLogs(ctx context.Context, createdAt pgtype.Timestamp) error {
	args := m.Called(ctx, createdAt)
	return args.Error(0)
}

func (m *MockQuerier) FindUserByEmail(ctx context.Context, email string) (db.ClaimctlUser, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(db.ClaimctlUser), args.Error(1)
}

func (m *MockQuerier) FindUserById(ctx context.Context, id uuid.UUID) (db.ClaimctlUser, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(db.ClaimctlUser), args.Error(1)
}

func (m *MockQuerier) VerifyUserEmailIsUnique(ctx context.Context, email string) (int64, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockQuerier) GetPasswordById(ctx context.Context, id uuid.UUID) (string, error) {
	args := m.Called(ctx, id)
	return args.String(0), args.Error(1)
}

func (m *MockQuerier) UpdateUserLastLogin(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockQuerier) FindAllUsers(ctx context.Context) ([]db.ClaimctlUser, error) {
	args := m.Called(ctx)
	return args.Get(0).([]db.ClaimctlUser), args.Error(1)
}

func (m *MockQuerier) UpdateUserById(ctx context.Context, arg db.UpdateUserByIdParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) DeleteUserById(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockQuerier) CountAdminUsers(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockQuerier) UpdateUserFailedLoginAttempts(ctx context.Context, arg db.UpdateUserFailedLoginAttemptsParams) (int32, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(int32), args.Error(1)
}

func (m *MockQuerier) ResetUserFailedLoginAttempts(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockQuerier) CreateNewResource(ctx context.Context, arg db.CreateNewResourceParams) (db.ClaimctlResource, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.ClaimctlResource), args.Error(1)
}

func (m *MockQuerier) FindResourceById(ctx context.Context, id uuid.UUID) (db.ClaimctlResource, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(db.ClaimctlResource), args.Error(1)
}

func (m *MockQuerier) FindAllResources(ctx context.Context) ([]db.ClaimctlResource, error) {
	args := m.Called(ctx)
	return args.Get(0).([]db.ClaimctlResource), args.Error(1)
}

func (m *MockQuerier) VerifyResourceNameIsUnique(ctx context.Context, name string) (int64, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockQuerier) UpdateResourceById(ctx context.Context, arg db.UpdateResourceByIdParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) DeleteResourceById(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockQuerier) GetResourceReservationStatus(ctx context.Context, id uuid.UUID) (db.GetResourceReservationStatusRow, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(db.GetResourceReservationStatusRow), args.Error(1)
}

func (m *MockQuerier) GetAllResourcesWithReservationStatus(ctx context.Context) ([]db.GetAllResourcesWithReservationStatusRow, error) {
	args := m.Called(ctx)
	return args.Get(0).([]db.GetAllResourcesWithReservationStatusRow), args.Error(1)
}

func (m *MockQuerier) CreateSpace(ctx context.Context, arg db.CreateSpaceParams) (db.ClaimctlSpace, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.ClaimctlSpace), args.Error(1)
}

func (m *MockQuerier) GetSpace(ctx context.Context, id uuid.UUID) (db.ClaimctlSpace, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(db.ClaimctlSpace), args.Error(1)
}

func (m *MockQuerier) ListSpaces(ctx context.Context) ([]db.ClaimctlSpace, error) {
	args := m.Called(ctx)
	return args.Get(0).([]db.ClaimctlSpace), args.Error(1)
}

func (m *MockQuerier) GetSpaceByName(ctx context.Context, name string) (db.ClaimctlSpace, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(db.ClaimctlSpace), args.Error(1)
}

func (m *MockQuerier) UpdateSpace(ctx context.Context, arg db.UpdateSpaceParams) (db.ClaimctlSpace, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.ClaimctlSpace), args.Error(1)
}

func (m *MockQuerier) DeleteSpace(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockQuerier) CreateReservation(ctx context.Context, arg db.CreateReservationParams) (db.ClaimctlReservation, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.ClaimctlReservation), args.Error(1)
}

func (m *MockQuerier) FindReservationById(ctx context.Context, id uuid.UUID) (db.ClaimctlReservation, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(db.ClaimctlReservation), args.Error(1)
}

func (m *MockQuerier) FindUserReservationForResource(ctx context.Context, arg db.FindUserReservationForResourceParams) (db.ClaimctlReservation, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.ClaimctlReservation), args.Error(1)
}

func (m *MockQuerier) ActivateReservation(ctx context.Context, arg db.ActivateReservationParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) CompleteReservation(ctx context.Context, arg db.CompleteReservationParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) CancelReservation(ctx context.Context, arg db.CancelReservationParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) FindActiveReservationByResource(ctx context.Context, resourceID uuid.UUID) (db.ClaimctlReservation, error) {
	args := m.Called(ctx, resourceID)
	return args.Get(0).(db.ClaimctlReservation), args.Error(1)
}

func (m *MockQuerier) FindUserActiveReservations(ctx context.Context, userID uuid.UUID) ([]db.FindUserActiveReservationsRow, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]db.FindUserActiveReservationsRow), args.Error(1)
}

func (m *MockQuerier) FindReservationsByResource(ctx context.Context, resourceID uuid.UUID) ([]db.FindReservationsByResourceRow, error) {
	args := m.Called(ctx, resourceID)
	return args.Get(0).([]db.FindReservationsByResourceRow), args.Error(1)
}

func (m *MockQuerier) FindAllReservations(ctx context.Context) ([]db.FindAllReservationsRow, error) {
	args := m.Called(ctx)
	return args.Get(0).([]db.FindAllReservationsRow), args.Error(1)
}

func (m *MockQuerier) GetNextInQueue(ctx context.Context, resourceID uuid.UUID) (db.ClaimctlReservation, error) {
	args := m.Called(ctx, resourceID)
	return args.Get(0).(db.ClaimctlReservation), args.Error(1)
}

func (m *MockQuerier) GetUserQueuePosition(ctx context.Context, arg db.GetUserQueuePositionParams) (pgtype.Int4, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(pgtype.Int4), args.Error(1)
}

func (m *MockQuerier) UpdateQueuePositions(ctx context.Context, arg db.UpdateQueuePositionsParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) AddReservationHistoryLog(ctx context.Context, arg db.AddReservationHistoryLogParams) (db.ClaimctlReservationHistory, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.ClaimctlReservationHistory), args.Error(1)
}

func (m *MockQuerier) GetRecentHistoryByAction(ctx context.Context, arg db.GetRecentHistoryByActionParams) ([]db.GetRecentHistoryByActionRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]db.GetRecentHistoryByActionRow), args.Error(1)
}

// Additional missing methods
func (m *MockQuerier) CleanupOldCompletedReservations(ctx context.Context, updatedAt pgtype.Int8) error {
	args := m.Called(ctx, updatedAt)
	return args.Error(0)
}

func (m *MockQuerier) CountPendingReservations(ctx context.Context, resourceID uuid.UUID) (int64, error) {
	args := m.Called(ctx, resourceID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockQuerier) FindPendingReservationsByResource(ctx context.Context, resourceID uuid.UUID) ([]db.ClaimctlReservation, error) {
	args := m.Called(ctx, resourceID)
	return args.Get(0).([]db.ClaimctlReservation), args.Error(1)
}

func (m *MockQuerier) FindReservationsByUser(ctx context.Context, userID uuid.UUID) ([]db.FindReservationsByUserRow, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]db.FindReservationsByUserRow), args.Error(1)
}

func (m *MockQuerier) FindResourceByName(ctx context.Context, name string) (db.ClaimctlResource, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(db.ClaimctlResource), args.Error(1)
}

func (m *MockQuerier) GetReservationHistory(ctx context.Context, reservationID pgtype.UUID) ([]db.GetReservationHistoryRow, error) {
	args := m.Called(ctx, reservationID)
	return args.Get(0).([]db.GetReservationHistoryRow), args.Error(1)
}

func (m *MockQuerier) GetResourceReservationHistory(ctx context.Context, resourceID uuid.UUID) ([]db.GetResourceReservationHistoryRow, error) {
	args := m.Called(ctx, resourceID)
	return args.Get(0).([]db.GetResourceReservationHistoryRow), args.Error(1)
}

func (m *MockQuerier) GetResourceStats(ctx context.Context, resourceID uuid.UUID) (db.GetResourceStatsRow, error) {
	args := m.Called(ctx, resourceID)
	return args.Get(0).(db.GetResourceStatsRow), args.Error(1)
}

func (m *MockQuerier) GetResourceTest(ctx context.Context, id uuid.UUID) (db.GetResourceTestRow, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(db.GetResourceTestRow), args.Error(1)
}

func (m *MockQuerier) GetUserReservationHistory(ctx context.Context, userID uuid.UUID) ([]db.GetUserReservationHistoryRow, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]db.GetUserReservationHistoryRow), args.Error(1)
}

func (m *MockQuerier) PromoteNextInQueue(ctx context.Context, arg db.PromoteNextInQueueParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) UpdateReservationStatus(ctx context.Context, arg db.UpdateReservationStatusParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) GetQueueForResource(ctx context.Context, resourceID uuid.UUID) ([]db.GetQueueForResourceRow, error) {
	args := m.Called(ctx, resourceID)
	return args.Get(0).([]db.GetQueueForResourceRow), args.Error(1)
}

func (m *MockQuerier) ExpireReservation(ctx context.Context, arg db.ExpireReservationParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) FindExpiredActiveReservations(ctx context.Context) ([]db.ClaimctlReservation, error) {
	args := m.Called(ctx)
	return args.Get(0).([]db.ClaimctlReservation), args.Error(1)
}

// Secrets
func (m *MockQuerier) CreateSecret(ctx context.Context, arg db.CreateSecretParams) (db.ClaimctlSecret, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.ClaimctlSecret), args.Error(1)
}

func (m *MockQuerier) GetSecret(ctx context.Context, id uuid.UUID) (db.ClaimctlSecret, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(db.ClaimctlSecret), args.Error(1)
}

func (m *MockQuerier) GetSecretByKey(ctx context.Context, key string) (db.ClaimctlSecret, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(db.ClaimctlSecret), args.Error(1)
}

func (m *MockQuerier) ListSecrets(ctx context.Context) ([]db.ClaimctlSecret, error) {
	args := m.Called(ctx)
	return args.Get(0).([]db.ClaimctlSecret), args.Error(1)
}

func (m *MockQuerier) UpdateSecret(ctx context.Context, arg db.UpdateSecretParams) (db.ClaimctlSecret, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.ClaimctlSecret), args.Error(1)
}

func (m *MockQuerier) DeleteSecret(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// Webhooks
func (m *MockQuerier) CreateWebhook(ctx context.Context, arg db.CreateWebhookParams) (db.ClaimctlWebhook, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.ClaimctlWebhook), args.Error(1)
}

func (m *MockQuerier) GetWebhook(ctx context.Context, id uuid.UUID) (db.ClaimctlWebhook, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(db.ClaimctlWebhook), args.Error(1)
}

func (m *MockQuerier) ListWebhooks(ctx context.Context) ([]db.ClaimctlWebhook, error) {
	args := m.Called(ctx)
	return args.Get(0).([]db.ClaimctlWebhook), args.Error(1)
}

func (m *MockQuerier) UpdateWebhook(ctx context.Context, arg db.UpdateWebhookParams) (db.ClaimctlWebhook, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.ClaimctlWebhook), args.Error(1)
}

func (m *MockQuerier) DeleteWebhook(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// Resource Webhooks
func (m *MockQuerier) AddResourceWebhook(ctx context.Context, arg db.AddResourceWebhookParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) CancelAllReservationsForResource(ctx context.Context, arg db.CancelAllReservationsForResourceParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) RemoveResourceWebhook(ctx context.Context, arg db.RemoveResourceWebhookParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) GetResourceWebhooks(ctx context.Context, resourceID uuid.UUID) ([]db.GetResourceWebhooksRow, error) {
	args := m.Called(ctx, resourceID)
	return args.Get(0).([]db.GetResourceWebhooksRow), args.Error(1)
}

func (m *MockQuerier) GetWebhooksForEvent(ctx context.Context, arg db.GetWebhooksForEventParams) ([]db.ClaimctlWebhook, error) {
	args := m.Called(ctx, arg)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]db.ClaimctlWebhook), args.Error(1)
}

func (m *MockQuerier) CreateWebhookLog(ctx context.Context, arg db.CreateWebhookLogParams) (db.ClaimctlWebhookLog, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.ClaimctlWebhookLog), args.Error(1)
}

func (m *MockQuerier) GetWebhookLogs(ctx context.Context, arg db.GetWebhookLogsParams) ([]db.ClaimctlWebhookLog, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]db.ClaimctlWebhookLog), args.Error(1)
}

// Settings
func (m *MockQuerier) GetSetting(ctx context.Context, key string) (db.AppSetting, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(db.AppSetting), args.Error(1)
}

func (m *MockQuerier) GetSettings(ctx context.Context) ([]db.AppSetting, error) {
	args := m.Called(ctx)
	return args.Get(0).([]db.AppSetting), args.Error(1)
}

func (m *MockQuerier) UpsertSetting(ctx context.Context, arg db.UpsertSettingParams) (db.AppSetting, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.AppSetting), args.Error(1)
}

// User Notification Preferences
func (m *MockQuerier) GetUserPreferences(ctx context.Context, userID uuid.UUID) ([]db.ClaimctlUserNotificationPreference, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]db.ClaimctlUserNotificationPreference), args.Error(1)
}

func (m *MockQuerier) UpsertPreference(ctx context.Context, arg db.UpsertPreferenceParams) (db.ClaimctlUserNotificationPreference, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.ClaimctlUserNotificationPreference), args.Error(1)
}

func (m *MockQuerier) GetUserPreference(ctx context.Context, arg db.GetUserPreferenceParams) (db.ClaimctlUserNotificationPreference, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.ClaimctlUserNotificationPreference), args.Error(1)
}

func (m *MockQuerier) UpdateUserChannelConfig(ctx context.Context, arg db.UpdateUserChannelConfigParams) (db.ClaimctlUser, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.ClaimctlUser), args.Error(1)
}

// Group & Permission Management

func (m *MockQuerier) CreateGroup(ctx context.Context, arg db.CreateGroupParams) (db.ClaimctlGroup, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.ClaimctlGroup), args.Error(1)
}

func (m *MockQuerier) GetGroup(ctx context.Context, id uuid.UUID) (db.ClaimctlGroup, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(db.ClaimctlGroup), args.Error(1)
}

func (m *MockQuerier) ListGroups(ctx context.Context) ([]db.ClaimctlGroup, error) {
	args := m.Called(ctx)
	return args.Get(0).([]db.ClaimctlGroup), args.Error(1)
}

func (m *MockQuerier) UpdateGroup(ctx context.Context, arg db.UpdateGroupParams) (db.ClaimctlGroup, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.ClaimctlGroup), args.Error(1)
}

func (m *MockQuerier) DeleteGroup(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockQuerier) AddUserToGroup(ctx context.Context, arg db.AddUserToGroupParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) RemoveUserFromGroup(ctx context.Context, arg db.RemoveUserFromGroupParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) ListGroupMembers(ctx context.Context, groupID uuid.UUID) ([]db.ListGroupMembersRow, error) {
	args := m.Called(ctx, groupID)
	return args.Get(0).([]db.ListGroupMembersRow), args.Error(1)
}

func (m *MockQuerier) GetUserGroups(ctx context.Context, userID uuid.UUID) ([]db.ClaimctlGroup, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]db.ClaimctlGroup), args.Error(1)
}

func (m *MockQuerier) AddSpacePermission(ctx context.Context, arg db.AddSpacePermissionParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) RemoveSpacePermission(ctx context.Context, arg db.RemoveSpacePermissionParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) GetSpacePermissions(ctx context.Context, spaceID pgtype.UUID) ([]db.GetSpacePermissionsRow, error) {
	args := m.Called(ctx, spaceID)
	return args.Get(0).([]db.GetSpacePermissionsRow), args.Error(1)
}

func (m *MockQuerier) ListSpacesForUser(ctx context.Context, userID pgtype.UUID) ([]db.ClaimctlSpace, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]db.ClaimctlSpace), args.Error(1)
}

func (m *MockQuerier) GetAllResourcesWithReservationStatusForUser(ctx context.Context, userID pgtype.UUID) ([]db.GetAllResourcesWithReservationStatusForUserRow, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]db.GetAllResourcesWithReservationStatusForUserRow), args.Error(1)
}

func (m *MockQuerier) GetGroupByName(ctx context.Context, name string) (db.ClaimctlGroup, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(db.ClaimctlGroup), args.Error(1)
}

// Maintenance methods
func (m *MockQuerier) GetMaintenanceHistory(ctx context.Context, resourceID uuid.UUID) ([]db.GetMaintenanceHistoryRow, error) {
	args := m.Called(ctx, resourceID)
	return args.Get(0).([]db.GetMaintenanceHistoryRow), args.Error(1)
}

func (m *MockQuerier) GetResourceMaintenanceStatus(ctx context.Context, id uuid.UUID) (pgtype.Bool, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(pgtype.Bool), args.Error(1)
}

func (m *MockQuerier) LogMaintenanceChange(ctx context.Context, arg db.LogMaintenanceChangeParams) (db.ClaimctlMaintenanceAuditLog, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.ClaimctlMaintenanceAuditLog), args.Error(1)
}

func (m *MockQuerier) SetResourceMaintenanceMode(ctx context.Context, arg db.SetResourceMaintenanceModeParams) (db.ClaimctlResource, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.ClaimctlResource), args.Error(1)
}

func (m *MockQuerier) GetResourceName(ctx context.Context, id uuid.UUID) (string, error) {
	args := m.Called(ctx, id)
	return args.String(0), args.Error(1)
}

func (m *MockQuerier) UpdateUserPassword(ctx context.Context, arg db.UpdateUserPasswordParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) AcquireResourceLock(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockQuerier) CreateAuditLog(ctx context.Context, arg db.CreateAuditLogParams) (db.ClaimctlAuditLog, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.ClaimctlAuditLog), args.Error(1)
}

func (m *MockQuerier) GetAuditLogs(ctx context.Context, arg db.GetAuditLogsParams) ([]db.GetAuditLogsRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]db.GetAuditLogsRow), args.Error(1)
}
