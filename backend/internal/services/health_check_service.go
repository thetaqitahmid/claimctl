package services

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/utils"
)

type HealthCheckType string

const (
	HealthCheckTypePing HealthCheckType = "ping"
	HealthCheckTypeHTTP HealthCheckType = "http"
	HealthCheckTypeTCP  HealthCheckType = "tcp"
)

type HealthStatus string

const (
	HealthStatusHealthy  HealthStatus = "healthy"
	HealthStatusDegraded HealthStatus = "degraded"
	HealthStatusDown     HealthStatus = "down"
	HealthStatusUnknown  HealthStatus = "unknown"
)

type HealthCheckService interface {
	StartMonitoring(ctx context.Context) error
	StopMonitoring() error
	ExecuteCheck(ctx context.Context, resourceID uuid.UUID) error
	GetHealthConfig(ctx context.Context, resourceID uuid.UUID) (*db.ClaimctlResourceHealthConfig, error)
	UpsertHealthConfig(ctx context.Context, config db.UpsertHealthConfigParams) (*db.ClaimctlResourceHealthConfig, error)
	DeleteHealthConfig(ctx context.Context, resourceID uuid.UUID) error
	GetHealthStatus(ctx context.Context, resourceID uuid.UUID) (*db.ClaimctlResourceHealthStatus, error)
	GetHealthHistory(ctx context.Context, resourceID uuid.UUID, limit int32) ([]db.ClaimctlResourceHealthStatus, error)
}

type healthCheckService struct {
	db          db.Store
	jobQueue    chan uuid.UUID
	stopChan    chan struct{}
	workerCount int
	queueSize   int
	wg          sync.WaitGroup
	mu          sync.Mutex
	running     bool
}

func NewHealthCheckService(database db.Store) HealthCheckService {
	workerCount := min(64, max(1, utils.GetEnvAsInt("HEALTH_CHECK_WORKERS", 10)))
	queueSize := min(1000, max(1, utils.GetEnvAsInt("HEALTH_CHECK_QUEUE_SIZE", 100)))

	return &healthCheckService{
		db:          database,
		jobQueue:    make(chan uuid.UUID, queueSize),
		stopChan:    make(chan struct{}),
		workerCount: workerCount,
		queueSize:   queueSize,
		running:     false,
	}
}

func (s *healthCheckService) StartMonitoring(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("health check monitoring already running")
	}
	s.running = true
	s.stopChan = make(chan struct{})
	s.mu.Unlock()

	slog.Info("Starting health check monitoring",
		"workers", s.workerCount,
		"queueSize", s.queueSize)

	// Start worker pool
	for i := 0; i < s.workerCount; i++ {
		s.wg.Add(1)
		go s.worker(ctx, i)
	}

	// Start scheduler
	s.wg.Add(1)
	go s.scheduler(ctx)

	return nil
}

func (s *healthCheckService) StopMonitoring() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return fmt.Errorf("health check monitoring not running")
	}
	slog.Info("Stopping health check monitoring")
	close(s.stopChan)
	s.wg.Wait()
	s.running = false
	s.mu.Unlock()
	slog.Info("Health check monitoring stopped")

	return nil
}

func (s *healthCheckService) worker(ctx context.Context, workerID int) {
	defer s.wg.Done()

	slog.Debug("Health check worker started", "workerID", workerID)

	for {
		select {
		case resourceID := <-s.jobQueue:
			slog.Debug("Worker processing health check",
				"workerID", workerID,
				"resourceID", resourceID)

			if err := s.ExecuteCheck(ctx, resourceID); err != nil {
				slog.Error("Health check failed",
					"workerID", workerID,
					"resourceID", resourceID,
					"error", err)
			}

		case <-s.stopChan:
			slog.Debug("Health check worker stopping", "workerID", workerID)
			return
		case <-ctx.Done():
			slog.Debug("Health check worker stopping via signal", "workerID", workerID)
			return
		}
	}
}

func (s *healthCheckService) scheduler(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	slog.Debug("Health check scheduler started")

	for {
		select {
		case <-ticker.C:
			if err := s.scheduleChecks(ctx); err != nil {
				slog.Error("Failed to schedule health checks", "error", err)
			}

		case <-s.stopChan:
			slog.Debug("Health check scheduler stopping")
			return
		case <-ctx.Done():
			slog.Debug("Health check scheduler stopping via signal")
			return
		}
	}
}

func (s *healthCheckService) scheduleChecks(ctx context.Context) error {
	// Use context with timeout for DB query
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resources, err := s.db.GetResourcesDueForCheck(dbCtx)
	if err != nil {
		return fmt.Errorf("failed to get resources due for check: %w", err)
	}

	slog.Debug("Scheduling health checks", "count", len(resources))

	for _, resource := range resources {
		select {
		case s.jobQueue <- resource.ResourceID:
			// Successfully queued
		default:
			// Queue is full, skip this resource for now
			slog.Warn("Health check queue full, skipping resource",
				"resourceID", resource.ResourceID)
		}
	}

	return nil
}

