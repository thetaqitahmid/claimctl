package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/utils"
)

type ResourceWithStatus struct {
	Resource                   db.ClaimctlResource              `json:"resource"`
	ActiveReservations         int64                                 `json:"activeReservations"`
	QueueLength                int64                                 `json:"queueLength"`
	NextUserID uuid.UUID                                 `json:"nextUserId"`
	NextQueuePosition          int32                                 `json:"nextQueuePosition"`
	HealthStatus               *db.ClaimctlResourceHealthStatus `json:"healthStatus,omitempty"`
	HealthConfig               *db.ClaimctlResourceHealthConfig `json:"healthConfig,omitempty"`
	ActiveReservationStartTime *int64                                `json:"activeReservationStartTime,omitempty"`
	ActiveReservationDuration  *string                               `json:"activeReservationDuration,omitempty"`
	ActiveReservationCreatedAt *int64                                `json:"activeReservationCreatedAt,omitempty"`
}

type ResourceService interface {
	CreateResource(ctx context.Context, req CreateResourceRequest) (*db.ClaimctlResource, error)
	GetResource(ctx context.Context, id uuid.UUID, userID uuid.UUID, isAdmin bool) (*db.ClaimctlResource, error)
	GetResourceWithStatus(ctx context.Context, id uuid.UUID, userID uuid.UUID, isAdmin bool) (*ResourceWithStatus, error)
	GetAllResources(ctx context.Context, labelFilter utils.FilterExpr) (*[]db.ClaimctlResource, error)
	GetAllResourcesWithStatus(ctx context.Context, labelFilter utils.FilterExpr) (*[]ResourceWithStatus, error)
	GetAllResourcesWithStatusForUser(ctx context.Context, userID uuid.UUID, isAdmin bool, labelFilter utils.FilterExpr) (*[]ResourceWithStatus, error)
	UpdateResource(ctx context.Context, req UpdateResourceRequest) (*db.ClaimctlResource, error)
	DeleteResource(ctx context.Context, id uuid.UUID) error
	SetMaintenanceMode(ctx context.Context, resourceID uuid.UUID, enabled bool, userID uuid.UUID, reason string) (*db.ClaimctlResource, error)
	GetMaintenanceHistory(ctx context.Context, resourceID uuid.UUID) (*[]db.GetMaintenanceHistoryRow, error)
}

type CreateResourceRequest = db.CreateNewResourceParams
type UpdateResourceRequest = db.UpdateResourceByIdParams

type resourceService struct {
	db          db.Querier
	store       db.Store
	realtimeSvc RealtimeService
}

func NewResourceService(store db.Store, realtimeSvc RealtimeService) ResourceService {
	return &resourceService{
		db:          store,
		store:       store,
		realtimeSvc: realtimeSvc,
	}
}

func (s *resourceService) CreateResource(ctx context.Context, req CreateResourceRequest) (*db.ClaimctlResource, error) {
	if req.Name == "" || req.Type == "" || len(req.Labels) == 0 {
		return nil, fmt.Errorf("invalid input values for creating resource")
	}

	if len(req.Properties) > 10 {
		return nil, fmt.Errorf("too many properties: maximum 10 allowed")
	}

	count, err := s.db.VerifyResourceNameIsUnique(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to validate resource name")
	}
	if count != 0 {
		return nil, fmt.Errorf("resource name '%s' already exists", req.Name)
	}

	var spaceID uuid.UUID
	if req.SpaceID != uuid.Nil {
		spaceID = req.SpaceID
		_, err := s.db.GetSpace(ctx, spaceID)
		if err != nil {
			return nil, fmt.Errorf("invalid space id %s: %w", spaceID, err)
		}
	} else {
		defaultSpace, err := s.db.GetSpaceByName(ctx, "Default Space")
		if err != nil {
			return nil, fmt.Errorf("failed to find Default Space: %w", err)
		}
		spaceID = defaultSpace.ID
	}
	req.SpaceID = spaceID

	resource, err := s.db.CreateNewResource(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create new resource: %w", err)
	}

	return &resource, nil
}

