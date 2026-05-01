package middleware

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/moon-panel/moon-panel/internal/security"
)

// LoginLockout protects /api/auth/login from brute-force enumeration.
// Phase 3d-2 introduced the basic IP-based lockout. Phase 4a adds:
//   1. **Trusted IP bypass** — IPs in the configured CIDR list skip all
//      lockout logic. Failures from a trusted IP are still audited but
//      never count toward the lockout. Targets the home-network DoS that
//      bit the user in 3d-2 testing (attacker on same NAT as user).
//   2. **Soft-lock fallback** — once locked, a CORRECT password subtracts
//      a fixed amount from remaining lockout. A real user who knows their
//      password can recover; an attacker who doesn't will keep extending.
//      Each WRONG attempt during lockout extends the lock further.
//
// State machine per IP (when not trusted):
//   - 0 failures: requests pass; on success bucket is cleared, on failure
//     bucket starts counting.
//   - 1 to (Threshold-1) failures within FailWindow: still pass.
//   - Threshold failures within FailWindow: locked.
//   - Locked + wrong attempt: extend by SoftLockExtend.
//   - Locked + right attempt: subtract SoftUnlockSubtract; if remaining
//     drops to ≤ 0, fully unlock and let the caller proceed.
//
// Why per-IP and not per-username:
//   - Per-username lets an attacker DoS the legitimate user by spamming
//     wrong passwords for their username.
//   - Per-IP lets a determined attacker rotate IPs to evade, but rotating
//     IPs is a much higher bar than rotating passwords. For a single-user
//     personal panel, this is the right adversary boundary.
//
// State is in-memory; restart clears all locks. Acceptable trade-off
// (admin restarts are infrequent; persisting introduces write pressure on
// the hot login path).
type LoginLockout struct {
	mu                 sync.Mutex
	threshold          int
	failWindow         time.Duration
	lockDuration       time.Duration
	softLockExtend     time.Duration
	softUnlockSubtract time.Duration
	bucks              map[string]*lockoutBucket

	// trusted is hot-reloadable so admin updates to security.trusted_ips
	// take effect immediately without a container restart.
	trusted atomic.Pointer[security.TrustedIPMatcher]
}

type lockoutBucket struct {
	failures    int
	firstFailAt time.Time
	lockedUntil time.Time
}

// NewLoginLockout returns a lockout enforcer with the given thresholds.
// softLockExtend is added to lockedUntil on each wrong attempt during a
// lock; softUnlockSubtract is subtracted on each correct attempt. Both
// default to 5 minutes if zero is passed.
func NewLoginLockout(threshold int, failWindow, lockDuration time.Duration) *LoginLockout {
	return &LoginLockout{
		threshold:          threshold,
		failWindow:         failWindow,
		lockDuration:       lockDuration,
		softLockExtend:     5 * time.Minute,
		softUnlockSubtract: 5 * time.Minute,
		bucks:              make(map[string]*lockoutBucket),
	}
}

// SetTrustedMatcher swaps in a new trusted-IP matcher atomically. Safe to
// call from any goroutine; subsequent IsTrusted calls see the new matcher.
// Pass nil to clear the trusted list.
func (l *LoginLockout) SetTrustedMatcher(m *security.TrustedIPMatcher) {
	l.trusted.Store(m)
}

// IsTrusted reports whether the IP is in the trusted matcher.
func (l *LoginLockout) IsTrusted(ip string) bool {
	m := l.trusted.Load()
	if m == nil {
		return false
	}
	return m.Contains(ip)
}

// Middleware rejects requests from currently-locked, non-trusted IPs with
// 429. Trusted IPs always pass. Locked-but-correct-password recovery is
// NOT handled here — the auth handler must call RecordSuccess() so the
// soft-unlock logic can subtract from the lock. So this middleware is
// only useful for the TOTP verify endpoint where soft-unlock isn't enabled.
//
// For the password-login endpoint, callers should use the explicit Status
// + RecordFailure / RecordSuccess flow inside the handler so soft-unlock
// works correctly. See api/auth.go login() for the pattern.
func (l *LoginLockout) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if l.IsTrusted(ip) {
			c.Next()
			return
		}
		retryAfter, locked := l.checkLocked(ip)
		if locked {
			c.Header("Retry-After", retryAfter.String())
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code": 429,
				"msg":  "too many failed login attempts; locked for " + retryAfter.Truncate(time.Second).String(),
			})
			return
		}
		c.Next()
	}
}

// Status returns whether the IP is currently locked and how much time
// remains. Used by the password-login handler to decide whether to apply
// soft-unlock semantics.
func (l *LoginLockout) Status(ip string) (locked bool, remaining time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.evictStale()
	b, ok := l.bucks[ip]
	if !ok {
		return false, 0
	}
	now := time.Now()
	if now.Before(b.lockedUntil) {
		return true, time.Until(b.lockedUntil)
	}
	return false, 0
}

