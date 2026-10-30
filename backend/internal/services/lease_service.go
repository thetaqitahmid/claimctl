package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/utils"
)

type LifecycleService interface {
	Renew(context.Context, uuid.UUID, uuid.UUID, *string) (*db.ClaimctlReservation, error)
	Bundle(context.Context, uuid.UUID, BundleRequest, string) ([]db.ClaimctlReservation, error)
	Metrics(context.Context, uuid.UUID, int64, int64) (ResourceMetrics, error)
	AggregateMetrics(context.Context, int64, int64) ([]ResourceMetrics, error)
}

type BundleRequest struct {
	ResourceIDs   []uuid.UUID `json:"resourceIds"`
	Duration      *string     `json:"duration"`
	LeaseDuration *string     `json:"leaseDuration"`
}

func (s *reservationService) parseLease(value *string) (pgtype.Interval, error) {
	if value == nil {
		return pgtype.Interval{}, nil
	}
	d, err := utils.ParseDurationWithDays(*value)
	max, _ := utils.ParseDurationWithDays(s.policy.MaxDuration)
	if err != nil || d < time.Microsecond || d > max {
		return pgtype.Interval{}, ErrInvalidClaim
	}
	return pgtype.Interval{Microseconds: d.Microseconds(), Valid: true}, nil
}

func (s *reservationService) Renew(ctx context.Context, actor, id uuid.UUID, duration *string) (*db.ClaimctlReservation, error) {
	lease, err := s.parseLease(duration)
	if err != nil {
		return nil, err
	}
	var result db.ClaimctlReservation
	err = s.store.ExecTx(ctx, func(q db.Querier) error {
		user, err := q.LockClaimUser(ctx, actor)
		if err != nil || user.Status != "active" {
			return ErrAccessDenied
		}
		if auth, ok := AuthorizationFromContext(ctx); ok {
			if auth.User.ID != actor || CredentialVersion(auth.User.Password) != CredentialVersion(user.Password) {
				return ErrAccessDenied
			}
			auth.User = user
			ctx = WithAuthorization(ctx, auth)
		}
		res, err := q.FindReservationById(ctx, id)
		if err != nil {
			return err
		}
		if res.UserID != actor {
			return ErrAccessDenied
		}
		if err := AuthorizeRequestResource(ctx, q, res.ResourceID, false); err != nil {
			return err
		}
		if _, err = q.AcquireResourceLock(ctx, res.ResourceID); err != nil {
			return err
		}
		result, err = q.RenewReservationLease(ctx, db.RenewReservationLeaseParams{ID: id, LeaseDuration: lease})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrClaimConflict
		}
		return err
	})
	return &result, err
}

func reclaimAndPromote(ctx context.Context, q db.Querier, resource uuid.UUID) error {
	reclaimed, err := q.ReclaimResourceReservations(ctx, resource)
	if err != nil {
		return err
	}
	if len(reclaimed) == 0 {
		return nil
	}
	return promoteLiveQueue(ctx, q, resource)
}