func (s *resourceService) GetResource(ctx context.Context, id uuid.UUID, userID uuid.UUID, isAdmin bool) (*db.ClaimctlResource, error) {
	if !isAdmin {
		// Check if user has access to the space this resource belongs to
		resource, err := s.db.FindResourceById(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("resource not found")
		}
		hasPerm, err := s.db.HasSpacePermission(ctx, db.HasSpacePermissionParams{
			ID:     resource.SpaceID,
			UserID: pgtype.UUID{Bytes: userID, Valid: true},
		})
		if err != nil || !hasPerm {
			return nil, fmt.Errorf("access denied")
		}
		return &resource, nil
	}

	resource, err := s.db.FindResourceById(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("resource with ID %s not found", id)
	}
	return &resource, nil
}

func (s *resourceService) GetAllResources(ctx context.Context, labelFilter utils.FilterExpr) (*[]db.ClaimctlResource, error) {
	resources, err := s.db.FindAllResources(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve resources: %w", err)
	}

	if labelFilter == nil {
		return &resources, nil
	}

	var filtered []db.ClaimctlResource
	for _, r := range resources {
		labelsStr := make([]string, len(r.Labels))
		for i, l := range r.Labels {
			labelsStr[i] = fmt.Sprint(l)
		}
		if labelFilter.Matches(labelsStr) {
			filtered = append(filtered, r)
		}
	}

	return &filtered, nil
}

func (s *resourceService) GetResourceWithStatus(ctx context.Context, id uuid.UUID, userID uuid.UUID, isAdmin bool) (*ResourceWithStatus, error) {
	if !isAdmin {
		// Check space permission
		resource, err := s.db.FindResourceById(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("resource not found")
		}
		hasPerm, err := s.db.HasSpacePermission(ctx, db.HasSpacePermissionParams{
			ID:     resource.SpaceID,
			UserID: pgtype.UUID{Bytes: userID, Valid: true},
		})
		if err != nil || !hasPerm {
			return nil, fmt.Errorf("access denied")
		}
	}

	resourceStatus, err := s.db.GetResourceReservationStatus(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("resource with ID %s not found: %w", id, err)
	}

	resource := db.ClaimctlResource{
		ID:                 resourceStatus.ID,
		Name:               resourceStatus.Name,
		Type:               resourceStatus.Type,
		Labels:             resourceStatus.Labels,
		CreatedAt:          resourceStatus.CreatedAt,
		UpdatedAt:          resourceStatus.UpdatedAt,
		SpaceID:            resourceStatus.SpaceID,
		Properties:         resourceStatus.Properties,
		IsUnderMaintenance: resourceStatus.IsUnderMaintenance,
	}

	nextUserID := resourceStatus.NextUserID
	nextQueuePosition := resourceStatus.NextQueuePosition

	// Fetch health config and status
	var healthConfig *db.ClaimctlResourceHealthConfig
	var healthStatus *db.ClaimctlResourceHealthStatus

	config, err := s.db.GetHealthConfig(ctx, id)
	if err == nil {
		healthConfig = &config
	}

	status, err := s.db.GetHealthStatus(ctx, id)
	if err == nil {
		healthStatus = &status
	}

	var activeReservationStartTime *int64
	if resourceStatus.ActiveReservationStartTime.Valid {
		val := resourceStatus.ActiveReservationStartTime.Int64
		activeReservationStartTime = &val
	}

	var activeReservationCreatedAt *int64
	if resourceStatus.ActiveReservationCreatedAt.Valid {
		val := resourceStatus.ActiveReservationCreatedAt.Int64
		activeReservationCreatedAt = &val
	}

	var activeReservationDuration *string
	if resourceStatus.ActiveReservationDuration.Valid {
		micro := resourceStatus.ActiveReservationDuration.Microseconds
		days := resourceStatus.ActiveReservationDuration.Days
		months := resourceStatus.ActiveReservationDuration.Months
		totalMicros := micro + int64(days)*24*3600*1000000 + int64(months)*30*24*3600*1000000
		d := time.Duration(totalMicros * 1000)
		s := d.String()
		activeReservationDuration = &s
	}

	return &ResourceWithStatus{
		Resource:                   resource,
		ActiveReservations:         resourceStatus.ActiveReservations,
		QueueLength:                resourceStatus.QueueLength,
		NextUserID:                 nextUserID,
		NextQueuePosition:          nextQueuePosition,
		HealthConfig:               healthConfig,
		HealthStatus:               healthStatus,
		ActiveReservationStartTime: activeReservationStartTime,
		ActiveReservationDuration:  activeReservationDuration,
		ActiveReservationCreatedAt: activeReservationCreatedAt,
	}, nil
}

