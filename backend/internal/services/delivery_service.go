package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/thetaqitahmid/claimctl/internal/db"
)

func (s *WebhookService) StartDelivery(ctx context.Context) {
	s.deliveryOnce.Do(func() {
		for i := 0; i < 4; i++ {
			s.deliveryWG.Add(1)
			go func() {
				defer s.deliveryWG.Done()
				ticker := time.NewTicker(time.Second)
				defer ticker.Stop()
				for {
					if ctx.Err() != nil {
						return
					}
					worked, err := s.DeliverOne(ctx)
					if err != nil {
						slog.Error("Webhook delivery failed", "error", err)
					}
					if worked && err == nil {
						continue
					}
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
					}
				}
			}()
		}
	})
}

func (s *WebhookService) WaitDelivery() { s.deliveryWG.Wait() }

func (s *WebhookService) DeliverOne(ctx context.Context) (bool, error) {
	claimCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	job, err := s.q.ClaimWebhookDelivery(claimCtx, pgtype.UUID{Bytes: uuid.New(), Valid: true})
	cancel()
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if job.State == "dead" {
		return true, nil
	}
	sendCtx, stop := context.WithTimeout(ctx, 10*time.Second)
	status, retry, category := s.sendDelivery(sendCtx, job)
	stop()
	state := "delivered"
	if status < 200 || status >= 300 {
		state = "dead"
		if retry && job.Attempts < 8 {
			state = "pending"
		}
	}
	delay := int32(1 << min(job.Attempts, 8))
	ackCtx, done := context.WithTimeout(ctx, 5*time.Second)
	defer done()
	_, err = s.q.FinishWebhookDelivery(ackCtx, db.FinishWebhookDeliveryParams{
		ID: job.ID, Fence: job.Fence, State: state, LastStatus: status, LastError: category, DelaySeconds: delay,
	})
	return true, err
}

func decodeDelivery(job db.ClaimctlWebhookOutbox) (db.ClaimctlWebhook, WebhookPayload, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(job.Hook, &raw); err != nil {
		return db.ClaimctlWebhook{}, WebhookPayload{}, err
	}
	var hook db.ClaimctlWebhook
	for key, dest := range map[string]interface{}{
		"id": &hook.ID, "name": &hook.Name, "url": &hook.Url, "method": &hook.Method,
		"template": &hook.Template, "signing_secret": &hook.SigningSecret,
	} {
		if err := json.Unmarshal(raw[key], dest); err != nil {
			return hook, WebhookPayload{}, err
		}
	}
	hook.Headers = raw["headers"]
	var row map[string]json.RawMessage
	if err := json.Unmarshal(job.Reservation, &row); err != nil {
		return hook, WebhookPayload{}, err
	}
	camel := make(map[string]json.RawMessage)
	for key, value := range row {
		parts := strings.Split(key, "_")
		for i := 1; i < len(parts); i++ {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
		camel[strings.Join(parts, "")] = value
	}
	for _, key := range []string{"duration", "leaseDuration"} {
		if string(camel[key]) == "null" {
			camel[key], _ = json.Marshal(pgtype.Interval{})
			continue
		}
		var text string
		if err := json.Unmarshal(camel[key], &text); err != nil {
			return hook, WebhookPayload{}, err
		}
		var interval pgtype.Interval
		if err := pgtype.NewMap().Scan(pgtype.IntervalOID, pgtype.TextFormatCode, []byte(text), &interval); err != nil {
			return hook, WebhookPayload{}, err
		}
		camel[key], _ = json.Marshal(interval)
	}
	encoded, err := json.Marshal(camel)
	if err != nil {
		return hook, WebhookPayload{}, err
	}
	var reservation db.ClaimctlReservation
	if err := json.Unmarshal(encoded, &reservation); err != nil {
		return hook, WebhookPayload{}, err
	}
	return hook, WebhookPayload{ResourceID: job.ResourceID, Event: job.Event, Data: reservation, Reservation: reservation}, nil
}

func (s *WebhookService) sendDelivery(ctx context.Context, job db.ClaimctlWebhookOutbox) (int32, bool, string) {
	if job.Attempts > 8 {
		return 0, false, "attempt_limit"
	}
	hook, payload, err := decodeDelivery(job)
	if err != nil {
		return 0, false, "snapshot_invalid"
	}
	s.decryptWebhookSecret(&hook)
	if strings.HasPrefix(hook.SigningSecret, "ENC:") {
		return 0, true, "signing_secret_unavailable"
	}
	payload.Secrets = make(map[string]string)
	if s.secretService != nil {
		secrets, err := s.secretService.ListSecrets(ctx)
		if err != nil {
			return 0, true, "template_secrets_unavailable"
		}
		for _, secret := range secrets {
			payload.Secrets[secret.Key] = secret.Value
		}
	}
	body, err := json.Marshal(payload)
	if hook.Template.Valid && hook.Template.String != "" {
		body, err = s.executeTemplate(hook, payload)
	}
	if err != nil {
		return 0, false, "payload_invalid"
	}
	req, err := http.NewRequestWithContext(ctx, hook.Method, s.resolveSecrets(hook.Url, payload.Secrets), bytes.NewReader(body))
	if err != nil {
		return 0, false, "request_invalid"
	}
	var headers map[string]string
	if json.Unmarshal(hook.Headers, &headers) != nil {
		return 0, false, "headers_invalid"
	}
	for k, v := range headers {
		req.Header.Set(k, s.resolveSecrets(v, payload.Secrets))
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-claimctl-Delivery-ID", job.ID.String())
	if hook.SigningSecret != "" {
		mac := hmac.New(sha256.New, []byte(hook.SigningSecret))
		mac.Write(body)
		req.Header.Set("X-claimctl-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, true, "network_error"
	}
	_ = readResponseBody(resp)
	category := ""
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		category = "http_status"
	}
	return int32(resp.StatusCode), resp.StatusCode == 429 || resp.StatusCode >= 500, category
}

func (s *WebhookService) ListDeliveries(ctx context.Context, webhookID uuid.UUID) ([]db.ListWebhookDeliveriesRow, error) {
	auth, ok := AuthorizationFromContext(ctx)
	if ok && (auth.User.Role != "admin" || len(auth.SpaceIDs) > 0) {
		return nil, ErrAccessDenied
	}
	if ok && auth.TokenID != uuid.Nil {
		allowed := false
		for _, scope := range auth.Scopes {
			if scope == "webhooks:write" {
				allowed = true
			}
		}
		if !allowed {
			return nil, ErrAccessDenied
		}
	}
	rows, err := s.q.ListWebhookDeliveries(ctx, webhookID)
	if err != nil {
		return nil, fmt.Errorf("list deliveries: %w", err)
	}
	return rows, nil
}
