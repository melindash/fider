package mageos

import (
	"regexp"
	"strings"
)

var gitHubLogin = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9]|-[A-Za-z0-9]){0,38}$`)

// NormalizeGitHubLogin accepts "name", "@name" or a github.com profile URL and returns the bare login, or "" if it is not a valid GitHub username.
func NormalizeGitHubLogin(value string) string {
	v := strings.TrimSpace(value)
	for _, prefix := range []string{"https://github.com/", "http://github.com/", "github.com/", "@"} {
		v = strings.TrimPrefix(v, prefix)
	}
	v = strings.TrimSuffix(v, "/")
	if len(v) > 39 || !gitHubLogin.MatchString(v) {
		return ""
	}
	return v
}