func (s *healthCheckService) ExecuteCheck(ctx context.Context, resourceID uuid.UUID) error {
	if err := AuthorizeRequestResource(ctx, s.db, resourceID, true); err != nil {
		return err
	}
	checkCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	ctx = checkCtx
	name := "health:" + resourceID.String()
	fence := uuid.New()
	leaseCtx, leaseCancel := context.WithTimeout(ctx, 5*time.Second)
	_, err := s.db.AcquireWorkerLease(leaseCtx, db.AcquireWorkerLeaseParams{Name: name, Fence: fence})
	leaseCancel()
	if err == pgx.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	// Get health check configuration
	configCtx, configCancel := context.WithTimeout(ctx, 5*time.Second)
	config, err := s.db.GetHealthConfig(configCtx, resourceID)
	configCancel()

	if err != nil {
		return fmt.Errorf("failed to get health config for resource %d: %w", resourceID, err)
	}

	if !config.Enabled.Bool {
		return nil
	}

	// Execute the appropriate check
	var status HealthStatus
	var responseTimeMs int32
	var errorMessage string

	startTime := time.Now()

	err = executeHealthProbe(ctx, HealthCheckType(config.CheckType), config.Target, int(config.TimeoutSeconds.Int32), int(config.RetryCount.Int32))

	responseTimeMs = int32(time.Since(startTime).Milliseconds())

	if err != nil {
		status = HealthStatusDown
		errorMessage = err.Error()
	} else {
		status = HealthStatusHealthy
	}

	// Record the result
	return s.recordHealthStatus(ctx, resourceID, status, responseTimeMs, errorMessage, name, fence, max(1, config.IntervalSeconds.Int32))
}

func (s *healthCheckService) recordHealthStatus(ctx context.Context, resourceID uuid.UUID, status HealthStatus, responseTimeMs int32, errorMessage string, name string, fence uuid.UUID, interval int32) error {
	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var errMsg pgtype.Text
	if errorMessage != "" {
		errMsg = pgtype.Text{String: errorMessage, Valid: true}
	}

	var respTime pgtype.Int4
	if responseTimeMs > 0 {
		respTime = pgtype.Int4{Int32: responseTimeMs, Valid: true}
	}

	err := s.db.ExecTx(dbCtx, func(q db.Querier) error {
		count, err := q.FinishWorkerLease(dbCtx, db.FinishWorkerLeaseParams{Name: name, Fence: fence, DelaySeconds: interval})
		if err != nil || count == 0 {
			return err
		}
		_, err = q.CreateHealthStatus(dbCtx, db.CreateHealthStatusParams{
			ResourceID: resourceID, Status: string(status), ResponseTimeMs: respTime,
			ErrorMessage: errMsg, CheckedAt: time.Now().Unix(),
		})
		return err
	})

	if err != nil {
		return fmt.Errorf("failed to record health status: %w", err)
	}

	slog.Debug("Recorded health status",
		"resourceID", resourceID,
		"status", status,
		"responseTimeMs", responseTimeMs)

	return nil
}

func (s *healthCheckService) GetHealthConfig(ctx context.Context, resourceID uuid.UUID) (*db.ClaimctlResourceHealthConfig, error) {
	if err := AuthorizeRequestResource(ctx, s.db, resourceID, false); err != nil {
		return nil, err
	}
	config, err := s.db.GetHealthConfig(ctx, resourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get health config: %w", err)
	}
	return &config, nil
}

func (s *healthCheckService) UpsertHealthConfig(ctx context.Context, params db.UpsertHealthConfigParams) (*db.ClaimctlResourceHealthConfig, error) {
	if err := AuthorizeRequestResource(ctx, s.db, params.ResourceID, true); err != nil {
		return nil, err
	}
	if params.TimeoutSeconds.Int32 < 1 || params.TimeoutSeconds.Int32 > 30 || params.RetryCount.Int32 < 0 || params.RetryCount.Int32 > 3 || params.IntervalSeconds.Int32 < 10 || params.IntervalSeconds.Int32 > 86400 {
		return nil, fmt.Errorf("health interval/timeout/retries out of bounds")
	}
	policy, err := newHealthEgressPolicy()
	if err != nil {
		return nil, err
	}
	host, port, err := healthTarget(HealthCheckType(params.CheckType), params.Target)
	if err != nil {
		return nil, err
	}
	resolveCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if _, err := policy.addresses(resolveCtx, host, port); err != nil {
		return nil, err
	}
	config, err := s.db.UpsertHealthConfig(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert health config: %w", err)
	}
	return &config, nil
}

func (s *healthCheckService) DeleteHealthConfig(ctx context.Context, resourceID uuid.UUID) error {
	if err := AuthorizeRequestResource(ctx, s.db, resourceID, true); err != nil {
		return err
	}
	err := s.db.DeleteHealthConfig(ctx, resourceID)
	if err != nil {
		return fmt.Errorf("failed to delete health config: %w", err)
	}
	return nil
}

func (s *healthCheckService) GetHealthStatus(ctx context.Context, resourceID uuid.UUID) (*db.ClaimctlResourceHealthStatus, error) {
	if err := AuthorizeRequestResource(ctx, s.db, resourceID, false); err != nil {
		return nil, err
	}
	status, err := s.db.GetHealthStatus(ctx, resourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get health status: %w", err)
	}
	return &status, nil
}

func (s *healthCheckService) GetHealthHistory(ctx context.Context, resourceID uuid.UUID, limit int32) ([]db.ClaimctlResourceHealthStatus, error) {
	if err := AuthorizeRequestResource(ctx, s.db, resourceID, false); err != nil {
		return nil, err
	}
	history, err := s.db.GetHealthHistory(ctx, db.GetHealthHistoryParams{
		ResourceID: resourceID,
		Limit:      limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get health history: %w", err)
	}
	return history, nil
}
