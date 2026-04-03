# claimctl Database Documentation

## Overview

The claimctl database is a PostgreSQL database designed to manage resource
reservations in a multi-user environment. It supports queue-based reservation
management, comprehensive audit logging, role-based access control, and advanced
integration features like webhooks and health monitoring.

## Core Tables

### resources

Stores information about reservable resources.

| Column     | Type                         | Description                                     |
| ---------- | ---------------------------- | ----------------------------------------------- |
| id         | SERIAL PRIMARY KEY           | Unique identifier                               |
| name       | VARCHAR(255) NOT NULL UNIQUE | Resource name                                   |
| type       | VARCHAR(100) NOT NULL        | Resource category (server, license, etc.)       |
| labels     | JSONB                        | Flexible array of tags                          |
| properties | JSONB                        | Custom properties map                           |
| space_id   | INTEGER NOT NULL             | Reference to the Space this resource belongs to |
| created_at | BIGINT                       | Creation timestamp (epoch)                      |
| updated_at | BIGINT                       | Last modification timestamp (epoch)             |

**Index:** `idx_resources_name` **Foreign Key:** `space_id` → `spaces.id`

### users

Manages user accounts.

| Column            | Type                         | Description                            |
| ----------------- | ---------------------------- | -------------------------------------- |
| id                | SERIAL PRIMARY KEY           | Unique identifier                      |
| email             | VARCHAR(255) NOT NULL UNIQUE | Login email                            |
| name              | VARCHAR(255) NOT NULL        | Display name                           |
| password          | VARCHAR(255) NOT NULL        | Hashed password                        |
| admin             | BOOLEAN DEFAULT false        | Global admin flag                      |
| role              | VARCHAR(50) DEFAULT 'user'   | Role classification                    |
| status            | VARCHAR(50) DEFAULT 'active' | Account status                         |
| last_login        | TIMESTAMP WITH TIME ZONE     | Last login time                        |
| slack_destination | TEXT                         | Slack ID/Channel for notifications     |
| teams_webhook_url | TEXT                         | MS Teams Webhook URL for notifications |
| created_at        | TIMESTAMP WITH TIME ZONE     | Creation timestamp                     |
| updated_at        | TIMESTAMP WITH TIME ZONE     | Update timestamp                       |

**Index:** `idx_users_email`

### reservations

Manages resource usage and queues.

| Column             | Type                          | Description                                   |
| ------------------ | ----------------------------- | --------------------------------------------- |
| id                 | SERIAL PRIMARY KEY            | Unique identifier                             |
| resource_id        | INTEGER NOT NULL              | Reserved resource                             |
| user_id            | INTEGER NOT NULL              | User holding reservation                      |
| status             | VARCHAR(20) DEFAULT 'pending' | `pending`, `active`, `completed`, `cancelled` |
| queue_position     | INTEGER                       | Position in queue (if pending)                |
| start_time         | BIGINT                        | Actual start time (epoch)                     |
| end_time           | BIGINT                        | Actual end time (epoch)                       |
| scheduled_end_time | TIMESTAMP WITH TIME ZONE      | Expected end time                             |
| duration           | INTERVAL                      | Reserved duration                             |
| created_at         | BIGINT                        | Creation timestamp (epoch)                    |
| updated_at         | BIGINT                        | Update timestamp (epoch)                      |

**Foreign Keys:** `resource_id` → `resources.id`, `user_id` → `users.id`
**Indexes:** `idx_reservations_resource_status`, `idx_reservations_user_active`,
`idx_resource_active_reservation`

## Organization & Access Control

### spaces

Logical grouping of resources (e.g., Projects, Departments).

| Column      | Type               | Description        |
| ----------- | ------------------ | ------------------ |
| id          | SERIAL PRIMARY KEY | Unique identifier  |
| name        | VARCHAR NOT NULL   | Space name         |
| description | TEXT               | Description        |
| created_at  | BIGINT             | Creation timestamp |
| updated_at  | BIGINT             | Update timestamp   |

### groups

User groups for permission management.

| Column      | Type                     | Description        |
| ----------- | ------------------------ | ------------------ |
| id          | SERIAL PRIMARY KEY       | Unique identifier  |
| name        | VARCHAR NOT NULL         | Group name         |
| description | TEXT                     | Description        |
| created_at  | TIMESTAMP WITH TIME ZONE | Creation timestamp |
| updated_at  | TIMESTAMP WITH TIME ZONE | Update timestamp   |

### group_members

Many-to-Many relationship between Users and Groups.

| Column    | Type                     | Description        |
| --------- | ------------------------ | ------------------ |
| group_id  | INTEGER NOT NULL         | Reference to Group |
| user_id   | INTEGER NOT NULL         | Reference to User  |
| joined_at | TIMESTAMP WITH TIME ZONE | Join timestamp     |

**Primary Key:** `(group_id, user_id)`

### space_permissions

Defines access control for Spaces.

| Column     | Type                     | Description                   |
| ---------- | ------------------------ | ----------------------------- |
| id         | SERIAL PRIMARY KEY       | Unique identifier             |
| space_id   | INTEGER                  | Reference to Space            |
| group_id   | INTEGER                  | Reference to Group (nullable) |
| user_id    | INTEGER                  | Reference to User (nullable)  |
| created_at | TIMESTAMP WITH TIME ZONE | Creation timestamp            |

## Integration & Automation

### api_tokens

Personal Access Tokens for API authentication.

