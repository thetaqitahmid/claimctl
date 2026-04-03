package services

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/thetaqitahmid/claimctl/internal/db"
)

type AuditService interface {
	Log(ctx context.Context, actorID uuid.UUID, action, entityType, entityID string, changes interface{}, ip string)
	GetAuditLogs(ctx context.Context, limit, offset int32) ([]db.GetAuditLogsRow, error)
}

type auditService struct {
	q db.Querier
}

func NewAuditService(q db.Querier) AuditService {
	return &auditService{q: q}
}

func (s *auditService) Log(ctx context.Context, actorID uuid.UUID, action, entityType, entityID string, changes interface{}, ip string) {
	var changesJSON []byte
	if changes != nil {
		var err error
		changesJSON, err = json.Marshal(changes)
		if err != nil {
			slog.Error("[AUDIT] Failed to marshal changes for audit log", "error", err)
			return
		}
	}

	actorPgUUID := pgtype.UUID{Bytes: actorID, Valid: true}
	if actorID == uuid.Nil {
		actorPgUUID.Valid = false
	}

	entityIDPg := pgtype.Text{String: entityID, Valid: entityID != ""}
	ipPg := pgtype.Text{String: ip, Valid: ip != ""}

	_, err := s.q.CreateAuditLog(ctx, db.CreateAuditLogParams{
		ActorID:    actorPgUUID,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityIDPg,
		Changes:    changesJSON,
		IpAddress:  ipPg,
	})

	if err != nil {
		slog.Error("[AUDIT] Failed to write audit log", "error", err)
	}
}

func (s *auditService) GetAuditLogs(ctx context.Context, limit, offset int32) ([]db.GetAuditLogsRow, error) {
	return s.q.GetAuditLogs(ctx, db.GetAuditLogsParams{
		Limit:  limit,
		Offset: offset,
	})
}
