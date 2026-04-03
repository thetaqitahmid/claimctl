package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"

	"github.com/slack-go/slack"
)

type NotificationPayload struct {
	Subject string
	Message string
}

type NotificationDispatcher interface {
	Dispatch(ctx context.Context, recipient string, payload NotificationPayload) error
	Type() string
}

// EmailDispatcher sends emails using SMTP
type EmailDispatcher struct {
	settings *SettingsService
}

func NewEmailDispatcher(settings *SettingsService) *EmailDispatcher {
	return &EmailDispatcher{settings: settings}
}

func (d *EmailDispatcher) Type() string {
	return "email"
}

func (d *EmailDispatcher) Dispatch(ctx context.Context, recipient string, payload NotificationPayload) error {
	host, port, user, pass, from := d.settings.GetSMTPConfig(ctx)

	if host == "" || port == "" || user == "" || from == "" {
		return fmt.Errorf("SMTP configuration missing")
	}

	// Simple email construction
	auth := smtp.PlainAuth("", user, pass, host)
	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"Subject: %s\r\n"+
		"\r\n"+
		"%s\r\n", recipient, payload.Subject, payload.Message))

	addr := fmt.Sprintf("%s:%s", host, port)
	err := smtp.SendMail(addr, auth, from, []string{recipient}, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	return nil
}

// SlackDispatcher sends messages to Slack via Webhook or Bot Token
type SlackDispatcher struct {
	settings *SettingsService
}

func NewSlackDispatcher(settings *SettingsService) *SlackDispatcher {
	return &SlackDispatcher{settings: settings}
}

func (d *SlackDispatcher) Type() string {
	return "slack"
}

// Dispatch sends a Slack message.
// recipient: For 'bot' mode, this is the Channel ID. For 'webhook' mode, this matches the configured webhook (legacy).
// With dynamic settings, we check if the recipient looks like a URL (webhook) or ID.
// For now, let's assume if it starts with 'http', it's a webhook URL overridden by the user/caller.
// Otherwise, we use the bot token from settings.
func (d *SlackDispatcher) Dispatch(ctx context.Context, recipient string, payload NotificationPayload) error {
	// 1. Check if recipient is a Webhook URL
	if len(recipient) > 4 && recipient[:4] == "http" {
		return d.dispatchWebhook(ctx, recipient, payload)
	}

	// 2. Use Bot Token
	token := d.settings.GetSlackBotToken(ctx)
	if token == "" {
		return fmt.Errorf("Slack bot token not configured")
	}

	api := slack.New(token)
	_, _, err := api.PostMessageContext(ctx, recipient, slack.MsgOptionText(payload.Message, false))
	if err != nil {
		return fmt.Errorf("failed to send slack message: %w", err)
	}
	return nil
}

func (d *SlackDispatcher) dispatchWebhook(ctx context.Context, url string, payload NotificationPayload) error {
	msg := map[string]string{"text": fmt.Sprintf("*%s*\n%s", payload.Subject, payload.Message)}
	body, _ := json.Marshal(msg)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned status: %v", resp.Status)
	}
	return nil
}

// TeamsDispatcher sends messages to Microsoft Teams via Webhook
type TeamsDispatcher struct{}

func NewTeamsDispatcher() *TeamsDispatcher {
	return &TeamsDispatcher{}
}

func (d *TeamsDispatcher) Type() string {
	return "teams"
}

func (d *TeamsDispatcher) Dispatch(ctx context.Context, recipient string, payload NotificationPayload) error {
	// Recipient acts as the Webhook URL here
	webhookURL := recipient
	if webhookURL == "" {
		return fmt.Errorf("Teams webhook URL missing")
	}

	card := map[string]interface{}{
		"type": "message",
		"attachments": []map[string]interface{}{
			{
				"contentType": "application/vnd.microsoft.card.adaptive",
				"content": map[string]interface{}{
					"type":    "AdaptiveCard",
					"version": "1.2",
					"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
					"body": []map[string]interface{}{
						{
							"type":   "TextBlock",
							"text":   payload.Subject,
							"weight": "Bolder",
							"size":   "Medium",
						},
						{
							"type": "TextBlock",
							"text": payload.Message,
							"wrap": true,
						},
					},
				},
			},
		},
	}

	body, err := json.Marshal(card)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Teams API returned status: %s", resp.Status)
	}

	return nil
}
