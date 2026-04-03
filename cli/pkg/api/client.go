package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/bgentry/go-netrc/netrc"
)

type Client struct {
	BaseURL    string
	Token      string
	HttpClient *http.Client
	RetryCount int
	RetryWait  time.Duration
}

func NewClient(baseURL, token string, useNetrc bool) (*Client, error) {
	client := &Client{
		BaseURL:    baseURL,
		Token:      token,
		HttpClient: &http.Client{Timeout: 30 * time.Second},
		RetryCount: 3,
		RetryWait:  1 * time.Second,
	}

	if client.Token == "" && useNetrc {
		tokenFromNetrc, err := getTokenFromNetrc(baseURL)
		if err == nil && tokenFromNetrc != "" {
			client.Token = tokenFromNetrc
		}
	}

	return client, nil
}

func getTokenFromNetrc(serverURL string) (string, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return "", err
	}
	host := u.Hostname()

	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	netrcPath := filepath.Join(usr.HomeDir, ".netrc")

	// check if file exists
	if _, err := os.Stat(netrcPath); os.IsNotExist(err) {
		return "", nil // no netrc, not an error
	}

	n, err := netrc.ParseFile(netrcPath)
	if err != nil {
		return "", err
	}

	machine := n.FindMachine(host)
	if machine != nil {
		return machine.Password, nil
	}

	return "", nil
}

func (c *Client) request(method, path string, body interface{}) ([]byte, error) {
	var resp *http.Response

	for i := 0; i <= c.RetryCount; i++ {
		var bodyReader io.Reader
		if body != nil {
			// Re-create reader for each attempt
			jsonBody, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			bodyReader = bytes.NewBuffer(jsonBody)
		}

		req, err := http.NewRequest(method, c.BaseURL+path, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		if c.Token != "" {
			req.Header.Set("Authorization", "Bearer "+c.Token)
		}

		resp, err = c.HttpClient.Do(req)
		if err == nil && resp.StatusCode < 500 {
			// Success or non-retryable error (4xx)
			break
		}

		// If it's the last attempt, return the error
		if i == c.RetryCount {
			if err != nil {
				return nil, fmt.Errorf("request failed after %d retries: %w", c.RetryCount, err)
			}
			// If we got here, it means err was nil but StatusCode >= 500
			// We'll let the status code check below handle it
			break
		}

		// Wait before retrying (exponential backoff)
		sleepTime := c.RetryWait * time.Duration(1<<i)
		time.Sleep(sleepTime)
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("api error: %s (status: %d)", string(respBody), resp.StatusCode)
	}

	return respBody, nil
}

func (c *Client) GetResources(labelExpr string) ([]Resource, error) {
	path := "/api/resources/with-status"
	if labelExpr != "" {
		path += "?label_expr=" + url.QueryEscape(labelExpr)
	}

	resp, err := c.request("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var resourcesWithStatus []ResourceWithStatus
	if err := json.Unmarshal(resp, &resourcesWithStatus); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resources: %w", err)
	}

	resources := make([]Resource, len(resourcesWithStatus))
	for i, rws := range resourcesWithStatus {
		resources[i] = rws.ToCLIResource()
	}

	return resources, nil
}

func (c *Client) CreateResource(req CreateResourceRequest) (*Resource, error) {
	resp, err := c.request("POST", "/api/resources", req)
	if err != nil {
		return nil, err
	}

	var resource Resource
	if err := json.Unmarshal(resp, &resource); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resource: %w", err)
	}
	return &resource, nil
}

func (c *Client) DeleteResource(id string) error {
	_, err := c.request("DELETE", fmt.Sprintf("/api/resources/%s", id), nil)
	return err
}

