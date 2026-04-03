package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/valyala/fasthttp"
)

type RealtimeHandler struct {
	realtimeService services.RealtimeService
}

func NewRealtimeHandler(realtimeService services.RealtimeService) *RealtimeHandler {
	return &RealtimeHandler{realtimeService: realtimeService}
}

func (h *RealtimeHandler) HandleSSE(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	// Create a channel for this client
	events, err := h.realtimeService.Subscribe(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to subscribe"})
	}

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		slog.Info("SSE connection established")

		for {
			select {
			case event, ok := <-events:
				if !ok {
					return
				}
				data, err := json.Marshal(event)
				if err != nil {
					slog.Error("Failed to marshal event", "error", err)
					continue
				}
				fmt.Fprintf(w, "data: %s\n\n", data)
				if err := w.Flush(); err != nil {
					slog.Error("Failed to flush", "error", err)
					return
				}
			case <-time.After(30 * time.Second):
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
