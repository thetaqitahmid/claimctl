package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/utils"
)

var ErrClaimConflict = errors.New("claim unavailable or quota exceeded")
var ErrInvalidClaim = errors.New("invalid claim request")

type ReservationPolicy struct {
	DefaultDuration string `json:"defaultDuration"`
	MaxDuration     string `json:"maxDuration"`
	MaxLiveClaims   int64  `json:"maxLiveClaims"`
	AllowIndefinite bool   `json:"allowIndefinite"`
}

func LoadReservationPolicy() (ReservationPolicy, error) {
	p := ReservationPolicy{"1h", "24h", 8, false}
	for key, dst := range map[string]*string{"RESERVATION_DEFAULT_DURATION": &p.DefaultDuration, "RESERVATION_MAX_DURATION": &p.MaxDuration} {
		if v, ok := os.LookupEnv(key); ok {
			*dst = v
		}
	}
	if v, ok := os.LookupEnv("RESERVATION_MAX_LIVE_CLAIMS"); ok {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < 1 {
			return p, fmt.Errorf("invalid reservation quota")
		}
		p.MaxLiveClaims = n
	}
	if v, ok := os.LookupEnv("RESERVATION_ALLOW_INDEFINITE"); ok {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return p, err
		}
		p.AllowIndefinite = b
	}
	d, e := utils.ParseDurationWithDays(p.DefaultDuration)
	m, me := utils.ParseDurationWithDays(p.MaxDuration)
	if e != nil || me != nil || d < time.Second || m < d {
		return p, fmt.Errorf("invalid reservation durations")
	}
	return p, nil
}

type ClaimRequest struct {
	ResourceID     uuid.UUID `json:"resourceId"`
	Name           string    `json:"name"`
	Type           string    `json:"type"`
	LabelExpr      string    `json:"labelExpr"`
	RequireHealthy bool      `json:"requireHealthy"`
	NoQueue        bool      `json:"noQueue"`
	Duration       *string   `json:"duration"`
	LeaseDuration  *string   `json:"leaseDuration"`
	Indefinite     bool      `json:"indefinite"`
}

type ClaimService interface {
	Claim(context.Context, uuid.UUID, ClaimRequest, string) (*db.ClaimctlReservation, error)
	Policy() ReservationPolicy
}

func (s *reservationService) Policy() ReservationPolicy { return s.policy }