func (c *Client) CreateReservation(resourceID string, duration string) (*Reservation, error) {
	var resp []byte
	var err error

	if duration != "" {
		type CreateTimedReservationRequest struct {
			ResourceID string `json:"resourceId"`
			Duration   string `json:"duration"`
		}
		req := CreateTimedReservationRequest{ResourceID: resourceID, Duration: duration}
		resp, err = c.request("POST", "/api/reservations/timed", req)
	} else {
		req := CreateReservationRequest{ResourceID: resourceID}
		resp, err = c.request("POST", "/api/reservations", req)
	}

	if err != nil {
		return nil, err
	}

	var reservation Reservation
	if err := json.Unmarshal(resp, &reservation); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reservation: %w", err)
	}
	return &reservation, nil
}

func (c *Client) CompleteReservation(reservationID string) error {
	_, err := c.request("PATCH", fmt.Sprintf("/api/reservations/%s/complete", reservationID), nil)
	return err
}

func (c *Client) CancelReservation(reservationID string) error {
	_, err := c.request("PATCH", fmt.Sprintf("/api/reservations/%s/cancel", reservationID), nil)
	return err
}

func (c *Client) GetUserReservations() ([]UserReservation, error) {
	resp, err := c.request("GET", "/api/reservations", nil)
	if err != nil {
		return nil, err
	}

	var reservations []UserReservation
	if err := json.Unmarshal(resp, &reservations); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reservations: %w", err)
	}
	return reservations, nil
}

// Webhook Methods

func (c *Client) ListWebhooks() ([]Webhook, error) {
	resp, err := c.request("GET", "/api/webhooks", nil)
	if err != nil {
		return nil, err
	}

	var webhooks []Webhook
	if err := json.Unmarshal(resp, &webhooks); err != nil {
		return nil, fmt.Errorf("failed to unmarshal webhooks: %w", err)
	}
	return webhooks, nil
}

func (c *Client) CreateWebhook(req CreateWebhookRequest) (*Webhook, error) {
	resp, err := c.request("POST", "/api/webhooks", req)
	if err != nil {
		return nil, err
	}

	var webhook Webhook
	if err := json.Unmarshal(resp, &webhook); err != nil {
		return nil, fmt.Errorf("failed to unmarshal webhook: %w", err)
	}
	return &webhook, nil
}

func (c *Client) DeleteWebhook(id string) error {
	_, err := c.request("DELETE", fmt.Sprintf("/api/webhooks/%s", id), nil)
	return err
}

func (c *Client) AttachWebhook(resourceID string, req AddResourceWebhookRequest) error {
	_, err := c.request("POST", fmt.Sprintf("/api/resources/%s/webhooks", resourceID), req)
	return err
}

func (c *Client) DetachWebhook(resourceID string, webhookID string) error {
	_, err := c.request("DELETE", fmt.Sprintf("/api/resources/%s/webhooks/%s", resourceID, webhookID), nil)
	return err
}

// Secret Methods

func (c *Client) ListSecrets() ([]Secret, error) {
	resp, err := c.request("GET", "/api/secrets", nil)
	if err != nil {
		return nil, err
	}

	var secrets []Secret
	if err := json.Unmarshal(resp, &secrets); err != nil {
		return nil, fmt.Errorf("failed to unmarshal secrets: %w", err)
	}
	return secrets, nil
}

func (c *Client) CreateSecret(req CreateSecretRequest) (*Secret, error) {
	resp, err := c.request("POST", "/api/secrets", req)
	if err != nil {
		return nil, err
	}

	var secret Secret
	if err := json.Unmarshal(resp, &secret); err != nil {
		return nil, fmt.Errorf("failed to unmarshal secret: %w", err)
	}
	return &secret, nil
}

func (c *Client) UpdateSecret(id string, value, description string) (*Secret, error) {
	type UpdateSecretRequest struct {
		Value       string `json:"value"`
		Description string `json:"description"`
	}
	req := UpdateSecretRequest{Value: value, Description: description}
	resp, err := c.request("PUT", fmt.Sprintf("/api/secrets/%s", id), req)
	if err != nil {
		return nil, err
	}

	var secret Secret
	if err := json.Unmarshal(resp, &secret); err != nil {
		return nil, fmt.Errorf("failed to unmarshal secret: %w", err)
	}
	return &secret, nil
}

