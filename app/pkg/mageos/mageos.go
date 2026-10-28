// Package mageos holds Mage-OS fork features that have no upstream equivalent:
// anonymous voting, participant roles and their abuse controls.
package mageos

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

// VoterCookieName holds the anonymous voter token. The database stores only its SHA-256.
const VoterCookieName = "mageos_voter"

// VoterCookieLifetime keeps anonymous votes removable for a year from the same browser.
const VoterCookieLifetime = 365 * 24 * time.Hour

// NetworkHashRetention is how long the pseudonymous network hash stays on an anonymous vote.
const NetworkHashRetention = 30 * 24 * time.Hour

// ParticipantRoles are the self-reported, multi-select roles. Order is display order.
var ParticipantRoles = []string{"merchant", "developer", "agency", "marketing", "other"}

// ValidRoles reports whether every role is known and none repeat.
func ValidRoles(roles []string) bool {
	seen := map[string]bool{}
	for _, r := range roles {
		if seen[r] || !slices.Contains(ParticipantRoles, r) {
			return false
		}
		seen[r] = true
	}
	return true
}

// Limits are the anonymous-voting rate limits. Defaults are deliberately generous
// because offices, universities and mobile carriers put many people behind one address.
type Limits struct {
	NewVotersPerNetworkPerHour int
	VotesPerNetworkPerHour     int
	VotesPerVoterPerMinute     int
	// VotesPerNetworkPerPostPerDay is enforced from the database, so it survives restarts.
	VotesPerNetworkPerPostPerDay int
}

// AnonymousVotingEnabled is the kill switch for anonymous voting during abuse.
func AnonymousVotingEnabled() bool {
	return os.Getenv("MAGEOS_ANONYMOUS_VOTING") != "false"
}

// CurrentLimits reads limits from the environment, falling back to defaults.
func CurrentLimits() Limits {
	return Limits{
		NewVotersPerNetworkPerHour:   envInt("MAGEOS_NEW_VOTERS_PER_NETWORK_PER_HOUR", 60),
		VotesPerNetworkPerHour:       envInt("MAGEOS_VOTES_PER_NETWORK_PER_HOUR", 600),
		VotesPerVoterPerMinute:       envInt("MAGEOS_VOTES_PER_VOTER_PER_MINUTE", 30),
		VotesPerNetworkPerPostPerDay: envInt("MAGEOS_VOTES_PER_NETWORK_PER_POST_PER_DAY", 25),
	}
}

func envInt(name string, fallback int) int {
	if v, err := strconv.Atoi(os.Getenv(name)); err == nil && v > 0 {
		return v
	}
	return fallback
}

// NewVoterToken returns a random URL-safe token for the voter cookie.
func NewVoterToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// HashToken is what gets stored, so a database leak cannot be replayed as a cookie.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// ClientIP trusts a forwarding header only when MAGEOS_CLIENT_IP_HEADER names it.
// Set it to CF-Connecting-IP behind Cloudflare Tunnel, where nothing else can reach
// the app; otherwise any client could spoof its address and dodge the limits.
func ClientIP(getHeader func(string) string, remoteAddr string) string {
	if header := os.Getenv("MAGEOS_CLIENT_IP_HEADER"); header != "" {
		value := strings.TrimSpace(strings.Split(getHeader(header), ",")[0])
		if net.ParseIP(value) != nil {
			return value
		}
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}

// NetworkPrefix widens an address to the block one household or office usually shares
// (IPv4 /24, IPv6 /48), so limits apply per network rather than per device.
func NetworkPrefix(ip string) string {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return "unknown"
	}
	if v4 := parsed.To4(); v4 != nil {
		return v4.Mask(net.CIDRMask(24, 32)).String() + "/24"
	}
	return parsed.Mask(net.CIDRMask(48, 128)).String() + "/48"
}

// NetworkHash is a keyed hash of the network prefix. Keying it stops anyone with the
// database from reversing it by hashing every /24.
func NetworkHash(secret, prefix string) string {
	mac := hmac.New(sha256.New, []byte("mageos-network:"+secret))
	mac.Write([]byte(prefix))
	return hex.EncodeToString(mac.Sum(nil))
}
