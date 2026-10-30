package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bgentry/go-netrc/netrc"
)

type Client struct {
	BaseURL    string
	Token      string
	HttpClient *http.Client
	RetryCount int
	RetryWait  time.Duration
	Context    context.Context
}

func NewClient(baseURL, token string, useNetrc bool) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("server URL must be an absolute HTTP(S) URL without credentials, query or fragment")
	}
	client := &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Context:    context.Background(),
		Token:      token,
		HttpClient: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }},
		RetryCount: 3,
		RetryWait:  1 * time.Second,
	}

	if client.Token == "" && useNetrc {
		tokenFromNetrc, err := getTokenFromNetrc(baseURL)
		if err != nil {
			return nil, err
		}
		client.Token = tokenFromNetrc
	}

	return client, nil
}

func getTokenFromNetrc(serverURL string) (string, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return "", err
	}
	host := u.Hostname()

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	netrcPath := filepath.Join(home, ".netrc")
	info, err := os.Stat(netrcPath)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return "", fmt.Errorf(".netrc requires owner-only permissions")
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
	return c.requestHeaders(method, path, body, nil)
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

func (c *Client) GetResource(id string) (*Resource, error) {
	resp, err := c.request("GET", fmt.Sprintf("/api/resources/%s", id), nil)
	if err != nil {
		return nil, err
	}

	var resource Resource
	if err := json.Unmarshal(resp, &resource); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resource: %w", err)
	}
	return &resource, nil
}

func (c *Client) UpdateResource(id string, req UpdateResourceRequest) (*Resource, error) {
	resp, err := c.request("PATCH", fmt.Sprintf("/api/resources/%s", id), req)
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
	_, err := c.request("POST", fmt.Sprintf("/api/resources/%s/health/check", resourceID), nil)
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
	return c.waitForReservation(id, timeout, interval, progressFn)
}

// Space Methods

func (c *Client) ListSpaces() ([]Space, error) {
	resp, err := c.request("GET", "/api/spaces", nil)
	if err != nil {
		return nil, err
	}
	var spaces []Space
	if err := json.Unmarshal(resp, &spaces); err != nil {
		return nil, fmt.Errorf("failed to unmarshal spaces: %w", err)
	}
	return spaces, nil
}

func (c *Client) CreateSpace(name, description string) (*Space, error) {
	req := map[string]string{"name": name, "description": description}
	resp, err := c.request("POST", "/api/spaces", req)
	if err != nil {
		return nil, err
	}
	var space Space
	if err := json.Unmarshal(resp, &space); err != nil {
		return nil, fmt.Errorf("failed to unmarshal space: %w", err)
	}
	return &space, nil
}

func (c *Client) UpdateSpace(id, name, description string) (*Space, error) {
	req := map[string]string{"name": name, "description": description}
	resp, err := c.request("PATCH", fmt.Sprintf("/api/spaces/%s", id), req)
	if err != nil {
		return nil, err
	}
	var space Space
	if err := json.Unmarshal(resp, &space); err != nil {
		return nil, fmt.Errorf("failed to unmarshal space: %w", err)
	}
	return &space, nil
}

func (c *Client) DeleteSpace(id string) error {
	_, err := c.request("DELETE", fmt.Sprintf("/api/spaces/%s", id), nil)
	return err
}

// Group Methods

func (c *Client) ListGroups() ([]Group, error) {
	resp, err := c.request("GET", "/api/groups", nil)
	if err != nil {
		return nil, err
	}
	var groups []Group
	if err := json.Unmarshal(resp, &groups); err != nil {
		return nil, fmt.Errorf("failed to unmarshal groups: %w", err)
	}
	return groups, nil
}

func (c *Client) CreateGroup(name, description string) (*Group, error) {
	req := map[string]string{"name": name, "description": description}
	resp, err := c.request("POST", "/api/groups", req)
	if err != nil {
		return nil, err
	}
	var group Group
	if err := json.Unmarshal(resp, &group); err != nil {
		return nil, fmt.Errorf("failed to unmarshal group: %w", err)
	}
	return &group, nil
}

func (c *Client) UpdateGroup(id, name, description string) (*Group, error) {
	req := map[string]string{"name": name, "description": description}
	resp, err := c.request("PUT", fmt.Sprintf("/api/groups/%s", id), req)
	if err != nil {
		return nil, err
	}
	var group Group
	if err := json.Unmarshal(resp, &group); err != nil {
		return nil, fmt.Errorf("failed to unmarshal group: %w", err)
	}
	return &group, nil
}

