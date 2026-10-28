package mageos

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	. "github.com/getfider/fider/app/pkg/assert"
)

func TestParseGitHubURL(t *testing.T) {
	RegisterT(t)

	cases := map[string]GitHubTarget{
		"https://github.com/orgs/mage-os-lab/discussions/12":  {Kind: LinkDiscussion, Repo: "mage-os-lab", Number: 12},
		"https://github.com/mage-os/mageos-magento2/issues/7": {Kind: LinkIssue, Repo: "mage-os/mageos-magento2", Number: 7},
		"https://github.com/Mage-OS/mageos-magento2/pull/356/": {Kind: LinkPullRequest, Repo: "mage-os/mageos-magento2", Number: 356},
		"https://github.com/mage-os/mage-os/discussions/3":    {Kind: LinkDiscussion, Repo: "mage-os/mage-os", Number: 3},
	}
	for raw, want := range cases {
		got, ok := ParseGitHubURL(raw)
		Expect(ok).IsTrue()
		Expect(got.Kind).Equals(want.Kind)
		Expect(got.Repo).Equals(want.Repo)
		Expect(got.Number).Equals(want.Number)
	}

	for _, bad := range []string{
		"http://github.com/mage-os/mage-os/issues/1",
		"https://github.com.evil.example/mage-os/mage-os/issues/1",
		"https://github.com/mage-os/mage-os/issues/abc",
		"https://github.com/mage-os/mage-os",
		"javascript:alert(1)",
	} {
		_, ok := ParseGitHubURL(bad)
		Expect(ok).IsFalse()
	}
}

func TestValidGitHubSignature(t *testing.T) {
	RegisterT(t)

	body := []byte(`{"action":"opened"}`)
	mac := hmac.New(sha256.New, []byte("s3cret"))
	mac.Write(body)
	header := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	Expect(ValidGitHubSignature("s3cret", body, header)).IsTrue()
	Expect(ValidGitHubSignature("wrong", body, header)).IsFalse()
	Expect(ValidGitHubSignature("s3cret", []byte(`{"action":"closed"}`), header)).IsFalse()
	Expect(ValidGitHubSignature("", body, header)).IsFalse()
	Expect(ValidGitHubSignature("s3cret", body, "sha1=abc")).IsFalse()
}

func TestStatusMappingNeverCompletes(t *testing.T) {
	RegisterT(t)

	Expect(LinkStateForEvent("pull_request", "opened", false)).Equals("open")
	Expect(LinkStateForEvent("pull_request", "closed", true)).Equals("merged")
	Expect(LinkStateForEvent("pull_request", "closed", false)).Equals("closed")
	Expect(LinkStateForEvent("issues", "closed", false)).Equals("closed")
	Expect(LinkStateForEvent("issues", "labeled", false)).Equals("")

	Expect(StartsWork(LinkPullRequest, "open")).IsTrue()
	Expect(StartsWork(LinkIssue, "open")).IsTrue()
	Expect(StartsWork(LinkPullRequest, "merged")).IsFalse()
	Expect(StartsWork(LinkDiscussion, "open")).IsFalse()
}
