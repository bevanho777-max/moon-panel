// Package totp wraps github.com/pquerna/otp with project-specific defaults.
// Phase 3d-3.
//
// Why a wrapper rather than calling pquerna/otp directly:
//   - Backup-code generation, hashing, and verification live alongside TOTP
//     (same security domain). The wrapper bundles them so handlers don't
//     reimplement the bcrypt-each-code logic.
//   - Centralizes the issuer name ("Moon Panel") and account prefix; future
//     rebrand is a single-file change.
//   - Hides the otp.Key vs otp.Validate split — handlers see ergonomic
//     functions like NewSecret / Verify(secret, code).
package totp

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

const (
	// Issuer is the label shown in user authenticator apps.
	Issuer = "Moon Panel"

	// BackupCodeCount: 8 codes, balance between user-recoverability and
	// management burden (Google = 8, GitHub = 16). 8 is enough headroom for
	// "phone broke during a trip + lost one code in scramble".
	BackupCodeCount = 8

	// Period / Digits / Algorithm are RFC 6238 defaults (the standard
	// every authenticator app expects). Don't change without a migration
	// plan: changing these silently invalidates every existing enrollment.
	Period    = 30
	Digits    = otp.DigitsSix
	Algorithm = otp.AlgorithmSHA1
)

// Enrollment is the result of generating a fresh TOTP enrollment.
// Returned ONCE — the secret is not stored after the user confirms.
// Backup codes are returned plaintext one time; their bcrypt hashes are
// what gets persisted on confirm.
type Enrollment struct {
	Secret      string   // base32-encoded, persisted as-is on confirm
	OTPAuthURL  string   // otpauth://totp/... URI for QR-code generation
	BackupCodes []string // plaintext, shown to user only once
}

// NewEnrollment generates a fresh secret + provisioning URI + backup codes.
// Caller is responsible for persisting (Secret, hashed BackupCodes) only
// AFTER the user verifies their first TOTP code (via Verify), so a partial
// enrollment doesn't accidentally activate.
func NewEnrollment(accountName string) (*Enrollment, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      Issuer,
		AccountName: accountName,
		Period:      Period,
		Digits:      Digits,
		Algorithm:   Algorithm,
	})
	if err != nil {
		return nil, fmt.Errorf("generate totp key: %w", err)
	}
	codes, err := generateBackupCodes(BackupCodeCount)
	if err != nil {
		return nil, fmt.Errorf("generate backup codes: %w", err)
	}
	return &Enrollment{
		Secret:      key.Secret(),
		OTPAuthURL:  key.URL(),
		BackupCodes: codes,
	}, nil
}

// Verify checks if the given 6-digit code matches the secret at the current
// time. Allows a ±1 period drift (default ±30s) to handle clock skew between
// server and user's phone — pquerna/otp's totp.Validate handles this when we
// pass Skew: 1.
func Verify(secret, code string) bool {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return false
	}
	valid, _ := totp.ValidateCustom(code, secret, time.Now(), totp.ValidateOpts{
		Period:    Period,
		Skew:      1, // accept previous + current + next code
		Digits:    Digits,
		Algorithm: Algorithm,
	})
	return valid
}

// HashBackupCodes bcrypts each plaintext backup code and returns a JSON
// array of hashes suitable for persisting in users.totp_backup_codes.
func HashBackupCodes(plain []string) (string, error) {
	hashes := make([]string, len(plain))
	for i, c := range plain {
		h, err := bcrypt.GenerateFromPassword([]byte(c), bcrypt.DefaultCost)
		if err != nil {
			return "", err
		}
		hashes[i] = string(h)
	}
	out, err := json.Marshal(hashes)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// ConsumeBackupCode checks if `attempt` matches any of the stored hashed
// backup codes. On match, returns the updated JSON (with the matched hash
// removed) so the caller can persist single-use semantics. On no match,
// returns ("", false). Constant-time mismatch over the full list — no
// timing oracle for "code 3 was right vs code 7 was right".
func ConsumeBackupCode(storedHashesJSON, attempt string) (newJSON string, matched bool) {
	attempt = strings.TrimSpace(attempt)
	// Permissive normalization: strip dashes/spaces so "ABCD-1234" works for
	// users who type the displayed format.
	attempt = strings.ReplaceAll(attempt, "-", "")
	attempt = strings.ReplaceAll(attempt, " ", "")
	if len(attempt) < 4 {
		return "", false
	}

	var hashes []string
	if err := json.Unmarshal([]byte(storedHashesJSON), &hashes); err != nil {
		return "", false
	}

	matchedIdx := -1
	for i, h := range hashes {
		if bcrypt.CompareHashAndPassword([]byte(h), []byte(attempt)) == nil {
			matchedIdx = i
			// Don't break — finish the loop to drain the timing budget.
			// The cost is N bcrypt calls per backup code use; with
			// N=8 and bcrypt cost 10, it's ~800ms worst case. Acceptable
			// because backup code use is rare.
		}
	}
	if matchedIdx < 0 {
		return "", false
	}
	// Remove the matched code so it can't be reused.
	hashes = append(hashes[:matchedIdx], hashes[matchedIdx+1:]...)
	out, err := json.Marshal(hashes)
	if err != nil {
		return "", false
	}
	return string(out), true
}

// generateBackupCodes returns count plaintext backup codes formatted as
// XXXX-XXXX. Each block is 4 base32 characters drawn from crypto/rand.
// 8 base32 chars = 40 bits ≈ 1 trillion combinations per code. Even with
// the bcrypt comparison loop on every attempt, an attacker cannot brute
// force a code in any reasonable timeframe.
func generateBackupCodes(count int) ([]string, error) {
	codes := make([]string, count)
	enc := base32.StdEncoding.WithPadding(base32.NoPadding)
	for i := range codes {
		// 5 bytes → exactly 8 base32 characters, no padding.
		buf := make([]byte, 5)
		if _, err := rand.Read(buf); err != nil {
			return nil, err
		}
		s := enc.EncodeToString(buf)
		codes[i] = s[:4] + "-" + s[4:8]
	}
	return codes, nil
}
