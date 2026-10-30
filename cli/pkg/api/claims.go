package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type ClaimRequest struct {
	ResourceID     string `json:"resourceId,omitempty"`
	Name           string `json:"name,omitempty"`
	Type           string `json:"type,omitempty"`
	LabelExpr      string `json:"labelExpr,omitempty"`
	RequireHealthy bool   `json:"requireHealthy"`
	NoQueue        bool   `json:"noQueue"`
	Duration       string `json:"duration,omitempty"`
	LeaseDuration  string `json:"leaseDuration,omitempty"`
	Indefinite     bool   `json:"indefinite"`
}
type ReservationPolicy struct {
	DefaultDuration string `json:"defaultDuration"`
	MaxDuration     string `json:"maxDuration"`
	MaxLiveClaims   int64  `json:"maxLiveClaims"`
	AllowIndefinite bool   `json:"allowIndefinite"`
}
type BundleRequest struct {
	ResourceIDs   []string `json:"resourceIds"`
	Duration      string   `json:"duration,omitempty"`
	LeaseDuration string   `json:"leaseDuration,omitempty"`
}

func NewIdempotencyKey() (string, error) {
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
func (c *Client) Claim(req ClaimRequest, key string) (*Reservation, error) {
	if key == "" {
		var err error
		key, err = NewIdempotencyKey()
		if err != nil {
			return nil, err
		}
	}
	data, err := c.requestHeaders("POST", "/api/reservations/claim", req, http.Header{"Idempotency-Key": {key}})
	if err != nil {
		return nil, err
	}
	var result Reservation
	err = json.Unmarshal(data, &result)
	return &result, err
}
func (c *Client) Bundle(req BundleRequest, key string) ([]Reservation, error) {
	if key == "" {
		var err error
		key, err = NewIdempotencyKey()
		if err != nil {
			return nil, err
		}
	}
	data, err := c.requestHeaders("POST", "/api/reservations/bundle", req, http.Header{"Idempotency-Key": {key}})
	if err != nil {
		return nil, err
	}
	var result []Reservation
	err = json.Unmarshal(data, &result)
	return result, err
}
func (c *Client) Renew(id, duration string) (*Reservation, error) {
	req := map[string]string{}
	if duration != "" {
		req["leaseDuration"] = duration
	}
	data, err := c.request("POST", "/api/reservations/"+url.PathEscape(id)+"/renew", req)
	if err != nil {
		return nil, err
	}
	var result Reservation
	err = json.Unmarshal(data, &result)
	return &result, err
}
func (c *Client) Policy() (*ReservationPolicy, error) {
	data, err := c.request("GET", "/api/reservations/policy", nil)
	if err != nil {
		return nil, err
	}
	var result ReservationPolicy
	err = json.Unmarshal(data, &result)
	return &result, err
}
func (c *Client) Metrics(id string, from, to int64) (json.RawMessage, error) {
	path := "/api/admin/metrics"
	if id != "" {
		path = "/api/resources/" + url.PathEscape(id) + "/metrics"
	}
	q := url.Values{}
	if from != 0 {
		q.Set("from", fmt.Sprint(from))
	}
	if to != 0 {
		q.Set("to", fmt.Sprint(to))
	}
	data, err := c.request("GET", path+"?"+q.Encode(), nil)
	if err == nil && !json.Valid(data) {
		return nil, fmt.Errorf("invalid metrics JSON")
	}
	return data, err
}
func (c *Client) GenerateScopedToken(name, expires string, scopes, spaces []string) (*GenerateTokenResponse, error) {
	req := map[string]any{"name": name}
	if expires != "" {
		req["expires_in"] = expires
	}
	if scopes != nil {
		req["scopes"] = scopes
	}
	if spaces != nil {
		req["space_ids"] = spaces
	}
	data, err := c.request("POST", "/api/tokens", req)
	if err != nil {
		return nil, err
	}
	var result GenerateTokenResponse
	err = json.Unmarshal(data, &result)
	return &result, err
}
