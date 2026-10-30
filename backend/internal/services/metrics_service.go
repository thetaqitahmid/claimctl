package services

import (
	"context"
	"github.com/google/uuid"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"sort"
	"time"
)

type ResourceMetrics struct {
	ResourceID             uuid.UUID `json:"resourceId"`
	From                   int64     `json:"from"`
	To                     int64     `json:"to"`
	WindowSeconds          int64     `json:"windowSeconds"`
	OccupiedSeconds        int64     `json:"occupiedSeconds"`
	Utilization            float64   `json:"utilization"`
	ClaimCount             int64     `json:"claimCount"`
	OpenClaims             int64     `json:"openClaims"`
	QueueWaitSamples       int64     `json:"queueWaitSamples"`
	MeanQueueWaitSeconds   float64   `json:"meanQueueWaitSeconds"`
	OpenPendingSamples     int64     `json:"openPendingSamples"`
	OpenPendingWaitSeconds int64     `json:"openPendingWaitSeconds"`
}

func metricWindow(from, to int64) error {
	if from < 0 || to <= from || to-from > 90*86400 || to > time.Now().Unix() {
		return ErrInvalidClaim
	}
	return nil
}

func calculateMetrics(id uuid.UUID, rows []db.ClaimctlReservation, from, to int64) ResourceMetrics {
	m := ResourceMetrics{ResourceID: id, From: from, To: to, WindowSeconds: to - from}
	intervals := [][2]int64{}
	var waits int64
	for _, r := range rows {
		if r.CreatedAt.Valid && r.CreatedAt.Int64 >= from && r.CreatedAt.Int64 < to {
			m.ClaimCount++
		}
		open := r.CreatedAt.Int64 < to && (!r.EndTime.Valid || r.EndTime.Int64 > to)
		for _, deadline := range []int64{timestampSeconds(r.HardEndTime.Valid, r.HardEndTime.Time), timestampSeconds(r.LeaseExpiresAt.Valid, r.LeaseExpiresAt.Time), timestampSeconds(r.ScheduledEndTime.Valid, r.ScheduledEndTime.Time)} {
			if deadline >= 0 && deadline <= to {
				open = false
			}
		}
		if open {
			m.OpenClaims++
		}
		if r.StartTime.Valid {
			start := r.StartTime.Int64
			end := to
			if r.EndTime.Valid {
				end = min(end, r.EndTime.Int64)
			}
			for _, deadline := range []int64{timestampSeconds(r.ScheduledEndTime.Valid, r.ScheduledEndTime.Time), timestampSeconds(r.HardEndTime.Valid, r.HardEndTime.Time), timestampSeconds(r.LeaseExpiresAt.Valid, r.LeaseExpiresAt.Time)} {
				if deadline >= 0 {
					end = min(end, deadline)
				}
			}
			if start >= from && start < to {
				m.QueueWaitSamples++
				waits += max(int64(0), start-r.CreatedAt.Int64)
			}
			start = max(start, from)
			if end > start {
				intervals = append(intervals, [2]int64{start, end})
			}
		}
		if open && (!r.StartTime.Valid || r.StartTime.Int64 >= to) {
			end := to
			for _, deadline := range []int64{timestampSeconds(r.HardEndTime.Valid, r.HardEndTime.Time), timestampSeconds(r.LeaseExpiresAt.Valid, r.LeaseExpiresAt.Time)} {
				if deadline >= 0 {
					end = min(end, deadline)
				}
			}
			m.OpenPendingSamples++
			m.OpenPendingWaitSeconds += max(int64(0), end-max(from, r.CreatedAt.Int64))
		}
	}
	sort.Slice(intervals, func(i, j int) bool { return intervals[i][0] < intervals[j][0] })
	var end int64 = from
	for _, v := range intervals {
		m.OccupiedSeconds += max(int64(0), v[1]-max(end, v[0]))
		end = max(end, v[1])
	}
	if m.WindowSeconds > 0 {
		m.Utilization = float64(m.OccupiedSeconds) / float64(m.WindowSeconds)
	}
	if m.QueueWaitSamples > 0 {
		m.MeanQueueWaitSeconds = float64(waits) / float64(m.QueueWaitSamples)
	}
	return m
}
func timestampSeconds(valid bool, t time.Time) int64 {
	if !valid {
		return -1
	}
	return t.Unix()
}
func (s *reservationService) Metrics(ctx context.Context, id uuid.UUID, from, to int64) (ResourceMetrics, error) {
	if err := metricWindow(from, to); err != nil {
		return ResourceMetrics{}, err
	}
	if err := AuthorizeRequestResource(ctx, s.store, id, false); err != nil {
		return ResourceMetrics{}, err
	}
	rows, err := s.store.FindMetricReservations(ctx, id)
	return calculateMetrics(id, rows, from, to), err
}
func (s *reservationService) AggregateMetrics(ctx context.Context, from, to int64) ([]ResourceMetrics, error) {
	auth, ok := AuthorizationFromContext(ctx)
	if !ok || auth.User.Role != "admin" || len(auth.SpaceIDs) > 0 {
		return nil, ErrAccessDenied
	}
	if err := metricWindow(from, to); err != nil {
		return nil, err
	}
	resources, err := s.store.FindClaimCandidates(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ResourceMetrics, 0, len(resources))
	for _, r := range resources {
		m, err := s.Metrics(ctx, r.ID, from, to)
		if err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, nil
}
