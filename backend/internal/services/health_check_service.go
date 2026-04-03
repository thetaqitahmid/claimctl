package services

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5/pgtype"
	probing "github.com/prometheus-community/pro-bing"
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
	db          db.Querier
	jobQueue    chan uuid.UUID
	stopChan    chan struct{}
	workerCount int
	queueSize   int
	wg          sync.WaitGroup
	mu          sync.Mutex
	running     bool
}

func NewHealthCheckService(database db.Querier) HealthCheckService {
	workerCount := utils.GetEnvAsInt("HEALTH_CHECK_WORKERS", 10)
	queueSize := utils.GetEnvAsInt("HEALTH_CHECK_QUEUE_SIZE", 100)

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
	s.running = false
	s.mu.Unlock()

	slog.Info("Stopping health check monitoring")
	close(s.stopChan)
	s.wg.Wait()
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

	switch HealthCheckType(config.CheckType) {
	case HealthCheckTypePing:
		err = s.executePingCheck(config.Target, int(config.TimeoutSeconds.Int32), int(config.RetryCount.Int32))
	case HealthCheckTypeHTTP:
		err = s.executeHTTPCheck(config.Target, int(config.TimeoutSeconds.Int32), int(config.RetryCount.Int32))
	case HealthCheckTypeTCP:
		err = s.executeTCPCheck(config.Target, int(config.TimeoutSeconds.Int32), int(config.RetryCount.Int32))
	default:
		return fmt.Errorf("unknown check type: %s", config.CheckType)
	}

	responseTimeMs = int32(time.Since(startTime).Milliseconds())

	if err != nil {
		status = HealthStatusDown
		errorMessage = err.Error()
	} else {
		status = HealthStatusHealthy
	}

	// Record the result
	return s.recordHealthStatus(ctx, resourceID, status, responseTimeMs, errorMessage)
}

func (s *healthCheckService) recordHealthStatus(ctx context.Context, resourceID uuid.UUID, status HealthStatus, responseTimeMs int32, errorMessage string) error {
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

	_, err := s.db.CreateHealthStatus(dbCtx, db.CreateHealthStatusParams{
		ResourceID:     resourceID,
		Status:         string(status),
		ResponseTimeMs: respTime,
		ErrorMessage:   errMsg,
		CheckedAt:      time.Now().Unix(),
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

func (s *healthCheckService) executePingCheck(target string, timeoutSeconds int, retryCount int) error {
	pinger, err := probing.NewPinger(target)
	if err != nil {
		return fmt.Errorf("failed to create pinger: %w", err)
	}

	pinger.Count = retryCount + 1
	pinger.Timeout = time.Duration(timeoutSeconds) * time.Second
	pinger.SetPrivileged(false)

	err = pinger.Run()
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	stats := pinger.Statistics()
	if stats.PacketsRecv == 0 {
		return fmt.Errorf("no packets received")
	}

	return nil
}

func (s *healthCheckService) executeHTTPCheck(target string, timeoutSeconds int, retryCount int) error {
	client := &http.Client{
		Timeout: time.Duration(timeoutSeconds) * time.Second,
	}

	var lastErr error
	for attempt := 0; attempt <= retryCount; attempt++ {
		resp, err := client.Get(target)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			return nil
		}

		lastErr = fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	if lastErr != nil {
		return fmt.Errorf("HTTP check failed after %d retries: %w", retryCount, lastErr)
	}

	return fmt.Errorf("HTTP check failed after %d retries", retryCount)
}

func (s *healthCheckService) executeTCPCheck(target string, timeoutSeconds int, retryCount int) error {
	var lastErr error
	timeout := time.Duration(timeoutSeconds) * time.Second

	for attempt := 0; attempt <= retryCount; attempt++ {
		conn, err := net.DialTimeout("tcp", target, timeout)
		if err != nil {
			lastErr = err
			continue
		}
		conn.Close()
		return nil
	}

	if lastErr != nil {
		return fmt.Errorf("TCP check failed after %d retries: %w", retryCount, lastErr)
	}

	return fmt.Errorf("TCP check failed after %d retries", retryCount)
}

func (s *healthCheckService) GetHealthConfig(ctx context.Context, resourceID uuid.UUID) (*db.ClaimctlResourceHealthConfig, error) {
	config, err := s.db.GetHealthConfig(ctx, resourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get health config: %w", err)
	}
	return &config, nil
}

func (s *healthCheckService) UpsertHealthConfig(ctx context.Context, params db.UpsertHealthConfigParams) (*db.ClaimctlResourceHealthConfig, error) {
	config, err := s.db.UpsertHealthConfig(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert health config: %w", err)
	}
	return &config, nil
}

func (s *healthCheckService) DeleteHealthConfig(ctx context.Context, resourceID uuid.UUID) error {
	err := s.db.DeleteHealthConfig(ctx, resourceID)
	if err != nil {
		return fmt.Errorf("failed to delete health config: %w", err)
	}
	return nil
}

func (s *healthCheckService) GetHealthStatus(ctx context.Context, resourceID uuid.UUID) (*db.ClaimctlResourceHealthStatus, error) {
	status, err := s.db.GetHealthStatus(ctx, resourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get health status: %w", err)
	}
	return &status, nil
}

func (s *healthCheckService) GetHealthHistory(ctx context.Context, resourceID uuid.UUID, limit int32) ([]db.ClaimctlResourceHealthStatus, error) {
	history, err := s.db.GetHealthHistory(ctx, db.GetHealthHistoryParams{
		ResourceID: resourceID,
		Limit:      limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get health history: %w", err)
	}
	return history, nil
}
