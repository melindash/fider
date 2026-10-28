package mageos

import (
	"testing"
	"time"

	. "github.com/getfider/fider/app/pkg/assert"
)

func TestNetworkPrefix(t *testing.T) {
	RegisterT(t)

	Expect(NetworkPrefix("203.0.113.77")).Equals("203.0.113.0/24")
	Expect(NetworkPrefix("2001:db8:abcd:12::1")).Equals("2001:db8:abcd::/48")
	Expect(NetworkPrefix("not-an-ip")).Equals("unknown")
}

func TestValidRoles(t *testing.T) {
	RegisterT(t)

	Expect(ValidRoles([]string{})).IsTrue()
	Expect(ValidRoles([]string{"merchant", "developer"})).IsTrue()
	Expect(ValidRoles([]string{"developer", "developer"})).IsFalse()
	Expect(ValidRoles([]string{"admin"})).IsFalse()
}

func TestHashTokenIsStableAndNotTheToken(t *testing.T) {
	RegisterT(t)

	token := NewVoterToken()
	Expect(HashToken(token)).Equals(HashToken(token))
	Expect(HashToken(token)).NotEquals(token)
	Expect(len(HashToken(token))).Equals(64)
	Expect(NewVoterToken()).NotEquals(token)
}

func TestClientIPIgnoresHeadersUnlessConfigured(t *testing.T) {
	RegisterT(t)

	headers := func(name string) string {
		if name == "CF-Connecting-IP" {
			return "198.51.100.9"
		}
		return ""
	}

	t.Setenv("MAGEOS_CLIENT_IP_HEADER", "")
	Expect(ClientIP(headers, "172.18.0.5:41234")).Equals("172.18.0.5")

	t.Setenv("MAGEOS_CLIENT_IP_HEADER", "CF-Connecting-IP")
	Expect(ClientIP(headers, "172.18.0.5:41234")).Equals("198.51.100.9")
}

func TestLimiterWindow(t *testing.T) {
	RegisterT(t)

	now := time.Date(2026, 10, 28, 10, 0, 0, 0, time.UTC)
	limiter := NewLimiter()
	limiter.now = func() time.Time { return now }

	Expect(limiter.Allow("k", 2, time.Hour)).IsTrue()
	Expect(limiter.Allow("k", 2, time.Hour)).IsTrue()
	Expect(limiter.Allow("k", 2, time.Hour)).IsFalse()
	Expect(limiter.Allow("other", 2, time.Hour)).IsTrue()

	now = now.Add(time.Hour)
	Expect(limiter.Allow("k", 2, time.Hour)).IsTrue()
}
