package security

import (
	"errors"
	"net"
	"strings"
)

// TrustedIPMatcher checks whether a request IP is in the configured trusted
// list. Lookups are O(N) where N is the number of CIDR entries — for the
// expected scale (a handful of home/office networks) this is microseconds.
//
// Construction is fail-fast: invalid CIDR or "match-all" entries are rejected
// at parse time so a misconfigured trusted list can't accidentally bypass
// security on every request.
type TrustedIPMatcher struct {
	nets []*net.IPNet
}

// ParseTrustedCIDRs validates and parses each CIDR string. Returns an error
// for any malformed entry or for "match-all" CIDRs (0.0.0.0/0 or ::/0) —
// these would defeat the purpose of a trusted list and we refuse to accept
// them at the API layer rather than silently widening exposure.
func ParseTrustedCIDRs(cidrs []string) (*TrustedIPMatcher, error) {
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, s := range cidrs {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		_, ipNet, err := net.ParseCIDR(s)
		if err != nil {
			return nil, errors.New("invalid CIDR: " + s)
		}
		ones, bits := ipNet.Mask.Size()
		if ones == 0 && bits > 0 {
			// /0 means "all of v4" or "all of v6" — we refuse to accept this.
			return nil, errors.New("CIDR " + s + " covers the entire address space; refusing to whitelist")
		}
		nets = append(nets, ipNet)
	}
	return &TrustedIPMatcher{nets: nets}, nil
}

// Contains reports whether ipStr falls within any of the configured ranges.
// Empty matchers (no entries) always return false — the typical default.
// Invalid IPs return false (non-trusted). Used by login/2fa lockout to
// decide whether to skip the failure-counting path.
func (m *TrustedIPMatcher) Contains(ipStr string) bool {
	if m == nil || len(m.nets) == 0 {
		return false
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, n := range m.nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// Size returns the number of configured entries — useful for logging the
// active matcher size at boot.
func (m *TrustedIPMatcher) Size() int {
	if m == nil {
		return 0
	}
	return len(m.nets)
}