func (s *resourceService) GetAllResourcesWithStatus(ctx context.Context, labelFilter utils.FilterExpr) (*[]ResourceWithStatus, error) {
	resourcesStatus, err := s.db.GetAllResourcesWithReservationStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve resources with status: %w", err)
	}
	return s.enrichResourcesWithHealth(ctx, resourcesStatus, labelFilter)
}

func (s *resourceService) GetAllResourcesWithStatusForUser(ctx context.Context, userID uuid.UUID, isAdmin bool, labelFilter utils.FilterExpr) (*[]ResourceWithStatus, error) {
	if isAdmin {
		return s.GetAllResourcesWithStatus(ctx, labelFilter)
	}

	resourcesStatus, err := s.db.GetAllResourcesWithReservationStatusForUser(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve resources with status for user: %w", err)
	}

	// Filter by labels first to avoid unnecessary enrichment work
	var filteredStatus []db.GetAllResourcesWithReservationStatusForUserRow
	if labelFilter != nil {
		for _, r := range resourcesStatus {
			labelsStr := make([]string, len(r.Labels))
			for j, l := range r.Labels {
				labelsStr[j] = fmt.Sprint(l)
			}
			if labelFilter.Matches(labelsStr) {
				filteredStatus = append(filteredStatus, r)
			}
		}
	} else {
		filteredStatus = resourcesStatus
	}

	resources := make([]ResourceWithStatus, len(filteredStatus))
	var wg sync.WaitGroup
	errChan := make(chan error, len(filteredStatus))

	// Optimizing concurrent health fetches
	for i, r := range filteredStatus {
		wg.Add(1)
		go func(idx int, resourceStatus db.GetAllResourcesWithReservationStatusForUserRow) {
			defer wg.Done()

			// Build Base Resource Object
			resource := db.ClaimctlResource{
				ID:                 resourceStatus.ID,
				Name:               resourceStatus.Name,
				Type:               resourceStatus.Type,
				Labels:             resourceStatus.Labels,
				CreatedAt:          resourceStatus.CreatedAt,
				UpdatedAt:          resourceStatus.UpdatedAt,
				SpaceID:            resourceStatus.SpaceID,
				Properties:         resourceStatus.Properties,
				IsUnderMaintenance: resourceStatus.IsUnderMaintenance,
			}

			var healthConfig *db.ClaimctlResourceHealthConfig
			var healthStatus *db.ClaimctlResourceHealthStatus

			// In a real high-load scenario, we might want to batch these or cache them
			hCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()

			config, err := s.db.GetHealthConfig(hCtx, resourceStatus.ID)
			if err == nil {
				healthConfig = &config
			}

			status, err := s.db.GetHealthStatus(hCtx, resourceStatus.ID)
			if err == nil {
				healthStatus = &status
			}

			var activeReservationStartTime *int64
			if resourceStatus.ActiveReservationStartTime.Valid {
				val := resourceStatus.ActiveReservationStartTime.Int64
				activeReservationStartTime = &val
			}

			var activeReservationCreatedAt *int64
			if resourceStatus.ActiveReservationCreatedAt.Valid {
				val := resourceStatus.ActiveReservationCreatedAt.Int64
				activeReservationCreatedAt = &val
			}

			var activeReservationDuration *string
			if resourceStatus.ActiveReservationDuration.Valid {
				micro := resourceStatus.ActiveReservationDuration.Microseconds
				days := resourceStatus.ActiveReservationDuration.Days
				months := resourceStatus.ActiveReservationDuration.Months
				totalMicros := micro + int64(days)*24*3600*1000000 + int64(months)*30*24*3600*1000000
				d := time.Duration(totalMicros * 1000)
				s := d.String()
				activeReservationDuration = &s
			}

			resources[idx] = ResourceWithStatus{
				Resource:                   resource,
				ActiveReservations:         resourceStatus.ActiveReservations,
				QueueLength:                resourceStatus.QueueLength,
				NextUserID:                 resourceStatus.NextUserID,
				NextQueuePosition:          resourceStatus.NextQueuePosition,
				HealthConfig:               healthConfig,
				HealthStatus:               healthStatus,
				ActiveReservationStartTime: activeReservationStartTime,
				ActiveReservationDuration:  activeReservationDuration,
				ActiveReservationCreatedAt: activeReservationCreatedAt,
			}
		}(i, r)
	}

	wg.Wait()
	close(errChan)
	if len(errChan) > 0 {
		// Log errors but return what we have? Or return error?
		// For now, ignoring individual health fetch errors as they are optional fields
	}

	return &resources, nil
}