func promoteLiveQueue(ctx context.Context, q db.Querier, resource uuid.UUID) error {
	_, err := q.FindActiveReservationByResource(ctx, resource)
	if err == nil {
		return normalizeQueue(ctx, q, resource)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	next, err := q.GetNextInQueue(ctx, resource)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	activated, err := q.ActivateReservation(ctx, next.ID)
	if err != nil {
		return err
	}
	if activated != 1 {
		return ErrClaimConflict
	}
	return normalizeQueue(ctx, q, resource)
}

func normalizeQueue(ctx context.Context, q db.Querier, resource uuid.UUID) error {
	pending, err := q.FindPendingReservationsByResource(ctx, resource)
	if err != nil {
		return err
	}
	for i, r := range pending {
		if err = q.SetPendingQueuePosition(ctx, db.SetPendingQueuePositionParams{ID: r.ID, QueuePosition: pgtype.Int4{Int32: int32(i + 1), Valid: true}}); err != nil {
			return err
		}
	}
	return nil
}

func (s *reservationService) Bundle(ctx context.Context, actor uuid.UUID, req BundleRequest, key string) ([]db.ClaimctlReservation, error) {
	if len(req.ResourceIDs) == 0 || int64(len(req.ResourceIDs)) > s.policy.MaxLiveClaims || len(key) > 200 {
		return nil, ErrInvalidClaim
	}
	req.ResourceIDs = append([]uuid.UUID(nil), req.ResourceIDs...)
	sort.Slice(req.ResourceIDs, func(i, j int) bool { return req.ResourceIDs[i].String() < req.ResourceIDs[j].String() })
	for i, id := range req.ResourceIDs {
		if id == uuid.Nil || i > 0 && id == req.ResourceIDs[i-1] {
			return nil, ErrInvalidClaim
		}
	}
	lease, err := s.parseLease(req.LeaseDuration)
	if err != nil {
		return nil, err
	}
	duration := s.policy.DefaultDuration
	if req.Duration != nil {
		duration = *req.Duration
	}
	d, err := utils.ParseDurationWithDays(duration)
	max, _ := utils.ParseDurationWithDays(s.policy.MaxDuration)
	if err != nil || d < time.Microsecond || d > max {
		return nil, ErrInvalidClaim
	}
	encoded, _ := json.Marshal(req)
	hash := sha256.Sum256(encoded)
	fingerprint := hex.EncodeToString(hash[:])
	result := []db.ClaimctlReservation{}
	replay := false
	err = s.store.ExecTx(ctx, func(q db.Querier) error {
		user, e := q.LockClaimUser(ctx, actor)
		if e != nil || user.Status != "active" {
			return ErrAccessDenied
		}
		if auth, ok := AuthorizationFromContext(ctx); ok {
			if auth.User.ID != actor || CredentialVersion(auth.User.Password) != CredentialVersion(user.Password) {
				return ErrAccessDenied
			}
			auth.User = user
			ctx = WithAuthorization(ctx, auth)
		}
		if key != "" {
			saved, e := q.GetBundleIdempotency(ctx, db.GetBundleIdempotencyParams{UserID: actor, Key: key})
			if e == nil {
				if saved.Fingerprint != fingerprint {
					return ErrClaimConflict
				}
				if e = json.Unmarshal(saved.Result, &result); e != nil {
					return e
				}
				for _, r := range result {
					if e = AuthorizeRequestResource(ctx, q, r.ResourceID, false); e != nil {
						return e
					}
				}
				replay = true
				return nil
			}
			if !errors.Is(e, pgx.ErrNoRows) {
				return e
			}
		}
		count, e := q.CountLiveClaims(ctx, actor)
		if e != nil {
			return e
		}
		if count+int64(len(req.ResourceIDs)) > s.policy.MaxLiveClaims {
			return ErrClaimConflict
		}
		for _, id := range req.ResourceIDs {
			if _, e = AuthorizeResource(ctx, q, id, actor, user.Role == "admin"); e != nil {
				return e
			}
			_, e = q.AcquireResourceLock(ctx, id)
			if e != nil {
				return e
			}
			resource, e := q.FindResourceById(ctx, id)
			if e != nil {
				return e
			}
			if e = AuthorizeSpace(ctx, q, resource.SpaceID, actor, user.Role == "admin"); e != nil {
				return e
			}
			if resource.IsUnderMaintenance.Bool {
				return ErrClaimConflict
			}
			if e = reclaimAndPromote(ctx, q, id); e != nil {
				return e
			}
			live, e := q.GetQueueForResource(ctx, id)
			if e != nil {
				return e
			}
			if len(live) > 0 {
				return ErrClaimConflict
			}
		}
		for _, id := range req.ResourceIDs {
			r, e := q.CreateReservation(ctx, db.CreateReservationParams{ResourceID: id, UserID: actor, Duration: pgtype.Interval{Microseconds: d.Microseconds(), Valid: true}})
			if e != nil {
				return e
			}
			r, e = q.SetReservationLifetime(ctx, db.SetReservationLifetimeParams{ID: r.ID, MaxDuration: pgtype.Interval{Microseconds: max.Microseconds(), Valid: true}, LeaseDuration: lease})
			if e != nil {
				return e
			}
			result = append(result, r)
		}
		if key != "" {
			payload, e := json.Marshal(result)
			if e != nil {
				return e
			}
			return q.SaveBundleIdempotency(ctx, db.SaveBundleIdempotencyParams{UserID: actor, Key: key, Fingerprint: fingerprint, Result: payload})
		}
		return nil
	})
	if err == nil && !replay && s.webhookSvc != nil && s.notificationSvc != nil && s.realtimeSvc != nil {
		for _, r := range result {
			s.publishCreated(ctx, r)
		}
	}
	return result, err
}