func (c *Client) DeleteSecret(id string) error {
	_, err := c.request("DELETE", fmt.Sprintf("/api/secrets/%s", id), nil)
	return err
}

// Health Check Methods

func (c *Client) GetHealthConfig(resourceID string) (*HealthConfig, error) {
	resp, err := c.request("GET", fmt.Sprintf("/api/resources/%s/health/config", resourceID), nil)
	if err != nil {
		return nil, err
	}

	var config HealthConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal health config: %w", err)
	}
	return &config, nil
}

func (c *Client) UpsertHealthConfig(resourceID string, req HealthConfigRequest) (*HealthConfig, error) {
	resp, err := c.request("PUT", fmt.Sprintf("/api/resources/%s/health/config", resourceID), req)
	if err != nil {
		return nil, err
	}

	var config HealthConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal health config: %w", err)
	}
	return &config, nil
}

func (c *Client) DeleteHealthConfig(resourceID string) error {
	_, err := c.request("DELETE", fmt.Sprintf("/api/resources/%s/health/config", resourceID), nil)
	return err
}

func (c *Client) GetHealthStatus(resourceID string) (*HealthStatus, error) {
	resp, err := c.request("GET", fmt.Sprintf("/api/resources/%s/health/status", resourceID), nil)
	if err != nil {
		return nil, err
	}

	var status HealthStatus
	if err := json.Unmarshal(resp, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal health status: %w", err)
	}
	return &status, nil
}

func (c *Client) GetHealthHistory(resourceID string, limit int32) ([]HealthStatus, error) {
	path := fmt.Sprintf("/api/resources/%s/health/history?limit=%d", resourceID, limit)
	resp, err := c.request("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var history []HealthStatus
	if err := json.Unmarshal(resp, &history); err != nil {
		return nil, fmt.Errorf("failed to unmarshal health history: %w", err)
	}
	return history, nil
}

func (c *Client) TriggerHealthCheck(resourceID string) error {
	_, err := c.request("POST", fmt.Sprintf("/api/resources/%s/health/trigger", resourceID), nil)
	return err
}

func (c *Client) GetReservation(id string) (*Reservation, error) {
	resp, err := c.request("GET", fmt.Sprintf("/api/reservations/%s", id), nil)
	if err != nil {
		return nil, err
	}

	var reservation Reservation
	if err := json.Unmarshal(resp, &reservation); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reservation: %w", err)
	}
	return &reservation, nil
}

func (c *Client) WaitForReservation(id string, timeout, interval int, progressFn func(status string, queuePos *int32)) error {
	elapsed := 0

	for elapsed < timeout {
		reservation, err := c.GetReservation(id)
		if err != nil {
			return fmt.Errorf("failed to get reservation status: %w", err)
		}

		// Call progress callback if provided
		if progressFn != nil {
			progressFn(reservation.Status, reservation.QueuePosition)
		}

		// Check if reservation is active
		if reservation.Status == "active" {
			return nil
		}

		// Check if reservation was cancelled or completed
		if reservation.Status == "cancelled" || reservation.Status == "completed" {
			return fmt.Errorf("reservation was %s", reservation.Status)
		}

		// Wait before next poll
		if elapsed+interval > timeout {
			// Don't sleep past timeout
			break
		}

		// Sleep for interval (in seconds)
		sleepDuration := interval
		if elapsed+interval > timeout {
			sleepDuration = timeout - elapsed
		}

		time.Sleep(time.Duration(sleepDuration) * time.Second)
		elapsed += interval
	}

	return fmt.Errorf("timeout waiting for reservation to become active")
}

// Backup Methods

// CreateBackup downloads the backup JSON from the server.
func (c *Client) CreateBackup() ([]byte, error) {
	return c.request("GET", "/api/admin/backup", nil)
}

// RestoreBackup uploads a backup JSON file to the server.
func (c *Client) RestoreBackup(data []byte) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "backup.json")
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return fmt.Errorf("failed to write backup data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/api/admin/restore", body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("restore failed: %s (status: %d)", string(respBody), resp.StatusCode)
	}

	return nil
}
