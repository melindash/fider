package mageos

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Link kinds. Discussions are linked for context only; issues and pull requests track implementation.
const (
	LinkDiscussion  = "discussion"
	LinkIssue       = "issue"
	LinkPullRequest = "pull_request"
)

// GitHubTarget is what a GitHub URL points at.
type GitHubTarget struct {
	Kind   string
	Repo   string // "owner/repo", or "org" for organisation discussions
	Number int
	URL    string
}

var githubPath = regexp.MustCompile(`^/(?:orgs/([A-Za-z0-9-]+)/discussions|([A-Za-z0-9-]+/[A-Za-z0-9._-]+)/(issues|pull|discussions))/([0-9]+)/?$`)

// ParseGitHubURL accepts only https://github.com links to a discussion, issue or pull request.
func ParseGitHubURL(raw string) (*GitHubTarget, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.Host != "github.com" {
		return nil, false
	}
	m := githubPath.FindStringSubmatch(u.Path)
	if m == nil {
		return nil, false
	}
	number, _ := strconv.Atoi(m[4])
	target := &GitHubTarget{Number: number}
	switch {
	case m[1] != "":
		target.Kind, target.Repo = LinkDiscussion, strings.ToLower(m[1])
	case m[3] == "issues":
		target.Kind, target.Repo = LinkIssue, strings.ToLower(m[2])
	case m[3] == "pull":
		target.Kind, target.Repo = LinkPullRequest, strings.ToLower(m[2])
	default:
		target.Kind, target.Repo = LinkDiscussion, strings.ToLower(m[2])
	}
	target.URL = "https://github.com" + strings.TrimSuffix(u.Path, "/")
	return target, true
}

// GitHubWebhookSecret is empty when the webhook receiver is disabled.
func GitHubWebhookSecret() string {
	return os.Getenv("MAGEOS_GITHUB_WEBHOOK_SECRET")
}

// GitHubAutoStatus allows a linked issue or pull request opening to move a Planned idea to Started.
func GitHubAutoStatus() bool {
	return os.Getenv("MAGEOS_GITHUB_AUTO_STATUS") == "true"
}

// ValidGitHubSignature checks X-Hub-Signature-256 in constant time.
func ValidGitHubSignature(secret string, body []byte, header string) bool {
	if secret == "" || !strings.HasPrefix(header, "sha256=") {
		return false
	}
	given, err := hex.DecodeString(strings.TrimPrefix(header, "sha256="))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hmac.Equal(given, mac.Sum(nil))
}

// LinkStateForEvent maps a GitHub issues/pull_request action to the stored link state.
// It returns "" for actions that do not change state.
//
// A merged pull request or closed issue is recorded but never marks an idea Completed:
// merged code is not released code, and maintainers decide when an idea has shipped.
func LinkStateForEvent(event, action string, merged bool) string {
	switch action {
	case "opened", "reopened":
		return "open"
	case "closed":
		if event == "pull_request" && merged {
			return "merged"
		}
		return "closed"
	}
	return ""
}

// StartsWork reports whether a state change may move a Planned idea to Started.
func StartsWork(kind, state string) bool {
	return state == "open" && (kind == LinkIssue || kind == LinkPullRequest)
}