// Helper to avoid duplication between Admin and User flows for the existing GetAllResourcesWithStatus
func (s *resourceService) enrichResourcesWithHealth(ctx context.Context, resourcesStatus []db.GetAllResourcesWithReservationStatusRow, labelFilter utils.FilterExpr) (*[]ResourceWithStatus, error) {
	// Filter by labels first
	var filteredStatus []db.GetAllResourcesWithReservationStatusRow
	if labelFilter != nil {
		for _, r := range resourcesStatus {
			labelsStr := make([]string, len(r.Labels))
			for j, l := range r.Labels {
				labelsStr[j] = fmt.Sprint(l)
			}
			if labelFilter.Matches(labelsStr) {
				filteredStatus = append(filteredStatus, r)
			}
		}
	} else {
		filteredStatus = resourcesStatus
	}

	resources := make([]ResourceWithStatus, len(filteredStatus))

	// Avoiding N+1 problem naively for now, but parallelizing helper
	// Ideally we'd have bulk fetch queries for health

	for i, resourceStatus := range filteredStatus {
		resource := db.ClaimctlResource{
			ID:                 resourceStatus.ID,
			Name:               resourceStatus.Name,
			Type:               resourceStatus.Type,
			Labels:             resourceStatus.Labels,
			CreatedAt:          resourceStatus.CreatedAt,
			UpdatedAt:          resourceStatus.UpdatedAt,
			SpaceID:            resourceStatus.SpaceID,
			Properties:         resourceStatus.Properties,
			IsUnderMaintenance: resourceStatus.IsUnderMaintenance,
		}

		var healthConfig *db.ClaimctlResourceHealthConfig
		var healthStatus *db.ClaimctlResourceHealthStatus

		config, err := s.db.GetHealthConfig(ctx, resourceStatus.ID)
		if err == nil {
			healthConfig = &config
		}

		status, err := s.db.GetHealthStatus(ctx, resourceStatus.ID)
		if err == nil {
			healthStatus = &status
		}

		var activeReservationStartTime *int64
		if resourceStatus.ActiveReservationStartTime.Valid {
			val := resourceStatus.ActiveReservationStartTime.Int64
			activeReservationStartTime = &val
		}

		var activeReservationCreatedAt *int64
		if resourceStatus.ActiveReservationCreatedAt.Valid {
			val := resourceStatus.ActiveReservationCreatedAt.Int64
			activeReservationCreatedAt = &val
		}

		var activeReservationDuration *string
		if resourceStatus.ActiveReservationDuration.Valid {
			micro := resourceStatus.ActiveReservationDuration.Microseconds
			days := resourceStatus.ActiveReservationDuration.Days
			months := resourceStatus.ActiveReservationDuration.Months
			totalMicros := micro + int64(days)*24*3600*1000000 + int64(months)*30*24*3600*1000000
			d := time.Duration(totalMicros * 1000)
			s := d.String()
			activeReservationDuration = &s
		}

		resources[i] = ResourceWithStatus{
			Resource:                   resource,
			ActiveReservations:         resourceStatus.ActiveReservations,
			QueueLength:                resourceStatus.QueueLength,
			NextUserID:                 resourceStatus.NextUserID,
			NextQueuePosition:          resourceStatus.NextQueuePosition,
			HealthConfig:               healthConfig,
			HealthStatus:               healthStatus,
			ActiveReservationStartTime: activeReservationStartTime,
			ActiveReservationDuration:  activeReservationDuration,
			ActiveReservationCreatedAt: activeReservationCreatedAt,
		}
	}
	return &resources, nil
}

