package types

import "github.com/google/uuid"
const (
	EventReservationUpdate = "reservation_update"
	EventQueueUpdate       = "queue_update"
	EventMaintenanceChange = "maintenance_change"
)

type MaintenanceChangeEvent struct {
	ResourceID uuid.UUID  `json:"resource_id"`
	ResourceName       string `json:"resource_name"`
	IsUnderMaintenance bool   `json:"is_under_maintenance"`
	ChangedBy          string `json:"changed_by"`
	ChangedAt          int64  `json:"changed_at"`
	Reason             string `json:"reason,omitempty"`
}

type Event struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}
