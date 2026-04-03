# Webhooks & Automation

claimctl supports powerful automation via Webhooks, allowing it to
integrate with other systems (e.g., Slack, Discord, Home Assistant, custom
scripts) when reservation events occur.

## Webhook Concepts

A **Webhook** in claimctl describes _where_ to send data and _how_ to
format it.

- **URL**: The destination endpoint.
- **Method**: HTTP method (POST, PUT, GET, etc.).
- **Headers**: Custom keys (e.g., `Authorization`, `Content-Type`).
- **Template**: A JSON or text body that defines the payload. You can use
  placeholders like `{{User}}`, `{{Resource}}`, etc.

## Managing Webhooks

1. Go to **Admin Panel > Webhooks**.
2. Create a new Webhook definition.
3. Test the webhook to ensure connectivity.

## Resource Association

Webhooks are not global by default; they are attached to specific **Resources**.
For example, you might want a webhook to turn on a smart plug when a specific
"Microscope" is reserved, but not when a "Book" is reserved.

### How to Link

1. Go to a Resource's detail page.
2. Select **Add Webhook**.
3. Choose the Webhook definition and the **Events** that trigger it (e.g.,
   `ReservationActivated`, `ReservationCompleted`).

## Secrets Management

To securely store API keys or tokens needed for your webhooks (e.g., a Slack Bot
Token), use **Secrets**.

1. Go to **Admin Panel > Secrets**.
2. Create a Secret (Key-Value pair).
3. The value is stored securely and masked in the UI.
4. **Usage**: In your Webhook headers or template, reference the secret using
   `{{MySecretKey}}`. The system will inject the actual value at runtime.
