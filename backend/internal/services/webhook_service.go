package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
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
	"github.com/thetaqitahmid/claimctl/internal/utils"
)

var allowedMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true,
}

const maxTemplateOutput = 1 << 20 // 1 MB

type WebhookService struct {
	q             db.Querier
	secretService *SecretService
	httpClient    *http.Client
	encryptionKey string
}

func NewWebhookService(q db.Querier, secretService *SecretService, encryptionKey string) *WebhookService {
	return &WebhookService{
		q:             q,
		secretService: secretService,
		httpClient:    &http.Client{Timeout: 10 * time.Second},
		encryptionKey: encryptionKey,
	}
}

func (s *WebhookService) CreateWebhook(ctx context.Context, name, rawURL, method string, headers map[string]string, tmpl, description string) (db.ClaimctlWebhook, error) {
	if err := validateWebhookURL(rawURL); err != nil {
		return db.ClaimctlWebhook{}, err
	}
	if !allowedMethods[method] {
		return db.ClaimctlWebhook{}, fmt.Errorf("invalid HTTP method: %s", method)
	}
	if err := validateTemplate(tmpl); err != nil {
		return db.ClaimctlWebhook{}, fmt.Errorf("invalid template: %w", err)
	}

	headersJSON := []byte("{}")
	if headers != nil {
		var err error
		headersJSON, err = json.Marshal(headers)
		if err != nil {
			return db.ClaimctlWebhook{}, err
		}
	}

	signingSecretBytes := make([]byte, 32)
	if _, err := rand.Read(signingSecretBytes); err != nil {
		return db.ClaimctlWebhook{}, fmt.Errorf("failed to generate signing secret: %w", err)
	}
	signingSecret := hex.EncodeToString(signingSecretBytes)

	encryptedSecret, err := utils.Encrypt(signingSecret, s.encryptionKey)
	if err != nil {
		return db.ClaimctlWebhook{}, fmt.Errorf("failed to encrypt signing secret: %w", err)
	}

	webhook, err := s.q.CreateWebhook(ctx, db.CreateWebhookParams{
		Name:          name,
		Url:           rawURL,
		Method:        method,
		Headers:       headersJSON,
		Template:      pgtype.Text{String: tmpl, Valid: tmpl != ""},
		Description:   pgtype.Text{String: description, Valid: description != ""},
		SigningSecret: "ENC:" + encryptedSecret,
	})
	if err != nil {
		return webhook, err
	}
	webhook.SigningSecret = signingSecret
	return webhook, nil
}

func (s *WebhookService) GetWebhook(ctx context.Context, id uuid.UUID) (db.ClaimctlWebhook, error) {
	w, err := s.q.GetWebhook(ctx, id)
	if err != nil {
		return w, err
	}
	s.decryptWebhookSecret(&w)
	return w, nil
}

func (s *WebhookService) ListWebhooks(ctx context.Context) ([]db.ClaimctlWebhook, error) {
	webhooks, err := s.q.ListWebhooks(ctx)
	if err != nil {
		return nil, err
	}
	for i := range webhooks {
		webhooks[i].SigningSecret = ""
	}
	return webhooks, nil
}

func (s *WebhookService) UpdateWebhook(ctx context.Context, id uuid.UUID, name, rawURL, method string, headers map[string]string, tmpl, description string) (db.ClaimctlWebhook, error) {
	if err := validateWebhookURL(rawURL); err != nil {
		return db.ClaimctlWebhook{}, err
	}
	if !allowedMethods[method] {
		return db.ClaimctlWebhook{}, fmt.Errorf("invalid HTTP method: %s", method)
	}
	if err := validateTemplate(tmpl); err != nil {
		return db.ClaimctlWebhook{}, fmt.Errorf("invalid template: %w", err)
	}

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
		Url:         rawURL,
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

func (s *WebhookService) decryptWebhookSecret(w *db.ClaimctlWebhook) {
	if strings.HasPrefix(w.SigningSecret, "ENC:") {
		decrypted, err := utils.Decrypt(strings.TrimPrefix(w.SigningSecret, "ENC:"), s.encryptionKey)
		if err == nil {
			w.SigningSecret = decrypted
		} else {
			slog.Warn("failed to decrypt webhook signing secret", "webhook_id", w.ID, "error", err)
		}
	}
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

	for i := range webhooks {
		s.decryptWebhookSecret(&webhooks[i])
		go s.executeWebhook(context.Background(), webhooks[i], payloadInfo)
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
		body, err = s.executeTemplate(hook, payload)
		if err != nil {
			s.logExecution(ctx, hook.ID, payload.Event, 0, "Template Error: "+err.Error(), "", 0)
			return
		}
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

func (s *WebhookService) executeTemplate(hook db.ClaimctlWebhook, payload WebhookPayload) ([]byte, error) {
	funcMap := template.FuncMap{
		"urlquery": url.QueryEscape,
		"secret": func(key string) string {
			return payload.Secrets[key]
		},
	}

	tmpl, err := template.New("webhook").Funcs(funcMap).Parse(hook.Template.String)
	if err != nil {
		slog.Error("error parsing template for webhook", "name", hook.Name, "error", err)
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, payload); err != nil {
		slog.Error("error executing template for webhook", "name", hook.Name, "error", err)
		return nil, err
	}

	if buf.Len() > maxTemplateOutput {
		return nil, fmt.Errorf("template output exceeds %d byte limit", maxTemplateOutput)
	}
	return buf.Bytes(), nil
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

func validateWebhookURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("url is required")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("url must use http or https scheme")
	}
	if parsed.Host == "" {
		return fmt.Errorf("url must have a host")
	}
	host := parsed.Hostname()
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return fmt.Errorf("url must not point to loopback address")
	}
	ip := net.ParseIP(host)
	if ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return fmt.Errorf("url must not point to a private or reserved address")
		}
	}
	return nil
}

func validateTemplate(tmpl string) error {
	if tmpl == "" {
		return nil
	}
	funcMap := template.FuncMap{
		"urlquery": url.QueryEscape,
		"secret":   func(string) string { return "" },
	}
	_, err := template.New("validate").Funcs(funcMap).Parse(tmpl)
	return err
}
