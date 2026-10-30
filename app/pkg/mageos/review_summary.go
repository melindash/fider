package mageos

import (
	"fmt"
	"html"
	"strings"

	"github.com/getfider/fider/app/models/entity"
)

var reviewOutcomeLabels = map[string]string{
	"needs_clarification": "Needs clarification",
	"seek_comments":       "Seek community comments",
	"investigate":         "Investigate / scope",
	"prioritize":          "Prioritize",
	"defer":               "Defer",
	"decline":             "Decline",
}

var reviewEscalationLabels = map[string]string{
	"technical_team": "Technical team input requested",
	"board_liaison":  "Escalated to the Board liaison",
	"board":          "Escalated to the Board",
}

var reviewDeliveryLabels = map[string]string{
	"none":           "not scheduled",
	"scheduled":      "scheduled",
	"in_development": "in development",
	"delivered":      "delivered",
}

type summaryLine struct{ label, value string }

// ReviewMeetingSummary renders a meeting's decisions as HTML (for pasting into Google Docs) and plain text.
// Discussion notes and individual assessments are left out on purpose: the summary is shared more widely.
func ReviewMeetingSummary(baseURL string, meeting *entity.MageOSReviewMeeting, items []*entity.MageOSReviewItem) (string, string) {
	var h, t strings.Builder
	title := "Roadmap meeting " + meeting.HeldOn
	if meeting.IsSample {
		title += " (SAMPLE DATA, not real decisions)"
	}
	fmt.Fprintf(&h, "<h2>%s</h2>\n", html.EscapeString(title))
	fmt.Fprintf(&t, "%s\n", title)
	if meeting.Purpose != "" {
		fmt.Fprintf(&h, "<p><b>Purpose:</b> %s</p>\n", html.EscapeString(meeting.Purpose))
		fmt.Fprintf(&t, "Purpose: %s\n", meeting.Purpose)
	}
	final := 0
	for _, item := range items {
		if item.FinalizedAt != "" {
			final++
		}
	}
	fmt.Fprintf(&h, "<p>%d of %d decisions final.</p>\n", final, len(items))
	fmt.Fprintf(&t, "%d of %d decisions final.\n", final, len(items))

	for _, item := range items {
		c := item.Candidate
		if c == nil {
			continue
		}
		name := c.Title
		if c.IsSample {
			name += " (sample)"
		}
		fmt.Fprintf(&h, "<h3>%s</h3>\n<ul>\n", html.EscapeString(name))
		fmt.Fprintf(&t, "\n%s\n", name)

		outcome := "No outcome yet"
		if label, ok := reviewOutcomeLabels[item.Outcome]; ok {
			outcome = label
			if item.Outcome == "prioritize" {
				outcome += " (" + reviewDeliveryLabels[c.Delivery] + ")"
			}
		}
		if item.FinalizedAt != "" {
			outcome += ". Final, recorded by " + item.FinalizedByName
		} else {
			outcome += ". DRAFT, not final"
		}

		lines := []summaryLine{{"Decision requested", item.DecisionRequested}, {"Outcome", outcome}, {"Rationale", item.Rationale}}
		if c.Idea != nil {
			lines = append(lines, summaryLine{"Idea", fmt.Sprintf("%s/posts/%d/%s (%d votes)", baseURL, c.Idea.Number, c.Idea.Slug, c.Idea.Votes)})
		} else if c.SourceURL != "" {
			lines = append(lines, summaryLine{"Source", c.SourceURL})
		}
		lines = append(lines,
			summaryLine{"Objections", item.Objections},
			summaryLine{"Unresolved questions", item.Unresolved},
			summaryLine{"Escalation", reviewEscalationLabels[item.Escalation]},
		)
		for _, line := range lines {
			if strings.TrimSpace(line.value) == "" {
				continue
			}
			fmt.Fprintf(&h, "<li><b>%s:</b> %s</li>\n", line.label, html.EscapeString(line.value))
			fmt.Fprintf(&t, "- %s: %s\n", line.label, line.value)
		}

		open := []*entity.MageOSReviewAction{}
		for _, a := range item.Actions {
			if !a.IsDone {
				open = append(open, a)
			}
		}
		if len(open) > 0 {
			fmt.Fprint(&h, "<li><b>Next steps:</b><ul>\n")
			fmt.Fprint(&t, "- Next steps:\n")
			for _, a := range open {
				owner := a.Owner
				if owner == "" {
					owner = "Unassigned"
				}
				step := owner + ": " + a.NextStep
				if a.ReviewOn != "" {
					step += " (review " + a.ReviewOn + ")"
				}
				fmt.Fprintf(&h, "<li>%s</li>\n", html.EscapeString(step))
				fmt.Fprintf(&t, "  - %s\n", step)
			}
			fmt.Fprint(&h, "</ul></li>\n")
		}
		fmt.Fprint(&h, "</ul>\n")
	}
	return h.String(), t.String()
}
