package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"text/template"
	"time"

	"github.com/google/uuid"

	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/thetaqitahmid/claimctl/internal/db"
)

type WebhookService struct {
	q             db.Querier
	secretService *SecretService
	httpClient    *http.Client
}

func NewWebhookService(q db.Querier, secretService *SecretService) *WebhookService {
	return &WebhookService{
		q:             q,
		secretService: secretService,
		httpClient:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *WebhookService) CreateWebhook(ctx context.Context, name, url, method string, headers map[string]string, tmpl, description string) (db.ClaimctlWebhook, error) {
	headersJSON := []byte("{}")
	if headers != nil {
		var err error
		headersJSON, err = json.Marshal(headers)
		if err != nil {
			return db.ClaimctlWebhook{}, err
		}
	}

	// Generate Signing Secret
	signingSecretBytes := make([]byte, 32)
	_, _ = rand.Read(signingSecretBytes) // If this fails, we have bigger problems, but ignoring err for brevity/randomness
	signingSecret := hex.EncodeToString(signingSecretBytes)

	return s.q.CreateWebhook(ctx, db.CreateWebhookParams{
		Name:          name,
		Url:           url,
		Method:        method,
		Headers:       headersJSON,
		Template:      pgtype.Text{String: tmpl, Valid: tmpl != ""},
		Description:   pgtype.Text{String: description, Valid: description != ""},
		SigningSecret: signingSecret,
	})
}

func (s *WebhookService) GetWebhook(ctx context.Context, id uuid.UUID) (db.ClaimctlWebhook, error) {
	return s.q.GetWebhook(ctx, id)
}

func (s *WebhookService) ListWebhooks(ctx context.Context) ([]db.ClaimctlWebhook, error) {
	return s.q.ListWebhooks(ctx)
}

func (s *WebhookService) UpdateWebhook(ctx context.Context, id uuid.UUID, name, url, method string, headers map[string]string, tmpl, description string) (db.ClaimctlWebhook, error) {
	headersJSON := []byte("{}")
	if headers != nil {
		var err error
		headersJSON, err = json.Marshal(headers)
		if err != nil {
			return db.ClaimctlWebhook{}, err
		}
	}
	return s.q.UpdateWebhook(ctx, db.UpdateWebhookParams{
		ID:          id,
		Name:        name,
		Url:         url,
		Method:      method,
		Headers:     headersJSON,
		Template:    pgtype.Text{String: tmpl, Valid: tmpl != ""},
		Description: pgtype.Text{String: description, Valid: description != ""},
	})
}

func (s *WebhookService) DeleteWebhook(ctx context.Context, id uuid.UUID) error {
	return s.q.DeleteWebhook(ctx, id)
}

func (s *WebhookService) AddResourceWebhook(ctx context.Context, resourceID, webhookID uuid.UUID, events []string) error {
	return s.q.AddResourceWebhook(ctx, db.AddResourceWebhookParams{
		ResourceID: resourceID,
		WebhookID:  webhookID,
		Events:     events,
	})
}

func (s *WebhookService) RemoveResourceWebhook(ctx context.Context, resourceID, webhookID uuid.UUID) error {
	return s.q.RemoveResourceWebhook(ctx, db.RemoveResourceWebhookParams{
		ResourceID: resourceID,
		WebhookID:  webhookID,
	})
}

func (s *WebhookService) GetResourceWebhooks(ctx context.Context, resourceID uuid.UUID) ([]db.GetResourceWebhooksRow, error) {
	return s.q.GetResourceWebhooks(ctx, resourceID)
}

func (s *WebhookService) GetWebhookLogs(ctx context.Context, webhookID uuid.UUID, limit, offset int32) ([]db.ClaimctlWebhookLog, error) {
	return s.q.GetWebhookLogs(ctx, db.GetWebhookLogsParams{
		WebhookID: webhookID,
		Limit:     limit,
		Offset:    offset,
	})
}

type WebhookPayload struct {
	ResourceID  uuid.UUID         `json:"resource_id"`
	Event       string            `json:"event"`
	Data        interface{}       `json:"data"`
	Secrets     map[string]string `json:"-"`
	Reservation interface{}       `json:"reservation,omitempty"`
}

func (s *WebhookService) TriggerWebhooks(ctx context.Context, resourceID uuid.UUID, event string, data interface{}) error {
	webhooks, err := s.q.GetWebhooksForEvent(ctx, db.GetWebhooksForEventParams{
		ResourceID: resourceID,
		Column2:    event,
	})
	if err != nil {
		return err
	}

	if len(webhooks) == 0 {
		return nil
	}

	secretsList, err := s.secretService.ListSecrets(ctx)
	if err != nil {
		slog.Info("Failed to list secrets: ", "error", err)
	}
	secretsMap := make(map[string]string)
	for _, secret := range secretsList {
		secretsMap[secret.Key] = secret.Value
	}

	payloadInfo := WebhookPayload{
		ResourceID: resourceID,
		Event:      event,
		Data:       data,
		Secrets:    secretsMap,
	}

	if res, ok := data.(db.ClaimctlReservation); ok {
		payloadInfo.Reservation = res
	}

	for _, hook := range webhooks {
		go s.executeWebhook(context.Background(), hook, payloadInfo)
	}

	return nil
}

func (s *WebhookService) executeWebhook(ctx context.Context, hook db.ClaimctlWebhook, payload WebhookPayload) {
	headers := make(map[string]string)
	if len(hook.Headers) > 0 {
		_ = json.Unmarshal(hook.Headers, &headers)
	}
	for k, v := range headers {
		headers[k] = s.resolveSecrets(v, payload.Secrets)
	}

	var body []byte
	var err error
	if hook.Template.Valid && hook.Template.String != "" {
		funcMap := template.FuncMap{
			"urlquery": url.QueryEscape,
			"secret": func(key string) string {
				return payload.Secrets[key]
			},
		}

		tmpl, err := template.New("webhook").Funcs(funcMap).Parse(hook.Template.String)
		if err != nil {
			s.logExecution(ctx, hook.ID, payload.Event, 0, "Template Error: "+err.Error(), "", 0)
			slog.Info("Error parsing template for webhook ", hook.Name, err)
			return
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, payload); err != nil {
			s.logExecution(ctx, hook.ID, payload.Event, 0, "Template Eval Error: "+err.Error(), "", 0)
			slog.Info("Error executing template for webhook ", hook.Name, err)
			return
		}
		body = buf.Bytes()
	} else {
		body, err = json.Marshal(payload)
		if err != nil {
			s.logExecution(ctx, hook.ID, payload.Event, 0, "JSON Marshal Error: "+err.Error(), "", 0)
			slog.Info("Error marshaling payload for webhook ", hook.Name, err)
			return
		}
	}

	maxRetries := 3
	var resp *http.Response
	var reqErr error
	var duration time.Duration

	for attempt := 0; attempt < maxRetries; attempt++ {
		start := time.Now()

		// Resolve secrets in URL
		resolvedUrl := s.resolveSecrets(hook.Url, payload.Secrets)

		req, err := http.NewRequest(hook.Method, resolvedUrl, bytes.NewBuffer(body))
		if err != nil {
			s.logExecution(ctx, hook.ID, payload.Event, 0, "Request Creation Error: "+err.Error(), "", 0)
			slog.Info("Error creating request for webhook ", hook.Name, err)
			return
		}

		for k, v := range headers {
			req.Header.Set(k, v)
		}
		if req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/json")
		}

		if hook.SigningSecret != "" {
			mac := hmac.New(sha256.New, []byte(hook.SigningSecret))
			mac.Write(body)
			signature := hex.EncodeToString(mac.Sum(nil))
			req.Header.Set("X-claimctl-Signature", "sha256="+signature)
		}

		resp, reqErr = s.httpClient.Do(req)
		duration = time.Since(start)

		if reqErr == nil && resp.StatusCode < 400 {
			break
		}

		if attempt < maxRetries-1 {
			waitTime := time.Duration(1<<attempt) * time.Second
			slog.Info("Webhook attempt failed", "attempt", attempt+1, "for webhook", hook.Name, "Retrying in %v...\n", waitTime)
			time.Sleep(waitTime)
		}
	}

	statusCode := 0
	respBody := ""

	if reqErr != nil {
		respBody = "Network Error: " + reqErr.Error()
	} else if resp != nil {
		statusCode = resp.StatusCode
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		respBody = buf.String()
		if len(respBody) > 1000 {
			respBody = respBody[:1000] + "... (truncated)"
		}
		resp.Body.Close()
	}

	s.logExecution(ctx, hook.ID, payload.Event, int32(statusCode), string(body), respBody, int32(duration.Milliseconds()))
}

func (s *WebhookService) logExecution(ctx context.Context, webhookID uuid.UUID, event string, statusCode int32, reqBody, respBody string, duration int32) {
	_, err := s.q.CreateWebhookLog(ctx, db.CreateWebhookLogParams{
		WebhookID:    webhookID,
		Event:        event,
		StatusCode:   statusCode,
		RequestBody:  reqBody,
		ResponseBody: respBody,
		DurationMs:   duration,
	})
	if err != nil {
		slog.Info("Failed to log webhook execution", "error", err)
	}
}

func (s *WebhookService) resolveSecrets(text string, secrets map[string]string) string {
	for k, v := range secrets {
		placeholder := fmt.Sprintf("{{Secret.%s}}", k)
		text = strings.ReplaceAll(text, placeholder, v)
	}
	return text
}