func (s *resourceService) UpdateResource(ctx context.Context, req UpdateResourceRequest) (*db.ClaimctlResource, error) {
	oldResource, err := s.db.FindResourceById(ctx, req.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve updated resource with ID %s: %w", req.ID, err)
	}

	if len(req.Name) == 0 {
		req.Name = oldResource.Name
	}
	if len(req.Type) == 0 {
		req.Type = oldResource.Type
	}
	if len(req.Labels) == 0 {
		req.Labels = oldResource.Labels
	}
	if req.Properties == nil {
		req.Properties = oldResource.Properties
	}

	if len(req.Properties) > 10 {
		return nil, fmt.Errorf("too many properties: maximum 10 allowed")
	}

	err = s.db.UpdateResourceById(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update resource with ID %s: %w", req.ID, err)
	}

	updatedResource, err := s.db.FindResourceById(ctx, req.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve updated resource with ID %s: %w", req.ID, err)
	}
	return &updatedResource, nil
}

func (s *resourceService) DeleteResource(ctx context.Context, id uuid.UUID) error {
	err := s.db.DeleteResourceById(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete resource with ID %s: %w", id, err)
	}
	return nil
}

func (s *resourceService) SetMaintenanceMode(ctx context.Context, resourceID uuid.UUID, enabled bool, userID uuid.UUID, reason string) (*db.ClaimctlResource, error) {
	// Get current resource to check current maintenance state
	currentResource, err := s.db.FindResourceById(ctx, resourceID)
	if err != nil {
		return nil, fmt.Errorf("resource with ID %s not found: %w", resourceID, err)
	}

	// Only update if the state is actually changing
	currentState := currentResource.IsUnderMaintenance.Valid && currentResource.IsUnderMaintenance.Bool
	if currentState == enabled {
		// No change needed, but still return the resource
		return &currentResource, nil
	}

	// Update maintenance mode in a transaction
	var updatedResource db.ClaimctlResource
	err = s.store.ExecTx(ctx, func(q db.Querier) error {
		var txErr error
		now := pgtype.Int8{Int64: time.Now().Unix(), Valid: true}
		maintenanceState := pgtype.Bool{Bool: enabled, Valid: true}

		updatedResource, txErr = q.SetResourceMaintenanceMode(ctx, db.SetResourceMaintenanceModeParams{
			ID:                 resourceID,
			IsUnderMaintenance: maintenanceState,
			UpdatedAt:          now,
		})
		if txErr != nil {
			return fmt.Errorf("failed to update maintenance mode for resource %d: %w", resourceID, txErr)
		}

		// Log the change
		reasonText := pgtype.Text{String: reason, Valid: reason != ""}
		_, txErr = q.LogMaintenanceChange(ctx, db.LogMaintenanceChangeParams{
			ResourceID:    resourceID,
			PreviousState: currentState,
			NewState:      enabled,
			ChangedBy:     userID,
			Reason:        reasonText,
		})
		if txErr != nil {
			return fmt.Errorf("failed to log maintenance change for resource %d: %w", resourceID, txErr)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// TODO: Broadcast maintenance change event via SSE
	// Requires moving realtimeService initialization before resourceService in routes.go

	return &updatedResource, nil
}

func (s *resourceService) GetMaintenanceHistory(ctx context.Context, resourceID uuid.UUID) (*[]db.GetMaintenanceHistoryRow, error) {
	history, err := s.db.GetMaintenanceHistory(ctx, resourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get maintenance history for resource %d: %w", resourceID, err)
	}
	return &history, nil
}
