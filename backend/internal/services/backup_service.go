package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// BackupMetadata contains information about the backup itself.
type BackupMetadata struct {
	Version   string `json:"version"`
	Timestamp int64  `json:"timestamp"`
	Note      string `json:"note,omitempty"`
}

// BackupData is the top-level structure serialized to JSON.
type BackupData struct {
	Metadata             BackupMetadata           `json:"metadata"`
	Spaces               []map[string]interface{} `json:"spaces"`
	Users                []map[string]interface{} `json:"users"`
	Resources            []map[string]interface{} `json:"resources"`
	Reservations         []map[string]interface{} `json:"reservations"`
	ReservationHistory   []map[string]interface{} `json:"reservation_history"`
	Webhooks             []map[string]interface{} `json:"webhooks"`
	Secrets              []map[string]interface{} `json:"secrets"`
	ResourceWebhooks     []map[string]interface{} `json:"resource_webhooks"`
	WebhookLogs          []map[string]interface{} `json:"webhook_logs"`
	Settings             []map[string]interface{} `json:"settings"`
	Groups               []map[string]interface{} `json:"groups"`
	GroupMembers         []map[string]interface{} `json:"group_members"`
	SpacePermissions     []map[string]interface{} `json:"space_permissions"`
	APITokens            []map[string]interface{} `json:"api_tokens"`
	HealthConfigs        []map[string]interface{} `json:"health_configs"`
	HealthStatuses       []map[string]interface{} `json:"health_statuses"`
	Preferences          []map[string]interface{} `json:"preferences"`
	MaintenanceAuditLogs []map[string]interface{} `json:"maintenance_audit_logs"`
}

// DBPool defines the methods used by BackupService from pgxpool.Pool.
// This allows for mocking in tests.
type DBPool interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

// BackupService handles creating and restoring backups.
type BackupService struct {
	pool DBPool
}

// NewBackupService creates a new BackupService.
func NewBackupService(pool DBPool) *BackupService {
	return &BackupService{pool: pool}
}

// tableQuery maps a friendly name to the SQL used to export it.
var tableQueries = []struct {
	Name  string
	Query string
}{
	{"spaces", "SELECT * FROM claimctl.spaces ORDER BY id"},
	{"users", "SELECT * FROM claimctl.users ORDER BY id"},
	{"resources", "SELECT * FROM claimctl.resources ORDER BY id"},
	{"reservations", "SELECT * FROM claimctl.reservations ORDER BY id"},
	{"reservation_history", "SELECT * FROM claimctl.reservation_history ORDER BY id"},
	{"webhooks", "SELECT * FROM claimctl.webhooks ORDER BY id"},
	{"secrets", "SELECT * FROM claimctl.secrets ORDER BY id"},
	{"resource_webhooks", "SELECT * FROM claimctl.resource_webhooks"},
	{"webhook_logs", "SELECT * FROM claimctl.webhook_logs ORDER BY id"},
	{"settings", "SELECT * FROM app_settings ORDER BY key"},
	{"groups", "SELECT * FROM claimctl.groups ORDER BY id"},
	{"group_members", "SELECT * FROM claimctl.group_members"},
	{"space_permissions", "SELECT * FROM claimctl.space_permissions ORDER BY id"},
	{"api_tokens", "SELECT * FROM claimctl.api_tokens ORDER BY created_at"},
	{"health_configs", "SELECT * FROM claimctl.resource_health_configs"},
	{"health_statuses", "SELECT * FROM claimctl.resource_health_status ORDER BY id"},
	{"preferences", "SELECT * FROM claimctl.user_notification_preferences"},
	{"maintenance_audit_logs", "SELECT * FROM claimctl.maintenance_audit_log ORDER BY id"},
}

