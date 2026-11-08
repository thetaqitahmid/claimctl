package server

import (
	"log/slog"
	"net"
	"net/netip"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/thetaqitahmid/claimctl/internal/utils"
)

func trustedProxy(addr string, cidrs []netip.Prefix) bool {
	ip, err := netip.ParseAddr(addr)
	if err != nil {
		return false
	}
	for _, cidr := range cidrs {
		if cidr.Contains(ip.Unmap()) {
			return true
		}
	}
	return false
}

func TrustedProxyMiddleware() fiber.Handler {
	var cidrs []netip.Prefix
	for _, raw := range strings.Split(utils.GetEnv("TRUSTED_PROXY_CIDRS", ""), ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		cidr, err := netip.ParsePrefix(raw)
		if err != nil {
			slog.Warn("Ignoring invalid TRUSTED_PROXY_CIDRS entry", "entry", raw, "error", err)
			continue
		}
		cidrs = append(cidrs, cidr)
	}
	return func(c *fiber.Ctx) error {
		peer := c.Context().RemoteIP().String()
		proto := "http"
		if c.Context().IsTLS() {
			proto = "https"
		}
		host := c.Get("Host")
		client := peer
		var forwardedHeaders []string
		c.Request().Header.VisitAll(func(key, _ []byte) {
			name := strings.ToLower(string(key))
			if strings.HasPrefix(name, "x-forwarded-") || name == "forwarded" || name == "x-real-ip" || name == "x-url-scheme" || name == "front-end-https" {
				forwardedHeaders = append(forwardedHeaders, string(key))
			}
		})
		forwarded := len(forwardedHeaders) > 0
		if forwarded && !trustedProxy(peer, cidrs) {
			return fiber.ErrBadRequest
		}
		if forwarded {
			if c.Get("Forwarded") != "" {
				return fiber.ErrBadRequest
			}
			if value := c.Get("X-Forwarded-Proto"); value != "" {
				if value != "http" && value != "https" {
					return fiber.ErrBadRequest
				}
				proto = value
			}
			if value := c.Get("X-Forwarded-Host"); value != "" {
				host = value
			}
			hops := strings.Split(c.Get("X-Forwarded-For"), ",")
			for i := len(hops) - 1; i >= 0 && trustedProxy(client, cidrs); i-- {
				hop := strings.TrimSpace(hops[i])
				if hop == "" {
					continue
				}
				ip, err := netip.ParseAddr(hop)
				if err != nil {
					return fiber.ErrBadRequest
				}
				client = ip.Unmap().String()
			}
		}
		parsed, err := url.Parse(proto + "://" + host)
		if err != nil || parsed.Host != host || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Hostname() == "" || strings.ContainsAny(host, ", \\\t\r\n") {
			return fiber.ErrBadRequest
		}
		if port := parsed.Port(); port != "" {
			if _, err := net.LookupPort("tcp", port); err != nil {
				return fiber.ErrBadRequest
			}
		}
		c.Locals("external_origin", proto+"://"+host)
		c.Locals("client_ip", client)
		for _, name := range forwardedHeaders {
			c.Request().Header.Del(name)
		}
		// Fiber's proxy-aware helpers receive only canonicalized values.
		c.Request().Header.Set("X-Forwarded-Host", host)
		c.Request().Header.Set("X-Forwarded-Proto", proto)
		c.Request().Header.Set("X-Forwarded-For", client)
		return c.Next()
	}
}

func ExternalOrigin(c *fiber.Ctx) string {
	if origin, ok := c.Locals("external_origin").(string); ok {
		return origin
	}
	proto := "http"
	if c.Context().IsTLS() {
		proto = "https"
	}
	return proto + "://" + c.Get("Host")
}
func ClientIP(c *fiber.Ctx) string {
	if ip, ok := c.Locals("client_ip").(string); ok {
		return ip
	}
	return c.Context().RemoteIP().String()
}
