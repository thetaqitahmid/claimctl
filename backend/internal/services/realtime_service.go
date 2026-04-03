package services

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/thetaqitahmid/claimctl/internal/types"
)

type RealtimeService interface {
	Subscribe(ctx context.Context) (chan types.Event, error)
	Broadcast(event types.Event)
}

type realtimeService struct {
	clients    map[chan types.Event]struct{}
	clientsMux sync.RWMutex
}

func NewRealtimeService() RealtimeService {
	return &realtimeService{
		clients: make(map[chan types.Event]struct{}),
	}
}

func (s *realtimeService) Subscribe(ctx context.Context) (chan types.Event, error) {
	// Create a new channel for the client
	// Using a buffered channel to prevent blocking the broadcaster if the client is slow
	clientChan := make(chan types.Event, 10)

	s.clientsMux.Lock()
	s.clients[clientChan] = struct{}{}
	s.clientsMux.Unlock()

	slog.Info("New client subscribed to realtime events", "total_clients", len(s.clients))

	// Handle client disconnection
	go func() {
		<-ctx.Done()
		s.removeClient(clientChan)
	}()

	return clientChan, nil
}

func (s *realtimeService) Broadcast(event types.Event) {
	s.clientsMux.RLock()
	defer s.clientsMux.RUnlock()

	payloadBytes, _ := json.Marshal(event.Payload)
	slog.Info("Broadcasting event", "type", event.Type, "payload_preview", string(payloadBytes), "active_clients", len(s.clients))

	for clientChan := range s.clients {
		select {
		case clientChan <- event:
			// Event sent successfully
		default:
			// Client channel is full, skip this client or handle accordingly
			slog.Warn("Client channel full, dropping event", "type", event.Type)
		}
	}
}

func (s *realtimeService) removeClient(clientChan chan types.Event) {
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()

	if _, ok := s.clients[clientChan]; ok {
		delete(s.clients, clientChan)
		close(clientChan)
		slog.Info("Client disconnected from realtime events", "total_clients", len(s.clients))
	}
}