// queryToRows runs a SELECT and returns the result as a slice of maps.
func (s *BackupService) queryToRows(ctx context.Context, query string) ([]map[string]interface{}, error) {
	rows, err := s.pool.Query(ctx, "SELECT row_to_json(backup_row) FROM ("+query+") AS backup_row")
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var result []map[string]interface{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		var row map[string]interface{}
		if err := json.Unmarshal(raw, &row); err != nil {
			return nil, fmt.Errorf("invalid exported row: %w", err)
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	if result == nil {
		result = []map[string]interface{}{}
	}
	return result, nil
}

// CreateBackup exports all application data into a BackupData struct.
func (s *BackupService) CreateBackup(ctx context.Context) (*BackupData, error) {
	backup := &BackupData{
		Metadata: BackupMetadata{
			Version:   "1",
			Timestamp: time.Now().Unix(),
			Note:      "Secrets and settings marked as is_secret are stored in their encrypted form. The same APP_ENCRYPTION_KEY must be used on the target instance.",
		},
	}

	for _, tq := range tableQueries {
		rows, err := s.queryToRows(ctx, tq.Query)
		if err != nil {
			return nil, fmt.Errorf("failed to export %s: %w", tq.Name, err)
		}
		switch tq.Name {
		case "spaces":
			backup.Spaces = rows
		case "users":
			backup.Users = rows
		case "resources":
			backup.Resources = rows
		case "reservations":
			backup.Reservations = rows
		case "reservation_history":
			backup.ReservationHistory = rows
		case "webhooks":
			backup.Webhooks = rows
		case "secrets":
			backup.Secrets = rows
		case "resource_webhooks":
			backup.ResourceWebhooks = rows
		case "webhook_logs":
			backup.WebhookLogs = rows
		case "settings":
			backup.Settings = rows
		case "groups":
			backup.Groups = rows
		case "group_members":
			backup.GroupMembers = rows
		case "space_permissions":
			backup.SpacePermissions = rows
		case "api_tokens":
			backup.APITokens = rows
		case "health_configs":
			backup.HealthConfigs = rows
		case "health_statuses":
			backup.HealthStatuses = rows
		case "preferences":
			backup.Preferences = rows
		case "maintenance_audit_logs":
			backup.MaintenanceAuditLogs = rows
		}
	}

	return backup, nil
}

// truncation order: children first, then parents to respect FK constraints.
var truncateOrder = []string{
	"claimctl.realtime_events",
	"claimctl.shared_storage",
	"claimctl.rate_limits",
	"claimctl.webhook_outbox",
	"claimctl.worker_leases",
	"claimctl.webhook_logs",
	"claimctl.resource_webhooks",
	"claimctl.resource_health_status",
	"claimctl.resource_health_configs",
	"claimctl.reservation_history",
	"claimctl.reservations",
	"claimctl.maintenance_audit_log",
	"claimctl.user_notification_preferences",
	"claimctl.group_members",
	"claimctl.space_permissions",
	"claimctl.api_tokens",
	"claimctl.webhooks",
	"claimctl.secrets",
	"claimctl.resources",
	"claimctl.groups",
	"claimctl.spaces",
	"claimctl.users",
	"app_settings",
}

// TableSchemas defines the allowed columns for each table to prevent SQL injection via malicious JSON keys.
var TableSchemas = map[string][]string{
	"app_settings":                           {"key", "value", "category", "description", "is_secret", "updated_at"},
	"claimctl.spaces":                        {"id", "name", "description", "created_at", "updated_at"},
	"claimctl.users":                         {"id", "email", "name", "password", "role", "last_login", "status", "failed_login_attempts", "locked_until", "slack_destination", "teams_webhook_url", "notification_email", "auth_provider", "oidc_subject", "created_at", "updated_at"},
	"claimctl.groups":                        {"id", "name", "description", "created_at", "updated_at"},
	"claimctl.resources":                     {"id", "name", "type", "labels", "created_at", "updated_at", "space_id", "properties", "is_under_maintenance"},
	"claimctl.secrets":                       {"id", "key", "value", "description", "created_at", "updated_at"},
	"claimctl.webhooks":                      {"id", "name", "url", "method", "headers", "template", "description", "signing_secret", "created_at", "updated_at"},
	"claimctl.reservations":                  {"id", "resource_id", "user_id", "status", "start_time", "end_time", "created_at", "updated_at", "queue_position", "duration", "scheduled_end_time", "lease_duration", "lease_expires_at", "hard_end_time"},
	"claimctl.reservation_history":           {"id", "resource_id", "reservation_id", "action", "user_id", "timestamp", "details"},
	"claimctl.resource_webhooks":             {"resource_id", "webhook_id", "events"},
	"claimctl.webhook_logs":                  {"id", "webhook_id", "event", "status_code", "request_body", "response_body", "duration_ms", "created_at"},
	"claimctl.group_members":                 {"group_id", "user_id", "joined_at"},
	"claimctl.space_permissions":             {"id", "space_id", "group_id", "user_id", "created_at"},
	"claimctl.api_tokens":                    {"id", "user_id", "name", "token_hash", "last_used_at", "expires_at", "created_at", "scopes", "space_ids"},
	"claimctl.resource_health_configs":       {"resource_id", "enabled", "check_type", "target", "interval_seconds", "timeout_seconds", "retry_count", "created_at", "updated_at"},
	"claimctl.resource_health_status":        {"id", "resource_id", "status", "response_time_ms", "error_message", "checked_at", "created_at"},
	"claimctl.user_notification_preferences": {"user_id", "event_type", "channel", "enabled", "created_at", "updated_at"},
	"claimctl.maintenance_audit_log":         {"id", "resource_id", "previous_state", "new_state", "changed_by", "changed_at", "reason"},
}

// RestoreBackup replaces all application data with the supplied backup.
func (s *BackupService) RestoreBackup(ctx context.Context, data []byte) error {
	var backup BackupData
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&backup); err != nil {
		return fmt.Errorf("invalid backup JSON: %w", err)
	}
	if err := decoder.Decode(new(interface{})); err != io.EOF {
		return fmt.Errorf("invalid backup JSON trailing data")
	}

	if backup.Metadata.Version != "1" {
		return fmt.Errorf("unsupported backup version")
	}
	if err := validateBackupSecurity(&backup); err != nil {
		return err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, "SET LOCAL lock_timeout = '5s'"); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(725025)"); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "SET LOCAL claimctl.restoring = 'on'"); err != nil {
		return fmt.Errorf("failed to suppress restore delivery: %w", err)
	}

	// 1. Truncate all tables
	for _, table := range truncateOrder {
		if _, err := tx.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)); err != nil {
			return fmt.Errorf("failed to truncate %s: %w", table, err)
		}
	}
	slog.Info("Restore: all tables truncated")

	// 2. Insert data in dependency order (parents first)
	insertOrder := []struct {
		Table string
		Rows  []map[string]interface{}
	}{
		{"app_settings", backup.Settings},
		{"claimctl.spaces", backup.Spaces},
		{"claimctl.users", backup.Users},
		{"claimctl.groups", backup.Groups},
		{"claimctl.resources", backup.Resources},
		{"claimctl.secrets", backup.Secrets},
		{"claimctl.webhooks", backup.Webhooks},
		{"claimctl.reservations", backup.Reservations},
		{"claimctl.reservation_history", backup.ReservationHistory},
		{"claimctl.resource_webhooks", backup.ResourceWebhooks},
		{"claimctl.webhook_logs", backup.WebhookLogs},
		{"claimctl.group_members", backup.GroupMembers},
		{"claimctl.space_permissions", backup.SpacePermissions},
		{"claimctl.api_tokens", backup.APITokens},
		{"claimctl.resource_health_configs", backup.HealthConfigs},
		{"claimctl.resource_health_status", backup.HealthStatuses},
		{"claimctl.user_notification_preferences", backup.Preferences},
		{"claimctl.maintenance_audit_log", backup.MaintenanceAuditLogs},
	}

	for _, entry := range insertOrder {
		if len(entry.Rows) == 0 {
			continue
		}
		if err := insertRows(ctx, tx, entry.Table, entry.Rows); err != nil {
			return fmt.Errorf("failed to insert into %s: %w", entry.Table, err)
		}
		slog.Info("Restore: inserted rows", "table", entry.Table, "count", len(entry.Rows))
	}

	if _, err := tx.Exec(ctx, "INSERT INTO claimctl.realtime_events(event_type) VALUES ('resync')"); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit restore transaction: %w", err)
	}

	slog.Info("Restore: completed successfully")
	return nil
}

