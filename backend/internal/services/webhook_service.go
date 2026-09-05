package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"text/template"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/utils"
)

var allowedMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true,
}

const maxTemplateOutput = 1 << 20 // 1 MB

// maxResponseLogBytes caps how much of a webhook response body is read and
// stored in webhook_logs.
const maxResponseLogBytes = 1000

// webhookRetryBaseDelay is the base backoff between webhook retry attempts;
// overridable in tests.
var webhookRetryBaseDelay = time.Second

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
		httpClient:    newSafeHTTPClient(10 * time.Second),
		encryptionKey: encryptionKey,
	}
}

func (s *WebhookService) CreateWebhook(ctx context.Context, name, rawURL, method string, headers map[string]string, tmpl, description string) (db.ClaimctlWebhook, error) {
	if err := ValidateWebhookURL(ctx, rawURL); err != nil {
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
	if err := ValidateWebhookURL(ctx, rawURL); err != nil {
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
	statusCode := 0
	respBody := ""
	var totalDuration time.Duration

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

		resp, err := s.httpClient.Do(req)
		totalDuration += time.Since(start)

		if err != nil {
			statusCode = 0
			respBody = "Network Error: " + err.Error()
		} else {
			statusCode = resp.StatusCode
			respBody = readResponseBody(resp)
		}

		// Only network errors, 429, and 5xx responses are worth retrying.
		if err == nil && statusCode != http.StatusTooManyRequests && statusCode < 500 {
			break
		}

		if attempt < maxRetries-1 {
			waitTime := time.Duration(1<<attempt) * webhookRetryBaseDelay
			slog.Info("Webhook attempt failed", "attempt", attempt+1, "webhook", hook.Name, "retry_in", waitTime.String())
			time.Sleep(waitTime)
		}
	}

	s.logExecution(ctx, hook.ID, payload.Event, int32(statusCode), string(body), respBody, int32(totalDuration.Milliseconds()))
}

// readResponseBody reads at most maxResponseLogBytes+1 bytes and closes the
// body, so responses are always drained/closed even across retries and a
// misbehaving endpoint cannot exhaust memory.
func readResponseBody(resp *http.Response) string {
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(io.LimitReader(resp.Body, maxResponseLogBytes+1))
	body := buf.String()
	if len(body) > maxResponseLogBytes {
		cut := body[:maxResponseLogBytes]
		// Do not split a multi-byte rune in half.
		for len(cut) > 0 && !utf8.ValidString(cut) {
			cut = cut[:len(cut)-1]
		}
		body = cut + "... (truncated)"
	}
	return body
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

// resolveHostIPs resolves a hostname to its IP addresses. It is a package
// variable so tests can stub DNS lookups and stay hermetic.
var resolveHostIPs = func(ctx context.Context, host string) ([]net.IP, error) {
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	ips := make([]net.IP, len(addrs))
	for i, addr := range addrs {
		ips[i] = addr.IP
	}
	return ips, nil
}

// isBlockedIP reports whether an IP is a destination that outbound webhook
// or notification traffic must never reach: loopback, private, link-local,
// multicast, or unspecified addresses.
func isBlockedIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified()
}

// safeDialContext is an http.Transport DialContext that only connects to
// public addresses. Hostnames are resolved here rather than by the transport,
// and a validated IP is dialed directly, so a hostname that resolves to an
// internal address cannot be reached even if DNS changes between validation
// and connection. Redirects are covered too, because every connection the
// client makes goes through this dialer.
func safeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	var dialAddrs []string
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return nil, fmt.Errorf("connection to %s is not allowed", ip)
		}
		dialAddrs = []string{net.JoinHostPort(ip.String(), port)}
	} else {
		ips, err := resolveHostIPs(ctx, host)
		if err != nil {
			return nil, err
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("no addresses resolved for host %q", host)
		}
		for _, ip := range ips {
			if isBlockedIP(ip) {
				return nil, fmt.Errorf("host %q resolves to blocked address %s", host, ip)
			}
		}
		for _, ip := range ips {
			dialAddrs = append(dialAddrs, net.JoinHostPort(ip.String(), port))
		}
	}

	// Try each resolved address in turn so a single dead node behind
	// round-robin DNS does not fail the delivery.
	var lastErr error
	for _, dialAddr := range dialAddrs {
		conn, err := outboundDial(ctx, network, dialAddr)
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// outboundDial dials a validated address; package variable so tests can stub
// the network.
var outboundDial = func(ctx context.Context, network, addr string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	return dialer.DialContext(ctx, network, addr)
}

// newSafeHTTPClient returns a client whose connections are restricted to
// public destinations, on top of the given overall request timeout.
//
// Proxies are disabled on purpose: with ProxyFromEnvironment, a request would
// connect to the proxy itself, and this dialer would rightly refuse
// loopback/private proxy addresses -- breaking all traffic in environments
// that set HTTP_PROXY. Going direct also keeps egress policy enforceable
// here instead of delegating to an unvalidated proxy.
func newSafeHTTPClient(timeout time.Duration) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = safeDialContext
	return &http.Client{Timeout: timeout, Transport: transport}
}

// ValidateWebhookURL checks that rawURL is a well-formed http(s) URL whose
// host resolves only to public addresses. DNS resolution respects ctx and is
// capped at a short deadline so callers in request handlers cannot hang on a
// slow resolver.
func ValidateWebhookURL(ctx context.Context, rawURL string) error {
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
	if host == "localhost" {
		return fmt.Errorf("url must not point to loopback address")
	}
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return fmt.Errorf("url must not point to a private or reserved address")
		}
		return nil
	}

	// Hostname: resolve it and reject if any resulting address is internal,
	// so names such as metadata.google.internal cannot be used to reach
	// internal services.
	resolveCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	ips, err := resolveHostIPs(resolveCtx, host)
	if err != nil {
		return fmt.Errorf("could not resolve url host %q: %w", host, err)
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return fmt.Errorf("url host %q resolves to a private or reserved address", host)
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
