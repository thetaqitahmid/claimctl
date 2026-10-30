package services

import "github.com/google/uuid"

type TokenOptions struct {
	Scopes   []string    `json:"scopes"`
	SpaceIDs []uuid.UUID `json:"space_ids"`
}

var TokenScopes = map[string]bool{
	"read": false, "reserve": false,
	"resources:write": true, "health:write": true, "webhooks:write": true,
	"secrets:write": true, "secrets:read": true, "backup:read": true,
	"restore:write": true, "audit:read": true, "users:write": true,
	"users:read":   true,
	"spaces:write": true, "groups:write": true, "settings:read": true,
	"settings:write": true, "reservations:write": true, "metrics:read": true,
}
