package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test fixtures
var (
	testResource1 = Resource{
		ID:         "550e8400-e29b-41d4-a716-446655440001",
		Name:       "Meeting Room A",
		Type:       "Conference Room",
		Labels:     []string{"projector", "whiteboard"},
		Properties: map[string]interface{}{"capacity": 10},
		SpaceID:    "550e8400-e29b-41d4-a716-446655440999",
		CreatedAt:  1640995200,
		UpdatedAt:  1640995200,
		Status:     "Available",
	}

	testResource2 = Resource{
		ID:         "550e8400-e29b-41d4-a716-446655440002",
		Name:       "Development Server",
		Type:       "Server",
		Labels:     []string{"linux", "dev"},
		Properties: map[string]interface{}{"cpu": "8 cores", "memory": "16GB"},
		SpaceID:    "550e8400-e29b-41d4-a716-446655440999",
		CreatedAt:  1640995200,
		UpdatedAt:  1640995200,
		Status:     "Reserved",
	}

	resourcesListResponse = `[` +
		`{"resource":{"id":"550e8400-e29b-41d4-a716-446655440001","name":"Meeting Room A","type":"Conference Room","labels":["projector","whiteboard"],"properties":{"capacity":10},"spaceId":"550e8400-e29b-41d4-a716-446655440999","createdAt":1640995200,"updatedAt":1640995200},"activeReservations":0,"queueLength":0,"nextUserId":"","nextQueuePosition":0},` +
		`{"resource":{"id":"550e8400-e29b-41d4-a716-446655440002","name":"Development Server","type":"Server","labels":["linux","dev"],"properties":{"cpu":"8 cores","memory":"16GB"},"spaceId":"550e8400-e29b-41d4-a716-446655440999","createdAt":1640995200,"updatedAt":1640995200},"activeReservations":1,"queueLength":0,"nextUserId":"","nextQueuePosition":0}` +
		`]`

	singleResourceResponse = `{"id":"550e8400-e29b-41d4-a716-446655440001","name":"Meeting Room A","type":"Conference Room","labels":["projector","whiteboard"],"properties":{"capacity":10},"spaceId":"550e8400-e29b-41d4-a716-446655440999","createdAt":1640995200,"updatedAt":1640995200,"status":"Available"}`

	reservationResponse = `{"id":"550e8400-e29b-41d4-a716-446655440001","resourceId":"550e8400-e29b-41d4-a716-446655440001","userId":"550e8400-e29b-41d4-a716-446655440123","status":"active","queuePosition":null,"startTime":1640995200,"endTime":1640998800,"createdAt":1640995200,"updatedAt":1640995200,"scheduledEndTime":"2021-12-31T17:00:00Z"}`

	userReservationsResponse = `[` +
		`{"id":"550e8400-e29b-41d4-a716-446655440001","resourceId":"550e8400-e29b-41d4-a716-446655440001","userId":"550e8400-e29b-41d4-a716-446655440123","status":"active","queuePosition":null,"startTime":1640995200,"endTime":1640998800,"createdAt":1640995200,"resourceName":"Meeting Room A","resourceType":"Conference Room"},` +
		`{"id":"550e8400-e29b-41d4-a716-446655440002","resourceId":"550e8400-e29b-41d4-a716-446655440002","userId":"550e8400-e29b-41d4-a716-446655440123","status":"queued","queuePosition":2,"startTime":0,"endTime":0,"createdAt":1640995300,"resourceName":"Development Server","resourceType":"Server"}` +
		`]`

	webhooksResponse = `[` +
		`{"id":"550e8400-e29b-41d4-a716-446655440001","name":"Resource Created Webhook","url":"https://example.com/webhooks/resource-created","method":"POST","headers":{"Content-Type":"application/json"},"template":"{\"event\": \"resource.created\", \"data\": {{.}}}","description":"Notifies when a resource is created","signingSecret":"secret123","createdAt":1640995200}` +
		`]`

	createResourceRequestResponse = `{"id":"550e8400-e29b-41d4-a716-446655440003","name":"New Resource","type":"Test Type","labels":["test"],"properties":{"key":"value"},"spaceId":"550e8400-e29b-41d4-a716-446655440999","createdAt":1640995200,"updatedAt":1640995200}`

	unauthorizedResponse    = `{"error":"Unauthorized"}`
	notFoundResponse        = `{"error":"Resource not found"}`
	validationErrorResponse = `{"error":"Validation failed","details":{"name":"Name is required"}}`
)