func validateBackupSecurity(backup *BackupData) error {
	for _, user := range backup.Users {
		if provider, exists := user["auth_provider"]; exists && provider != "local" && provider != "oidc" {
			return fmt.Errorf("invalid backup auth_provider")
		}
	}
	for _, token := range backup.APITokens {
		if raw, exists := token["scopes"]; exists {
			scopes, ok := raw.([]interface{})
			if !ok {
				return fmt.Errorf("invalid backup token scopes")
			}
			for _, rawScope := range scopes {
				scope, ok := rawScope.(string)
				if !ok {
					return fmt.Errorf("invalid backup token scope")
				}
				if _, known := TokenScopes[scope]; !known {
					return fmt.Errorf("unknown backup token scope")
				}
			}
		}
		if raw, exists := token["space_ids"]; exists {
			ids, ok := raw.([]interface{})
			if !ok {
				return fmt.Errorf("invalid backup token space_ids")
			}
			for _, rawID := range ids {
				text, ok := rawID.(string)
				if !ok {
					return fmt.Errorf("invalid backup token space ID")
				}
				id, err := uuid.Parse(text)
				if err != nil || id == uuid.Nil {
					return fmt.Errorf("invalid backup token space ID")
				}
			}
		}
	}
	return nil
}

// insertRows dynamically inserts a slice of maps into the given table using a column whitelist.
func insertRows(ctx context.Context, tx pgx.Tx, table string, rows []map[string]interface{}) error {
	if len(rows) == 0 {
		return nil
	}

	allowedColumns, ok := TableSchemas[table]
	if !ok {
		return fmt.Errorf("table %s is not in the allowed schema list", table)
	}

	for _, row := range rows {
		// Only include allowed columns that are present in the row
		var rowColumns []string
		var values []interface{}

		for _, col := range allowedColumns {
			if val, exists := row[col]; exists {
				rowColumns = append(rowColumns, col)
				values = append(values, val)
			}
		}

		for col := range row {
			allowed := false
			for _, known := range allowedColumns {
				if col == known {
					allowed = true
					break
				}
			}
			if !allowed {
				return fmt.Errorf("unknown column %s in %s", col, table)
			}
		}
		if len(rowColumns) == 0 {
			return fmt.Errorf("empty row in %s", table)
		}

		// Build INSERT statement for this row
		colList := ""
		placeholders := ""
		for i, col := range rowColumns {
			if i > 0 {
				colList += ", "
				placeholders += ", "
			}
			colList += col
			placeholders += fmt.Sprintf("$%d", i+1)
		}

		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, colList, placeholders)

		if _, err := tx.Exec(ctx, query, values...); err != nil {
			return fmt.Errorf("insert into %s failed: %w", table, err)
		}
	}

	return nil
}
