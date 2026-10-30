package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/valyala/fasthttp"
)

type RealtimeHandler struct {
	realtimeService services.RealtimeService
	db              db.Querier
}

func NewRealtimeHandler(realtimeService services.RealtimeService, q db.Querier) *RealtimeHandler {
	return &RealtimeHandler{realtimeService: realtimeService, db: q}
}

func (h *RealtimeHandler) HandleSSE(c *fiber.Ctx) error {
	auth, ok := c.Locals("authorization").(services.Authorization)
	if !ok {
		return fiber.ErrUnauthorized
	}
	lifetime := 15 * time.Minute
	if !auth.ExpiresAt.IsZero() && time.Until(auth.ExpiresAt) < lifetime {
		lifetime = time.Until(auth.ExpiresAt)
	}
	if lifetime <= 0 {
		return fiber.ErrUnauthorized
	}
	streamCtx, cancel := context.WithTimeout(context.Background(), lifetime)
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-store")
	c.Set("X-Accel-Buffering", "no")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	// Create a channel for this client
	events, err := h.realtimeService.Subscribe(streamCtx)
	if err != nil {
		cancel()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to subscribe"})
	}

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		defer cancel()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		expiryTimer := time.NewTimer(lifetime)
		defer expiryTimer.Stop()
		updateExpiry := func(current services.Authorization) {
			if !current.ExpiresAt.IsZero() {
				expiryTimer.Reset(time.Until(current.ExpiresAt))
			}
		}
		slog.Info("SSE connection established")

		for {
			select {
			case event, ok := <-events:
				if !ok {
					return
				}
				checkCtx, checkCancel := context.WithTimeout(streamCtx, 5*time.Second)
				current, err := services.RefreshAuthorization(checkCtx, h.db, auth)
				allowed := err == nil && services.CanDeliverEvent(checkCtx, h.db, current, event)
				checkCancel()
				if err != nil {
					return
				}
				updateExpiry(current)
				if !allowed {
					continue
				}
				if !current.ExpiresAt.IsZero() && !current.ExpiresAt.After(time.Now()) {
					return
				}
				data, err := json.Marshal(event)
				if err != nil {
					slog.Error("Failed to marshal event", "error", err)
					continue
				}
				if event.ID > 0 {
					fmt.Fprintf(w, "id: %d\n", event.ID)
				}
				fmt.Fprintf(w, "data: %s\n\n", data)
				if err := w.Flush(); err != nil {
					slog.Error("Failed to flush", "error", err)
					return
				}
			case <-streamCtx.Done():
				return
			case <-expiryTimer.C:
				return
			case <-ticker.C:
				checkCtx, checkCancel := context.WithTimeout(streamCtx, 5*time.Second)
				current, err := services.RefreshAuthorization(checkCtx, h.db, auth)
				checkCancel()
				if err != nil {
					return
				}
				updateExpiry(current)
				// Keep-alive heartbeat
				fmt.Fprintf(w, ": keep-alive\n\n")
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	}))

	return nil
}
