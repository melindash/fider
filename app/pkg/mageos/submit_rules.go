package mageos

import (
	"os"
	"strings"
)

// SubmitRules turns on the Development Priorities Rules v1 submit form: a description and a focus area are required.
func SubmitRules() bool {
	return os.Getenv("MAGEOS_SUBMIT_RULES") == "true"
}

// DescriptionIsBlank also treats an untouched description template as blank, because the form prefills it.
func DescriptionIsBlank(description, template string) bool {
	d := strings.TrimSpace(description)
	return d == "" || (strings.TrimSpace(template) != "" && d == strings.TrimSpace(template))
}
