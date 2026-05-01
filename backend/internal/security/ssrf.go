// Package security implements SSRF defenses for endpoints that fetch
// user-supplied URLs (icon library, future wallpaper fetch, etc).
//
// Threat model and design rationale: see memory/feedback_ssrf_protection.md.
//
// Usage:
//   target, err := security.ValidateURL(rawURL, cfg)   // DNS resolve + IP block check
//   if err != nil { return err }
//   client := security.BuildSafeClient(target, cfg, 10*time.Second)
//   resp, err := client.Do(...)
package security

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Config struct {
	// AllowPrivate, when true, bypasses the private/loopback/link-local IP
	// blocklist. Cloud-metadata IPs (169.254.169.254) remain blocked even
	// with this flag, to avoid silent credential leakage on cloud hosts.
	AllowPrivate bool

	// AllowedHosts lets specific hostnames bypass the IP blocklist. Useful
	// for "I run a private CDN at internal.example.com" deployments.
	AllowedHosts []string
}

var (
	ErrInvalidURL    = errors.New("invalid URL")
	ErrSchemeBlocked = errors.New("only http(s) URLs are allowed")
	ErrPrivateIP     = errors.New("URL resolves to a private IP — disabled by default; set MOON_ALLOW_PRIVATE_FETCH=true to override")
	ErrCloudMetadata = errors.New("URL resolves to cloud metadata service (169.254.169.254) — always blocked to prevent credential leakage")
	ErrDNSFailed     = errors.New("DNS resolution failed")
)

// FetchTarget is the validated, resolved target for a safe fetch.
type FetchTarget struct {
	URL  string
	Host string
	Port string
	IPs  []net.IP // resolved IPs that passed the blocklist check
}

// ValidateURL parses + DNS resolves + checks IPs against the blocklist.
// Returns a target with the resolved IPs that BuildSafeClient pins to (defends
// against DNS rebinding by skipping the second resolve at fetch time).
func ValidateURL(rawURL string, cfg Config) (*FetchTarget, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("%w: scheme=%q", ErrSchemeBlocked, u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return nil, fmt.Errorf("%w: empty hostname", ErrInvalidURL)
	}
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDNSFailed, err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("%w: no IPs for %s", ErrDNSFailed, host)
	}

	allowedHost := false
	for _, allowed := range cfg.AllowedHosts {
		if strings.EqualFold(host, allowed) {
			allowedHost = true
			break
		}
	}

	for _, ip := range ips {
		// Cloud metadata is ALWAYS blocked, even with AllowPrivate or AllowedHost.
		if isCloudMetadata(ip) {
			return nil, ErrCloudMetadata
		}
		if !allowedHost && !cfg.AllowPrivate && IsPrivate(ip) {
			return nil, fmt.Errorf("%w (host %s → %s)", ErrPrivateIP, host, ip.String())
		}
	}

	return &FetchTarget{URL: rawURL, Host: host, Port: port, IPs: ips}, nil
}

// BuildSafeClient returns an http.Client that:
//   - Pins the connection to the IP that was validated (no DNS rebinding)
//   - Times out after the given duration
//   - Refuses to follow redirects (caller must check resp.StatusCode 3xx)
//
// TLS still uses the original hostname for SNI / certificate validation.
func BuildSafeClient(target *FetchTarget, timeout time.Duration) *http.Client {
	pinnedAddr := net.JoinHostPort(target.IPs[0].String(), target.Port)
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			// Ignore the addr from upper layers; always dial the validated IP.
			return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, network, pinnedAddr)
		},
		TLSClientConfig: &tls.Config{
			ServerName: target.Host, // SNI = original host so cert validation works
		},
		DisableKeepAlives: true,
		MaxIdleConns:      1,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
		// Reject redirects — they could point to a different (malicious) host.
		// Caller can paste the final URL after following redirects manually.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// IsPrivate returns true for any IP that should be blocked by default:
// loopback, RFC1918 private, link-local, multicast, unspecified.
func IsPrivate(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified()
}

// isCloudMetadata returns true for the well-known cloud-provider metadata IP.
// AWS / GCP / Azure all expose creds at 169.254.169.254 — block always.
func isCloudMetadata(ip net.IP) bool {
	return ip.Equal(net.IPv4(169, 254, 169, 254))
}
