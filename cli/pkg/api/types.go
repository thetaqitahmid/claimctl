package api

import (
	"time"
)

// Resource matches the inner `resource` object from the API.
type Resource struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Labels     []string               `json:"labels"`
	Properties map[string]interface{} `json:"properties"`
	SpaceID    string                 `json:"spaceId"`
	CreatedAt  int64                  `json:"createdAt"`
	UpdatedAt  int64                  `json:"updatedAt"`
	Status     string                 `json:"status,omitempty"` // Derived field
	Health     string                 `json:"health,omitempty"` // Derived field
}

// ResourceWithStatus matches the item in the array returned by GetResourcesWithStatus
type ResourceWithStatus struct {
	Resource           Resource      `json:"resource"`
	ActiveReservations int64         `json:"activeReservations"`
	QueueLength        int64         `json:"queueLength"`
	NextUserID         string        `json:"nextUserId"`
	NextQueuePosition  int32         `json:"nextQueuePosition"`
	HealthStatus       *HealthStatus `json:"healthStatus,omitempty"`
}

// Accessor to get a flat structure + derived status for the CLI commands
func (r *ResourceWithStatus) ToCLIResource() Resource {
	res := r.Resource
	// Simple derived status
	if r.ActiveReservations > 0 {
		res.Status = "Reserved"
	} else if r.QueueLength > 0 {
		res.Status = "Queue"
	} else {
		res.Status = "Available"
	}

	// Derived health
	if r.HealthStatus != nil {
		res.Health = r.HealthStatus.Status
	} else {
		res.Health = "N/A"
	}

	return res
}

// Reservation represents a booking.
type Reservation struct {
	ID               string    `json:"id"`
	ResourceID       string    `json:"resourceId"`
	UserID           string    `json:"userId"`
	Status           string    `json:"status"`
	QueuePosition    *int32    `json:"queuePosition,omitempty"`
	StartTime        int64     `json:"startTime"`
	EndTime          int64     `json:"endTime"`
	CreatedAt        int64     `json:"createdAt"`
	UpdatedAt        int64     `json:"updatedAt"`
	ScheduledEndTime time.Time `json:"scheduledEndTime"`
	Duration         any       `json:"duration,omitempty"`
}

// UserReservation matches the enriched reservation row from the API.
type UserReservation struct {
	ID            string `json:"id"`
	ResourceID    string `json:"resourceId"`
	UserID        string `json:"userId"`
	Status        string `json:"status"`
	QueuePosition *int32 `json:"queuePosition"`
	StartTime     int64  `json:"startTime"`
	EndTime       int64  `json:"endTime"`
	CreatedAt     int64  `json:"createdAt"`
	ResourceName  string `json:"resourceName"`
	ResourceType  string `json:"resourceType"`
	Duration      any    `json:"duration"`
}

// CreateReservationRequest is the payload for creating a reservation.
// Webhook represents a system webhook.
type Webhook struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Url           string            `json:"url"`
	Method        string            `json:"method"`
	Headers       map[string]string `json:"headers"`
	Template      string            `json:"template"`
	Description   string            `json:"description"`
	SigningSecret string            `json:"signingSecret"`
	CreatedAt     int64             `json:"createdAt"`
}

// CreateWebhookRequest is the payload for creating a webhook.
type CreateWebhookRequest struct {
	Name        string            `json:"name"`
	Url         string            `json:"url"`
	Method      string            `json:"method"`
	Headers     map[string]string `json:"headers"`
	Template    string            `json:"template"`
	Description string            `json:"description"`
}

// AddResourceWebhookRequest is the payload for attaching a webhook to a resource.
type AddResourceWebhookRequest struct {
	WebhookID string   `json:"webhook_id"`
	Events    []string `json:"events"`
}

// CreateReservationRequest is the payload for creating a reservation.
type CreateReservationRequest struct {
	ResourceID string `json:"resourceId"`
}

// CreateResourceRequest is the payload for creating a resource.
type CreateResourceRequest struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Labels     []string               `json:"labels"`
	Properties map[string]interface{} `json:"properties"`
}

// Secret matches the secret object from the API.
type Secret struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
	CreatedAt   int64  `json:"createdAt"`
	UpdatedAt   int64  `json:"updatedAt"`
}

// CreateSecretRequest is the payload for creating a secret.
type CreateSecretRequest struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

// Health Check Types

// HealthConfig represents health check configuration for a resource.
type HealthConfig struct {
	ResourceID      string `json:"resourceId"`
	Enabled         bool   `json:"enabled"`
	CheckType       string `json:"checkType"`
	Target          string `json:"target"`
	IntervalSeconds int32  `json:"intervalSeconds"`
	TimeoutSeconds  int32  `json:"timeoutSeconds"`
	RetryCount      int32  `json:"retryCount"`
	CreatedAt       int64  `json:"createdAt"`
	UpdatedAt       int64  `json:"updatedAt"`
}

// HealthStatus represents the current health status of a resource.
type HealthStatus struct {
	ID             string `json:"id"`
	ResourceID     string `json:"resourceId"`
	Status         string `json:"status"`
	ResponseTimeMs int32  `json:"responseTimeMs"`
	ErrorMessage   string `json:"errorMessage,omitempty"`
	CheckedAt      int64  `json:"checkedAt"`
}

// HealthConfigRequest is the payload for creating/updating health check configuration.
type HealthConfigRequest struct {
	Enabled         bool   `json:"enabled"`
	CheckType       string `json:"checkType"`
	Target          string `json:"target"`
	IntervalSeconds int32  `json:"intervalSeconds"`
	TimeoutSeconds  int32  `json:"timeoutSeconds"`
	RetryCount      int32  `json:"retryCount"`
}
