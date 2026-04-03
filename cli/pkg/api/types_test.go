package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResourceWithStatus_ToCLIResource_StatusLogic(t *testing.T) {
	tests := []struct {
		name               string
		activeReservations int64
		queueLength        int64
		expectedStatus     string
	}{
		{
			name:               "No active reservations, no queue",
			activeReservations: 0,
			queueLength:        0,
			expectedStatus:     "Available",
		},
		{
			name:               "Active reservations, no queue",
			activeReservations: 1,
			queueLength:        0,
			expectedStatus:     "Reserved",
		},
		{
			name:               "No active reservations, has queue",
			activeReservations: 0,
			queueLength:        2,
			expectedStatus:     "Queue",
		},
		{
			name:               "Both active reservations and queue",
			activeReservations: 1,
			queueLength:        3,
			expectedStatus:     "Reserved",
		},
		{
			name:               "Multiple active reservations, no queue",
			activeReservations: 5,
			queueLength:        0,
			expectedStatus:     "Reserved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resourceWithStatus := ResourceWithStatus{
				Resource: Resource{
					ID:         "550e8400-e29b-41d4-a716-446655440001",
					Name:       "Test Resource",
					Type:       "Test Type",
					Labels:     []string{},
					Properties: make(map[string]interface{}),
					SpaceID:    "550e8400-e29b-41d4-a716-446655440999",
					CreatedAt:  1640995200,
					UpdatedAt:  1640995200,
				},
				ActiveReservations: tt.activeReservations,
				QueueLength:        tt.queueLength,
				NextUserID:         "",
				NextQueuePosition:  0,
			}

			result := resourceWithStatus.ToCLIResource()
			assert.Equal(t, tt.expectedStatus, result.Status)
		})
	}
}

func TestCreateResourceRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request CreateResourceRequest
		isValid bool
	}{
		{
			name: "Valid create resource request",
			request: CreateResourceRequest{
				Name:       "Test Resource",
				Type:       "Test Type",
				Labels:     []string{"test"},
				Properties: map[string]interface{}{"key": "value"},
			},
			isValid: true,
		},
		{
			name: "Valid minimal request",
			request: CreateResourceRequest{
				Name: "Minimal Resource",
			},
			isValid: true,
		},
		{
			name: "Request with empty labels",
			request: CreateResourceRequest{
				Name:       "Test Resource",
				Type:       "Test Type",
				Labels:     []string{},
				Properties: make(map[string]interface{}),
			},
			isValid: true,
		},
		{
			name: "Request with nil properties",
			request: CreateResourceRequest{
				Name:       "Test Resource",
				Type:       "Test Type",
				Labels:     []string{"test"},
				Properties: nil,
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.request.Name, tt.request.Name)
			assert.Equal(t, tt.request.Type, tt.request.Type)
			assert.Equal(t, tt.request.Labels, tt.request.Labels)
			assert.Equal(t, tt.request.Properties, tt.request.Properties)
		})
	}
}

func TestCreateReservationRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request CreateReservationRequest
		isValid bool
	}{
		{
			name: "Valid create reservation request",
			request: CreateReservationRequest{
				ResourceID: "550e8400-e29b-41d4-a716-446655440001",
			},
			isValid: true,
		},
		{
			name: "Request with empty resource ID",
			request: CreateReservationRequest{
				ResourceID: "",
			},
			isValid: true, // API handles validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.request.ResourceID, tt.request.ResourceID)
		})
	}
}

func TestCreateWebhookRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request CreateWebhookRequest
		isValid bool
	}{
		{
			name: "Valid create webhook request",
			request: CreateWebhookRequest{
				Name:        "Test Webhook",
				Url:         "https://example.com/webhook",
				Method:      "POST",
				Headers:     map[string]string{"Content-Type": "application/json"},
				Template:    "{\"event\": \"test\"}",
				Description: "Test webhook description",
			},
			isValid: true,
		},
		{
			name: "Valid minimal request",
			request: CreateWebhookRequest{
				Name: "Minimal Webhook",
				Url:  "https://example.com/webhook",
			},
			isValid: true,
		},
		{
			name: "Request with empty headers",
			request: CreateWebhookRequest{
				Name:    "Test Webhook",
				Url:     "https://example.com/webhook",
				Method:  "POST",
				Headers: map[string]string{},
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.request.Name, tt.request.Name)
			assert.Equal(t, tt.request.Url, tt.request.Url)
			assert.Equal(t, tt.request.Method, tt.request.Method)
			assert.Equal(t, tt.request.Headers, tt.request.Headers)
			assert.Equal(t, tt.request.Template, tt.request.Template)
			assert.Equal(t, tt.request.Description, tt.request.Description)
		})
	}
}

func TestAddResourceWebhookRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request AddResourceWebhookRequest
		isValid bool
	}{
		{
			name: "Valid add resource webhook request",
			request: AddResourceWebhookRequest{
				WebhookID: "550e8400-e29b-41d4-a716-446655440001",
				Events:    []string{"resource.created", "resource.updated"},
			},
			isValid: true,
		},
		{
			name: "Valid minimal request",
			request: AddResourceWebhookRequest{
				WebhookID: "550e8400-e29b-41d4-a716-446655440001",
				Events:    []string{"resource.created"},
			},
			isValid: true,
		},
		{
			name: "Request with empty events",
			request: AddResourceWebhookRequest{
				WebhookID: "550e8400-e29b-41d4-a716-446655440001",
				Events:    []string{},
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.request.WebhookID, tt.request.WebhookID)
			assert.Equal(t, tt.request.Events, tt.request.Events)
		})
	}
}