| Column       | Type                     | Description            |
| ------------ | ------------------------ | ---------------------- |
| id           | UUID PRIMARY KEY         | Unique Token ID        |
| user_id      | INTEGER NOT NULL         | Owner                  |
| name         | VARCHAR NOT NULL         | Token description/name |
| token_hash   | VARCHAR NOT NULL         | Hashed token secret    |
| created_at   | TIMESTAMP WITH TIME ZONE | Creation timestamp     |
| expires_at   | TIMESTAMP WITH TIME ZONE | Expiration timestamp   |
| last_used_at | TIMESTAMP WITH TIME ZONE | Last usage timestamp   |

### webhooks

External webhooks configuration.

| Column         | Type               | Description                       |
| -------------- | ------------------ | --------------------------------- |
| id             | SERIAL PRIMARY KEY | Unique identifier                 |
| name           | VARCHAR NOT NULL   | Webhook name                      |
| url            | VARCHAR NOT NULL   | Destination URL                   |
| method         | VARCHAR NOT NULL   | HTTP Method                       |
| headers        | BYTEA              | Encrypted headers                 |
| template       | TEXT               | Payload template                  |
| signing_secret | VARCHAR            | Secret for signature verification |
| created_at     | BIGINT             | Creation timestamp                |

### resource_webhooks

Links Webhooks to specific Resources and Events.

| Column      | Type             | Description                                   |
| ----------- | ---------------- | --------------------------------------------- |
| resource_id | INTEGER NOT NULL | Reference to Resource                         |
| webhook_id  | INTEGER NOT NULL | Reference to Webhook                          |
| events      | TEXT[]           | Array of events (e.g., `reservation.created`) |

### secrets

Secure storage for sensitive data used in integrations.

| Column      | Type                    | Description            |
| ----------- | ----------------------- | ---------------------- |
| id          | SERIAL PRIMARY KEY      | Unique identifier      |
| key         | VARCHAR NOT NULL UNIQUE | Secret key name        |
| value       | VARCHAR NOT NULL        | Encrypted secret value |
| description | TEXT                    | Description            |

### webhook_logs

Audit logs for webhook executions.

| Column        | Type               | Description               |
| ------------- | ------------------ | ------------------------- |
| id            | SERIAL PRIMARY KEY | Log identifier            |
| webhook_id    | INTEGER NOT NULL   | Reference to Webhook      |
| event         | VARCHAR NOT NULL   | Triggering event          |
| status_code   | INTEGER            | HTTP status code received |
| request_body  | TEXT               | Sent payload              |
| response_body | TEXT               | Received response         |
| duration_ms   | INTEGER            | Execution time in ms      |
| created_at    | TIMESTAMP          | Execution timestamp       |

## Monitoring & Health

### resource_health_configs

Configuration for resource health checks.

| Column           | Type                | Description             |
| ---------------- | ------------------- | ----------------------- |
| resource_id      | INTEGER PRIMARY KEY | Reference to Resource   |
| enabled          | BOOLEAN             | Enable/Disable check    |
| check_type       | VARCHAR NOT NULL    | Type (ping, http, port) |
| target           | VARCHAR NOT NULL    | Target IP/URL           |
| interval_seconds | INTEGER             | Check frequency         |
| timeout_seconds  | INTEGER             | Timeout threshold       |

### resource_health_statuses

Latest health status for resources.

| Column           | Type               | Description                        |
| ---------------- | ------------------ | ---------------------------------- |
| id               | SERIAL PRIMARY KEY | Unique identifier                  |
| resource_id      | INTEGER NOT NULL   | Reference to Resource              |
| status           | VARCHAR NOT NULL   | `healthy`, `unhealthy`, `degraded` |
| response_time_ms | INTEGER            | Latency in ms                      |
| error_message    | TEXT               | Error details if any               |
| checked_at       | BIGINT             | Check timestamp                    |

## Configuration

### app_settings

Global application settings.

| Column    | Type                | Description                 |
| --------- | ------------------- | --------------------------- |
| key       | VARCHAR PRIMARY KEY | Setting key                 |
| value     | VARCHAR NOT NULL    | Setting value               |
| category  | VARCHAR             | Grouping category           |
| is_secret | BOOLEAN             | If true, value is encrypted |

### user_notification_preferences

User-specific notification settings.

| Column     | Type             | Description                       |
| ---------- | ---------------- | --------------------------------- |
| user_id    | INTEGER NOT NULL | Reference to User                 |
| event_type | VARCHAR NOT NULL | Event (e.g., `reservation_start`) |
| channel    | VARCHAR NOT NULL | Channel (email, slack, teams)     |
| enabled    | BOOLEAN          | Preference state                  |

**Primary Key:** `(user_id, event_type, channel)`

## Relationships Diagram

- **Space** 1 -- \* **Resource**
- **User** 1 -- \* **Reservation**
- **Resource** 1 -- \* **Reservation**
- **Resource** 1 -- 1 **HealthConfig**
- **Group** _ -- _ **User** (via GroupMember)
- **Space** 1 -- \* **Permission**
- **Webhook** _ -- _ **Resource** (via ResourceWebhook)

## Security & Data Integrity

- **Encryption**: Secrets, API Token hashes, and App Setting secrets are
  encrypted at rest.
- **Audit Logs**: `reservation_history` (legacy/core) and `webhook_logs` track
  system activity.
- **Foreign Keys**: Enforced on all major relationships with appropriate CASCADE
  rules.
- **Timestamps**: Uses Epoch (BIGINT) for core logic and TIMESTAMPTZ for user-
  facing audit fields.