func (c *Client) DeleteGroup(id string) error {
	_, err := c.request("DELETE", fmt.Sprintf("/api/groups/%s", id), nil)
	return err
}

func (c *Client) ListGroupMembers(groupID string) ([]GroupMember, error) {
	resp, err := c.request("GET", fmt.Sprintf("/api/groups/%s/members", groupID), nil)
	if err != nil {
		return nil, err
	}
	var members []GroupMember
	if err := json.Unmarshal(resp, &members); err != nil {
		return nil, fmt.Errorf("failed to unmarshal group members: %w", err)
	}
	return members, nil
}

func (c *Client) AddGroupMember(groupID, userID string) error {
	req := map[string]string{"userId": userID}
	_, err := c.request("POST", fmt.Sprintf("/api/groups/%s/members", groupID), req)
	return err
}

func (c *Client) RemoveGroupMember(groupID, userID string) error {
	_, err := c.request("DELETE", fmt.Sprintf("/api/groups/%s/members/%s", groupID, userID), nil)
	return err
}

// Token Methods

func (c *Client) ListTokens() ([]APIToken, error) {
	resp, err := c.request("GET", "/api/tokens", nil)
	if err != nil {
		return nil, err
	}
	var tokens []APIToken
	if err := json.Unmarshal(resp, &tokens); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tokens: %w", err)
	}
	return tokens, nil
}

func (c *Client) GenerateToken(name, expiresIn string) (*GenerateTokenResponse, error) {
	req := map[string]string{"name": name}
	if expiresIn != "" {
		req["expires_in"] = expiresIn
	}
	resp, err := c.request("POST", "/api/tokens", req)
	if err != nil {
		return nil, err
	}
	var token GenerateTokenResponse
	if err := json.Unmarshal(resp, &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}
	return &token, nil
}

func (c *Client) RevokeToken(id string) error {
	_, err := c.request("DELETE", fmt.Sprintf("/api/tokens/%s", id), nil)
	return err
}

// Audit Log Methods

func (c *Client) GetAuditLogs(limit, offset int) ([]AuditLog, error) {
	path := fmt.Sprintf("/api/admin/audit-logs?limit=%d&offset=%d", limit, offset)
	resp, err := c.request("GET", path, nil)
	if err != nil {
		return nil, err
	}
	var logs []AuditLog
	if err := json.Unmarshal(resp, &logs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal audit logs: %w", err)
	}
	return logs, nil
}

// History Methods

func (c *Client) GetUserHistory() ([]ReservationHistory, error) {
	resp, err := c.request("GET", "/api/me/history", nil)
	if err != nil {
		return nil, err
	}
	var history []ReservationHistory
	if err := json.Unmarshal(resp, &history); err != nil {
		return nil, fmt.Errorf("failed to unmarshal history: %w", err)
	}
	return history, nil
}

func (c *Client) GetResourceHistory(resourceID string) ([]ReservationHistory, error) {
	resp, err := c.request("GET", fmt.Sprintf("/api/resources/%s/history", resourceID), nil)
	if err != nil {
		return nil, err
	}
	var history []ReservationHistory
	if err := json.Unmarshal(resp, &history); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resource history: %w", err)
	}
	return history, nil
}

// Maintenance Methods

func (c *Client) SetMaintenance(resourceID string, enabled bool, reason string) error {
	req := map[string]interface{}{"is_under_maintenance": enabled, "reason": reason}
	_, err := c.request("PUT", fmt.Sprintf("/api/resources/%s/maintenance", resourceID), req)
	return err
}

func (c *Client) GetMaintenanceHistory(resourceID string) ([]MaintenanceHistory, error) {
	resp, err := c.request("GET", fmt.Sprintf("/api/resources/%s/maintenance/history", resourceID), nil)
	if err != nil {
		return nil, err
	}
	var history []MaintenanceHistory
	if err := json.Unmarshal(resp, &history); err != nil {
		return nil, fmt.Errorf("failed to unmarshal maintenance history: %w", err)
	}
	return history, nil
}

// UpdateWebhook updates an existing webhook.
func (c *Client) UpdateWebhook(id string, req UpdateWebhookRequest) (*Webhook, error) {
	resp, err := c.request("PUT", fmt.Sprintf("/api/webhooks/%s", id), req)
	if err != nil {
		return nil, err
	}
	var webhook Webhook
	if err := json.Unmarshal(resp, &webhook); err != nil {
		return nil, fmt.Errorf("failed to unmarshal webhook: %w", err)
	}
	return &webhook, nil
}