// helper function to create test client
func createTestClientWithResponse(response string, statusCode int) (*Client, *httptest.Server) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		w.Write([]byte(response))
	})

	server := httptest.NewServer(handler)

	client, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		panic("Failed to create test client: " + err.Error())
	}

	// Use the server's client
	client.HttpClient = server.Client()

	return client, server
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		token    string
		useNetrc bool
		wantErr  bool
	}{
		{
			name:     "Valid client with token",
			baseURL:  "http://example.com",
			token:    "test-token",
			useNetrc: false,
			wantErr:  false,
		},
		{
			name:     "Valid client without token",
			baseURL:  "http://example.com",
			token:    "",
			useNetrc: false,
			wantErr:  false,
		},
		{
			name:     "Valid client with netrc enabled",
			baseURL:  "http://example.com",
			token:    "",
			useNetrc: true,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.baseURL, tt.token, tt.useNetrc)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.Equal(t, tt.baseURL, client.BaseURL)
				assert.Equal(t, tt.token, client.Token)
				assert.NotNil(t, client.HttpClient)
			}
		})
	}
}

func TestGetTokenFromNetrc(t *testing.T) {
	tests := []struct {
		name      string
		serverURL string
		wantErr   bool
		wantToken string
	}{
		{
			name:      "Invalid URL - missing scheme",
			serverURL: "://missing-scheme.com",
			wantErr:   true,
			wantToken: "",
		},
		{
			name:      "Valid URL without netrc",
			serverURL: "http://example.com",
			wantErr:   false,
			wantToken: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := getTokenFromNetrc(tt.serverURL)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantToken, token)
		})
	}
}

func TestClient_GetResources(t *testing.T) {
	tests := []struct {
		name           string
		response       string
		statusCode     int
		expectedCount  int
		wantErr        bool
		expectedStatus []string
	}{
		{
			name:           "Successful resource list",
			response:       resourcesListResponse,
			statusCode:     http.StatusOK,
			expectedCount:  2,
			wantErr:        false,
			expectedStatus: []string{"Available", "Reserved"},
		},
		{
			name:       "API error response",
			response:   unauthorizedResponse,
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server := createTestClientWithResponse(tt.response, tt.statusCode)
			defer server.Close()

			resources, err := client.GetResources("")

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resources)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resources)
				assert.Len(t, resources, tt.expectedCount)

				if len(tt.expectedStatus) > 0 {
					for i, expectedStatus := range tt.expectedStatus {
						if i < len(resources) {
							assert.Equal(t, expectedStatus, resources[i].Status)
						}
					}
				}
			}
		})
	}
}

func TestClient_CreateResource(t *testing.T) {
	tests := []struct {
		name       string
		request    CreateResourceRequest
		response   string
		statusCode int
		wantErr    bool
	}{
		{
			name: "Successful resource creation",
			request: CreateResourceRequest{
				Name:       "New Resource",
				Type:       "Test Type",
				Labels:     []string{"test"},
				Properties: map[string]interface{}{"key": "value"},
			},
			response:   createResourceRequestResponse,
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name: "Validation error",
			request: CreateResourceRequest{
				Name: "", // Invalid - empty name
				Type: "Test Type",
			},
			response:   validationErrorResponse,
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server := createTestClientWithResponse(tt.response, tt.statusCode)
			defer server.Close()

			resource, err := client.CreateResource(tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resource)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resource)
				assert.Equal(t, "New Resource", resource.Name)
				assert.Equal(t, "Test Type", resource.Type)
			}
		})
	}
}

