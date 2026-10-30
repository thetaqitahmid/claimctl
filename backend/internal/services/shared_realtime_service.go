package services

import (
	"context"
	"sync"
	"time"

	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/types"
)

type sharedRealtimeService struct {
	ctx         context.Context
	q           db.Querier
	mu          sync.Mutex
	clients     map[chan types.Event]struct{}
	cursor      int64
	initialized bool
}

func NewSharedRealtimeService(ctx context.Context, q db.Querier) RealtimeService {
	s := &sharedRealtimeService{ctx: ctx, q: q, clients: make(map[chan types.Event]struct{})}
	initialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	bounds, err := q.RealtimeBounds(initialCtx)
	cancel()
	if err == nil {
		s.cursor = bounds.Newest
		s.initialized = true
	}
	go s.run()
	return s
}

func (s *sharedRealtimeService) Subscribe(ctx context.Context) (chan types.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ctx.Err(); err != nil {
		return nil, err
	}
	ch := make(chan types.Event, 100)
	s.clients[ch] = struct{}{}
	ch <- types.Event{Type: "resync"}
	go func() {
		select {
		case <-ctx.Done():
		case <-s.ctx.Done():
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		if _, ok := s.clients[ch]; ok {
			delete(s.clients, ch)
			close(ch)
		}
	}()
	return ch, nil
}

// Mutations are captured by database triggers in their committing transaction.
func (s *sharedRealtimeService) Broadcast(types.Event) {}

func (s *sharedRealtimeService) publish(event types.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.clients {
		select {
		case ch <- event:
		default:
		drain:
			for {
				select {
				case <-ch:
				default:
					break drain
				}
			}
			ch <- types.Event{Type: "resync", ID: event.ID}
		}
	}
}

func (s *sharedRealtimeService) run() {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	cleanup := time.NewTicker(time.Minute)
	defer cleanup.Stop()
	cursor := s.cursor
	initialized := s.initialized
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-cleanup.C:
			ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
			_ = s.q.CleanupReplicaState(ctx)
			_ = s.q.CleanupSharedValues(ctx)
			_ = s.q.CleanupRateLimits(ctx)
			cancel()
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
			bounds, err := s.q.RealtimeBounds(ctx)
			if err == nil {
				if !initialized || (cursor > 0 && (bounds.Newest < cursor || bounds.Oldest > cursor)) {
					cursor = bounds.Newest
					initialized = true
					s.publish(types.Event{Type: "resync", ID: cursor})
				}
				var rows []db.ClaimctlRealtimeEvent
				rows, err = s.q.ListRealtimeEvents(ctx, cursor)
				if err == nil {
					for _, row := range rows {
						event := types.Event{Type: row.EventType, ID: row.ID}
						if row.ResourceID.Valid && row.SpaceID.Valid {
							event.Payload = map[string]any{"resource_id": row.ResourceID, "space_id": row.SpaceID}
						}
						s.publish(event)
						cursor = row.ID
					}
				}
			}
			if err != nil {
				s.publish(types.Event{Type: "resync"})
			}
			cancel()
		}
	}
}