func (s *reservationService) Claim(ctx context.Context, userID uuid.UUID, req ClaimRequest, key string) (*db.ClaimctlReservation, error) {
	if len(key) > 200 {
		return nil, fmt.Errorf("%w: idempotency key too long", ErrInvalidClaim)
	}
	if req.ResourceID == uuid.Nil && req.Name == "" && req.Type == "" && req.LabelExpr == "" {
		return nil, fmt.Errorf("%w: selector required", ErrInvalidClaim)
	}
	filter, err := utils.ParseLabelFilter(req.LabelExpr)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid label expression", ErrInvalidClaim)
	}
	var interval pgtype.Interval
	if req.Indefinite {
		if !s.policy.AllowIndefinite || req.Duration != nil {
			return nil, fmt.Errorf("%w: indefinite duration not permitted", ErrInvalidClaim)
		}
	} else {
		value := s.policy.DefaultDuration
		if req.Duration != nil {
			value = *req.Duration
		}
		duration, e := utils.ParseDurationWithDays(value)
		max, _ := utils.ParseDurationWithDays(s.policy.MaxDuration)
		if e != nil || duration < time.Microsecond || duration > max {
			return nil, fmt.Errorf("%w: duration must be positive and within policy", ErrInvalidClaim)
		}
		interval = pgtype.Interval{Microseconds: duration.Microseconds(), Valid: true}
	}
	lease, err := s.parseLease(req.LeaseDuration)
	if err != nil {
		return nil, err
	}
	encoded, _ := json.Marshal(req)
	sum := sha256.Sum256(encoded)
	fingerprint := hex.EncodeToString(sum[:])
	var result db.ClaimctlReservation
	replay := false
	err = s.store.ExecTx(ctx, func(q db.Querier) error {
		user, e := q.LockClaimUser(ctx, userID)
		if e != nil || user.Status != "active" {
			return ErrAccessDenied
		}
		if auth, ok := AuthorizationFromContext(ctx); ok {
			if auth.User.ID != userID || CredentialVersion(auth.User.Password) != CredentialVersion(user.Password) {
				return ErrAccessDenied
			}
			auth.User = user
			ctx = WithAuthorization(ctx, auth)
		}
		if key != "" {
			saved, e := q.GetClaimIdempotency(ctx, db.GetClaimIdempotencyParams{UserID: userID, Key: key})
			if e == nil {
				if saved.Fingerprint != fingerprint {
					return ErrClaimConflict
				}
				if AuthorizeRequestResource(ctx, q, savedResource(saved.Result), false) != nil {
					return ErrAccessDenied
				}
				replay = true
				return json.Unmarshal(saved.Result, &result)
			}
			if !errors.Is(e, pgx.ErrNoRows) {
				return e
			}
		}
		count, e := q.CountLiveClaims(ctx, userID)
		if e != nil {
			return e
		}
		if count >= s.policy.MaxLiveClaims {
			return ErrClaimConflict
		}
		resources, e := q.FindClaimCandidates(ctx)
		if e != nil {
			return e
		}
		var queued *db.ClaimctlResource
		for _, candidate := range resources {
			if !matchesClaimSelector(req, filter, candidate) {
				continue
			}
			if _, e = AuthorizeResource(ctx, q, candidate.ID, userID, user.Role == "admin"); e != nil {
				if req.ResourceID != uuid.Nil {
					return e
				}
				continue
			}
			_, e := q.AcquireResourceLock(ctx, candidate.ID)
			if e != nil {
				return e
			}
			candidate, e = q.FindResourceById(ctx, candidate.ID)
			if e != nil {
				return e
			}
			if !matchesClaimSelector(req, filter, candidate) {
				continue
			}
			if e = AuthorizeSpace(ctx, q, candidate.SpaceID, userID, user.Role == "admin"); e != nil {
				if req.ResourceID != uuid.Nil {
					return e
				}
				continue
			}
			if candidate.IsUnderMaintenance.Valid && candidate.IsUnderMaintenance.Bool {
				continue
			}
			if req.RequireHealthy {
				health, e := q.GetHealthStatus(ctx, candidate.ID)
				if e != nil || health.Status != "healthy" {
					continue
				}
			}
			_, e = q.FindUserReservationForResource(ctx, db.FindUserReservationForResourceParams{UserID: userID, ResourceID: candidate.ID})
			if e == nil {
				continue
			}
			if !errors.Is(e, pgx.ErrNoRows) {
				return e
			}
			if e = reclaimAndPromote(ctx, q, candidate.ID); e != nil {
				return e
			}
			live, e := q.GetQueueForResource(ctx, candidate.ID)
			if e != nil {
				return e
			}
			if len(live) > 0 {
				if !req.NoQueue && queued == nil {
					copy := candidate
					queued = &copy
				}
				continue
			}
			queued = &candidate
			break
		}
		if queued == nil {
			return ErrClaimConflict
		}
		result, e = q.CreateReservation(ctx, db.CreateReservationParams{ResourceID: queued.ID, UserID: userID, Duration: interval})
		if e != nil {
			return e
		}
		max, _ := utils.ParseDurationWithDays(s.policy.MaxDuration)
		result, e = q.SetReservationLifetime(ctx, db.SetReservationLifetimeParams{ID: result.ID, MaxDuration: pgtype.Interval{Microseconds: max.Microseconds(), Valid: true}, LeaseDuration: lease})
		if e != nil {
			return e
		}
		if key != "" {
			payload, e := json.Marshal(result)
			if e != nil {
				return e
			}
			return q.SaveClaimIdempotency(ctx, db.SaveClaimIdempotencyParams{UserID: userID, Key: key, Fingerprint: fingerprint, ReservationID: result.ID, Result: payload})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !replay && s.webhookSvc != nil && s.notificationSvc != nil && s.realtimeSvc != nil {
		s.publishCreated(ctx, result)
	}
	return &result, nil
}

func matchesClaimSelector(req ClaimRequest, filter utils.FilterExpr, resource db.ClaimctlResource) bool {
	if req.ResourceID != uuid.Nil && req.ResourceID != resource.ID || req.Name != "" && req.Name != resource.Name || req.Type != "" && req.Type != resource.Type {
		return false
	}
	labels := make([]string, len(resource.Labels))
	for i, label := range resource.Labels {
		labels[i], _ = label.(string)
	}
	return filter == nil || filter.Matches(labels)
}

func savedResource(payload []byte) uuid.UUID {
	var r db.ClaimctlReservation
	_ = json.Unmarshal(payload, &r)
	return r.ResourceID
}