func TestClient_DeleteResource(t *testing.T) {
	tests := []struct {
		name       string
		resourceID string
		response   string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "Successful resource deletion",
			resourceID: "550e8400-e29b-41d4-a716-446655440001",
			response:   "",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "Resource not found",
			resourceID: "non-existent",
			response:   notFoundResponse,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server := createTestClientWithResponse(tt.response, tt.statusCode)
			defer server.Close()

			err := client.DeleteResource(tt.resourceID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClient_CreateReservation(t *testing.T) {
	tests := []struct {
		name           string
		resourceID     string
		duration       string
		response       string
		statusCode     int
		wantErr        bool
		expectedStatus string
	}{
		{
			name:           "Successful reservation creation",
			resourceID:     "550e8400-e29b-41d4-a716-446655440001",
			duration:       "",
			response:       reservationResponse,
			statusCode:     http.StatusOK,
			wantErr:        false,
			expectedStatus: "active",
		},
		{
			name:       "Resource not found",
			resourceID: "non-existent",
			duration:   "",
			response:   notFoundResponse,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server := createTestClientWithResponse(tt.response, tt.statusCode)
			defer server.Close()

			reservation, err := client.CreateReservation(tt.resourceID, tt.duration)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, reservation)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, reservation)
				assert.Equal(t, tt.resourceID, reservation.ResourceID)
				assert.Equal(t, tt.expectedStatus, reservation.Status)
			}
		})
	}
}

func TestClient_CompleteReservation(t *testing.T) {
	tests := []struct {
		name          string
		reservationID string
		response      string
		statusCode    int
		wantErr       bool
	}{
		{
			name:          "Successful reservation completion",
			reservationID: "550e8400-e29b-41d4-a716-446655440001",
			response:      "",
			statusCode:    http.StatusOK,
			wantErr:       false,
		},
		{
			name:          "Reservation not found",
			reservationID: "non-existent",
			response:      notFoundResponse,
			statusCode:    http.StatusNotFound,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server := createTestClientWithResponse(tt.response, tt.statusCode)
			defer server.Close()

			err := client.CompleteReservation(tt.reservationID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClient_CancelReservation(t *testing.T) {
	tests := []struct {
		name          string
		reservationID string
		response      string
		statusCode    int
		wantErr       bool
	}{
		{
			name:          "Successful reservation cancellation",
			reservationID: "550e8400-e29b-41d4-a716-446655440001",
			response:      "",
			statusCode:    http.StatusOK,
			wantErr:       false,
		},
		{
			name:          "Reservation not found",
			reservationID: "non-existent",
			response:      notFoundResponse,
			statusCode:    http.StatusNotFound,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server := createTestClientWithResponse(tt.response, tt.statusCode)
			defer server.Close()

			err := client.CancelReservation(tt.reservationID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClient_GetUserReservations(t *testing.T) {
	tests := []struct {
		name          string
		response      string
		statusCode    int
		expectedCount int
		wantErr       bool
	}{
		{
			name:          "Successful user reservations list",
			response:      userReservationsResponse,
			statusCode:    http.StatusOK,
			expectedCount: 2,
			wantErr:       false,
		},
		{
			name:       "Unauthorized response",
			response:   unauthorizedResponse,
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server := createTestClientWithResponse(tt.response, tt.statusCode)
			defer server.Close()

			reservations, err := client.GetUserReservations()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, reservations)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, reservations)
				assert.Len(t, reservations, tt.expectedCount)
			}
		})
	}
}

func TestClient_ListWebhooks(t *testing.T) {
	tests := []struct {
		name          string
		response      string
		statusCode    int
		expectedCount int
		wantErr       bool
	}{
		{
			name:          "Successful webhook list",
			response:      webhooksResponse,
			statusCode:    http.StatusOK,
			expectedCount: 1,
			wantErr:       false,
		},
		{
			name:       "API error response",
			response:   unauthorizedResponse,
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server := createTestClientWithResponse(tt.response, tt.statusCode)
			defer server.Close()

			webhooks, err := client.ListWebhooks()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, webhooks)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, webhooks)
				assert.Len(t, webhooks, tt.expectedCount)
			}
		})
	}
}

