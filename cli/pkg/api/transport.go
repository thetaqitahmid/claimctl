package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const maxResponseBytes = 64 << 20

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("api error: %s (status: %d)", e.Body, e.StatusCode)
}
func (c *Client) context() context.Context {
	if c.Context != nil {
		return c.Context
	}
	return context.Background()
}

func (c *Client) requestHeaders(method, path string, body any, headers http.Header) ([]byte, error) {
	var data []byte
	var err error
	if body != nil {
		data, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}
	attempts := 0
	if method == http.MethodGet || method == http.MethodHead || headers.Get("Idempotency-Key") != "" {
		attempts = c.RetryCount
	}
	for i := 0; i <= attempts; i++ {
		req, err := http.NewRequestWithContext(c.context(), method, c.BaseURL+path, bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		for key, values := range headers {
			req.Header[key] = values
		}
		if c.Token != "" {
			req.Header.Set("Authorization", "Bearer "+c.Token)
		}
		wait := c.RetryWait * time.Duration(1<<min(i, 6))
		resp, err := c.HttpClient.Do(req)
		if err == nil {
			if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
				if seconds, parseErr := strconv.ParseInt(retryAfter, 10, 64); parseErr == nil && seconds > 0 {
					wait = max(wait, time.Duration(min(seconds, 60))*time.Second)
				} else if deadline, parseErr := http.ParseTime(retryAfter); parseErr == nil {
					wait = max(wait, min(time.Until(deadline), 60*time.Second))
				}
			}
			result, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
			resp.Body.Close()
			if readErr != nil {
				return nil, readErr
			}
			if len(result) > maxResponseBytes {
				return nil, fmt.Errorf("response exceeds size limit")
			}
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return result, nil
			}
			err = &APIError{StatusCode: resp.StatusCode, Body: string(result)}
			if resp.StatusCode < 500 && resp.StatusCode != 429 {
				return nil, err
			}
		}
		if i == attempts {
			return nil, err
		}
		timer := time.NewTimer(wait)
		select {
		case <-c.context().Done():
			timer.Stop()
			return nil, c.context().Err()
		case <-timer.C:
		}
	}
	return nil, fmt.Errorf("request failed")
}

func (c *Client) waitForReservation(id string, timeout, interval int, progress func(string, *int32)) error {
	if timeout <= 0 || interval <= 0 {
		return fmt.Errorf("timeout and poll interval must be positive")
	}
	ctx, cancel := context.WithTimeout(c.context(), time.Duration(timeout)*time.Second)
	defer cancel()
	poll := *c
	poll.Context = ctx
	for {
		r, err := poll.GetReservation(id)
		if err != nil {
			return err
		}
		if progress != nil {
			progress(r.Status, r.QueuePosition)
		}
		switch r.Status {
		case "active":
			return nil
		case "cancelled", "completed", "expired":
			return fmt.Errorf("reservation was %s", r.Status)
		}
		timer := time.NewTimer(time.Duration(interval) * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("waiting for reservation: %w", ctx.Err())
		case <-timer.C:
		}
	}
}
