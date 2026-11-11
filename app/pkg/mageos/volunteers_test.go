package mageos_test

import (
	"testing"

	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/mageos"
)

func TestNormalizeGitHubLogin(t *testing.T) {
	RegisterT(t)

	for input, want := range map[string]string{
		"melindash":                   "melindash",
		" @melindash ":                "melindash",
		"https://github.com/mage-os/": "mage-os",
		"github.com/a-b-c":            "a-b-c",
		"-leading":                    "",
		"double--dash":                "",
		"trailing-":                   "",
		"has space":                   "",
		"":                            "",
		"https://github.com/org/repo": "",
		"a234567890123456789012345678901234567890": "",
	} {
		Expect(mageos.NormalizeGitHubLogin(input)).Equals(want)
	}
}