// RecordFailure increments the failure counter (and may transition to
// locked) when the IP is NOT yet locked. If the IP IS already locked,
// extends the lock by softLockExtend. Trusted IPs are no-ops — the caller
// is expected to check IsTrusted() before calling, but we also short-
// circuit here as defense-in-depth.
func (l *LoginLockout) RecordFailure(ip string) {
	if l.IsTrusted(ip) {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.evictStale()
	now := time.Now()

	b, ok := l.bucks[ip]
	if !ok {
		l.bucks[ip] = &lockoutBucket{failures: 1, firstFailAt: now}
		return
	}
	// Already inside an active lock → extend, don't reset.
	if now.Before(b.lockedUntil) {
		b.lockedUntil = b.lockedUntil.Add(l.softLockExtend)
		return
	}
	// Window expired since the first failure → reset counter.
	if now.Sub(b.firstFailAt) > l.failWindow {
		b.failures = 1
		b.firstFailAt = now
		return
	}
	b.failures++
	if b.failures >= l.threshold {
		b.lockedUntil = now.Add(l.lockDuration)
	}
}

// RecordSuccess clears the failure bucket for an IP that successfully
// authenticated AND was not locked. For the soft-unlock path during a
// lock, use TrySoftUnlock instead.
func (l *LoginLockout) RecordSuccess(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.bucks, ip)
}

// TrySoftUnlock applies the soft-lock fallback: a correct authentication
// attempt during an active lock subtracts softUnlockSubtract from the
// remaining lockout time. Returns:
//
//	fullyUnlocked: true if the lock has been fully cleared (caller should
//	  proceed to issue session); false if the lock still has time on it
//	  (caller should reject with 429 + new remaining time).
//	remaining: post-subtract remaining lock time.
//
// Trusted IPs and unlocked IPs are no-ops returning (true, 0) — the
// caller treats them the same as a normal success.
func (l *LoginLockout) TrySoftUnlock(ip string) (fullyUnlocked bool, remaining time.Duration) {
	if l.IsTrusted(ip) {
		return true, 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	b, ok := l.bucks[ip]
	if !ok {
		return true, 0
	}
	now := time.Now()
	if now.After(b.lockedUntil) {
		// Not actually locked. Treat as full success — clear the bucket.
		delete(l.bucks, ip)
		return true, 0
	}
	b.lockedUntil = b.lockedUntil.Add(-l.softUnlockSubtract)
	if !now.Before(b.lockedUntil) {
		// Subtract dropped remaining ≤ 0 → fully unlock.
		delete(l.bucks, ip)
		return true, 0
	}
	return false, time.Until(b.lockedUntil)
}

// ManualUnlock removes the bucket for ip — used by the admin "立即解锁"
// button. Returns whether anything was actually unlocked.
func (l *LoginLockout) ManualUnlock(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.bucks[ip]; ok {
		delete(l.bucks, ip)
		return true
	}
	return false
}

// LockedSnapshot is a read-only view of one currently-locked IP. Used by
// the admin /admin/security page to render the lockout dashboard.
type LockedSnapshot struct {
	IP          string        `json:"ip"`
	Failures    int           `json:"failures"`
	LockedUntil time.Time     `json:"locked_until"`
	Remaining   time.Duration `json:"remaining_seconds"` // serialized as int seconds via custom marshal? simpler: ms
}

// SnapshotLocked returns the list of currently-locked IPs (lockedUntil in
// the future). Trusted IPs that aren't locked aren't included; admins
// configure the trusted list separately.
func (l *LoginLockout) SnapshotLocked() []LockedSnapshot {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	out := make([]LockedSnapshot, 0)
	for ip, b := range l.bucks {
		if !now.Before(b.lockedUntil) {
			continue
		}
		out = append(out, LockedSnapshot{
			IP:          ip,
			Failures:    b.failures,
			LockedUntil: b.lockedUntil,
			Remaining:   time.Until(b.lockedUntil),
		})
	}
	return out
}

func (l *LoginLockout) checkLocked(ip string) (time.Duration, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.evictStale()
	b, ok := l.bucks[ip]
	if !ok {
		return 0, false
	}
	now := time.Now()
	if now.Before(b.lockedUntil) {
		return time.Until(b.lockedUntil), true
	}
	return 0, false
}

// evictStale drops buckets whose lock has expired AND whose failure window
// has long passed. Mirrors the opportunistic cleanup from ratelimit.go.
func (l *LoginLockout) evictStale() {
	now := time.Now()
	for k, b := range l.bucks {
		lockExpired := b.lockedUntil.IsZero() || now.After(b.lockedUntil)
		windowExpired := now.Sub(b.firstFailAt) > 5*l.failWindow
		if lockExpired && windowExpired {
			delete(l.bucks, k)
		}
	}
}