func TestClient_CreateWebhook(t *testing.T) {
	createWebhookRequestResponse := `{"id":"550e8400-e29b-41d4-a716-446655440002","name":"New Webhook","url":"https://example.com/new-webhook","method":"POST","headers":{},"template":"","description":"","signingSecret":"newsecret","createdAt":1640995200}`

	tests := []struct {
		name       string
		request    CreateWebhookRequest
		response   string
		statusCode int
		wantErr    bool
	}{
		{
			name: "Successful webhook creation",
			request: CreateWebhookRequest{
				Name:        "New Webhook",
				Url:         "https://example.com/new-webhook",
				Method:      "POST",
				Headers:     map[string]string{},
				Template:    "",
				Description: "",
			},
			response:   createWebhookRequestResponse,
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name: "Validation error",
			request: CreateWebhookRequest{
				Name: "", // Invalid - empty name
				Url:  "https://example.com/webhook",
			},
			response:   validationErrorResponse,
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server := createTestClientWithResponse(tt.response, tt.statusCode)
			defer server.Close()

			webhook, err := client.CreateWebhook(tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, webhook)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, webhook)
				assert.Equal(t, "New Webhook", webhook.Name)
				assert.Equal(t, "https://example.com/new-webhook", webhook.Url)
			}
		})
	}
}

func TestClient_DeleteWebhook(t *testing.T) {
	tests := []struct {
		name       string
		webhookID  string
		response   string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "Successful webhook deletion",
			webhookID:  "550e8400-e29b-41d4-a716-446655440001",
			response:   "",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "Webhook not found",
			webhookID:  "non-existent",
			response:   notFoundResponse,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server := createTestClientWithResponse(tt.response, tt.statusCode)
			defer server.Close()

			err := client.DeleteWebhook(tt.webhookID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClient_AttachWebhook(t *testing.T) {
	tests := []struct {
		name       string
		resourceID string
		request    AddResourceWebhookRequest
		response   string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "Successful webhook attachment",
			resourceID: "550e8400-e29b-41d4-a716-446655440001",
			request: AddResourceWebhookRequest{
				WebhookID: "550e8400-e29b-41d4-a716-446655440001",
				Events:    []string{"resource.created"},
			},
			response:   "",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "Resource not found",
			resourceID: "non-existent",
			request: AddResourceWebhookRequest{
				WebhookID: "550e8400-e29b-41d4-a716-446655440001",
				Events:    []string{"resource.created"},
			},
			response:   notFoundResponse,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server := createTestClientWithResponse(tt.response, tt.statusCode)
			defer server.Close()

			err := client.AttachWebhook(tt.resourceID, tt.request)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClient_DetachWebhook(t *testing.T) {
	tests := []struct {
		name       string
		resourceID string
		webhookID  string
		response   string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "Successful webhook detachment",
			resourceID: "550e8400-e29b-41d4-a716-446655440001",
			webhookID:  "550e8400-e29b-41d4-a716-446655440001",
			response:   "",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "Resource not found",
			resourceID: "non-existent",
			webhookID:  "550e8400-e29b-41d4-a716-446655440001",
			response:   notFoundResponse,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server := createTestClientWithResponse(tt.response, tt.statusCode)
			defer server.Close()

			err := client.DetachWebhook(tt.resourceID, tt.webhookID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClient_Secrets(t *testing.T) {
	createSecretResponse := `{"id":"550e8400-e29b-41d4-a716-446655440001","key":"TEST_KEY","value":"****","description":"Test secret","createdAt":1640995200,"updatedAt":1640995200}`
	listSecretsResponse := `[{"id":"550e8400-e29b-41d4-a716-446655440001","key":"TEST_KEY","value":"****","description":"Test secret","createdAt":1640995200,"updatedAt":1640995200}]`

	t.Run("CreateSecret", func(t *testing.T) {
		client, server := createTestClientWithResponse(createSecretResponse, http.StatusOK)
		defer server.Close()

		req := CreateSecretRequest{
			Key:         "TEST_KEY",
			Value:       "secret_value",
			Description: "Test secret",
		}

		secret, err := client.CreateSecret(req)
		assert.NoError(t, err)
		assert.NotNil(t, secret)
		assert.Equal(t, "TEST_KEY", secret.Key)
	})

	t.Run("ListSecrets", func(t *testing.T) {
		client, server := createTestClientWithResponse(listSecretsResponse, http.StatusOK)
		defer server.Close()

		secrets, err := client.ListSecrets()
		assert.NoError(t, err)
		assert.Len(t, secrets, 1)
		assert.Equal(t, "TEST_KEY", secrets[0].Key)
	})

	t.Run("DeleteSecret", func(t *testing.T) {
		client, server := createTestClientWithResponse("", http.StatusOK)
		defer server.Close()

		err := client.DeleteSecret("550e8400-e29b-41d4-a716-446655440001")
		assert.NoError(t, err)
	})
}
