package mageos

import "slices"

// Review vocabularies. The UI labels them; the server only accepts these values.
var (
	ReviewTracks      = []string{"product", "contributor_experience", "community"}
	ReviewStages      = []string{"new", "preparing", "ready", "decided", "archived"}
	ReviewDelivery    = []string{"none", "scheduled", "in_development", "delivered"}
	ReviewEfforts     = []string{"", "small", "medium", "large", "unknown"}
	ReviewRatings     = []string{"strong", "moderate", "weak", "unknown"}
	ReviewOutcomes    = []string{"", "needs_clarification", "seek_comments", "investigate", "prioritize", "defer", "decline"}
	ReviewEscalations = []string{"none", "technical_team", "board_liaison", "board"}

	// Value and delivery readiness are kept apart and never combined into one score.
	ReviewValueCriteria     = []string{"benefit", "alignment", "demand"}
	ReviewReadinessCriteria = []string{"feasibility", "people", "support"}
)

// ReviewCandidateFields maps editable candidate fields to their columns.
var ReviewCandidateFields = map[string]string{
	"title": "title", "sourceUrl": "source_url", "track": "track", "stage": "stage", "delivery": "delivery",
	"problem": "problem", "beneficiaries": "beneficiaries", "outcome": "outcome", "scope": "scope",
	"exclusions": "exclusions", "evidence": "evidence", "dependencies": "dependencies", "risks": "risks",
	"effort": "effort", "owner": "owner", "reviewer": "reviewer", "operatingNeeds": "operating_needs",
	"openQuestions": "open_questions", "links": "links",
}

// ReviewItemFields maps editable meeting item fields to their columns.
var ReviewItemFields = map[string]string{
	"decisionRequested": "decision_requested", "notes": "notes", "outcome": "outcome", "rationale": "rationale",
	"objections": "objections", "unresolved": "unresolved", "escalation": "escalation",
}

// ReviewRequiredFields are the card fields a candidate needs before it is ready for a decision.
// Effort "unknown" counts as answered: unknown is information, not a gap.
var ReviewRequiredFields = []string{"problem", "beneficiaries", "outcome", "scope", "evidence", "effort", "owner", "reviewer", "operatingNeeds"}

// ValidReviewValue reports whether value is allowed for an enumerated field. Free-text fields always pass.
func ValidReviewValue(field, value string) bool {
	switch field {
	case "track":
		return slices.Contains(ReviewTracks, value)
	case "stage":
		return slices.Contains(ReviewStages, value)
	case "delivery":
		return slices.Contains(ReviewDelivery, value)
	case "effort":
		return slices.Contains(ReviewEfforts, value)
	case "outcome":
		return slices.Contains(ReviewOutcomes, value)
	case "escalation":
		return slices.Contains(ReviewEscalations, value)
	}
	return true
}

// ValidReviewCriterion reports whether criterion is one of the comparison criteria.
func ValidReviewCriterion(criterion string) bool {
	return slices.Contains(ReviewValueCriteria, criterion) || slices.Contains(ReviewReadinessCriteria, criterion)
}

// ReviewMissingFields lists required card fields that are still empty.
func ReviewMissingFields(values map[string]string) []string {
	missing := []string{}
	for _, field := range ReviewRequiredFields {
		if values[field] == "" {
			missing = append(missing, field)
		}
	}
	return missing
}
