// Mage-OS fork: display labels for the review vocabulary. The allowed values come from the server.
import { PostStatus } from "@fider/models"

export const trackLabels: Record<string, string> = {
  product: "Product",
  contributor_experience: "Contributor experience",
  community: "Community",
}

export const stageLabels: Record<string, string> = {
  new: "New",
  preparing: "Preparing",
  ready: "Ready for decision",
  decided: "Decided",
  archived: "Archived",
}

export const deliveryLabels: Record<string, string> = {
  none: "Not scheduled",
  scheduled: "Scheduled",
  in_development: "In development",
  delivered: "Delivered",
}

export const effortLabels: Record<string, string> = {
  "": "Not set",
  small: "Small (up to 3 person-days)",
  medium: "Medium (4 to 10 person-days)",
  large: "Large (over 10 person-days, consider splitting)",
  unknown: "Unknown",
}

export const ratingLabels: Record<string, string> = {
  strong: "Strong",
  moderate: "Moderate",
  weak: "Weak",
  unknown: "Unknown",
}

export const criterionLabels: Record<string, { title: string; hint: string }> = {
  benefit: { title: "User / community benefit", hint: "How much it helps the people affected." },
  alignment: { title: "Fits the focus area", hint: "Fit with the focus areas and what the project has already agreed to prioritize." },
  demand: { title: "Evidence of demand", hint: "Votes, discussions, support requests. Votes are one input." },
  feasibility: { title: "Effort and feasibility", hint: "Strong means realistic to deliver." },
  people: { title: "Volunteers and Tech Team review available", hint: "Strong means people have committed." },
  support: { title: "Dependencies and ongoing support", hint: "Strong means few blockers and a clear maintainer." },
}

export const outcomeLabels: Record<string, string> = {
  "": "No outcome yet",
  needs_clarification: "Not decided: needs clarification",
  seek_comments: "Not decided: seek community comments",
  investigate: "Not decided: investigate / scope",
  prioritize: "Accept (prioritize)",
  defer: "Park (defer)",
  decline: "Decline",
}

export const escalationLabels: Record<string, string> = {
  none: "No escalation",
  technical_team: "Request technical team input",
  board_liaison: "Escalate to the Board liaison",
  board: "Escalate to the Board",
}

export const roleLabels: Record<string, string> = {
  merchant: "Merchant / e-commerce manager",
  developer: "Developer",
  agency: "Agency",
  marketing: "Marketing",
  other: "Other",
}

export const fieldLabels: Record<string, string> = {
  problem: "Problem",
  beneficiaries: "Who benefits",
  outcome: "Expected outcome",
  scope: "Scope",
  exclusions: "Out of scope",
  evidence: "Evidence and sources",
  dependencies: "Dependencies",
  risks: "Risks",
  effort: "Effort",
  owner: "Task owner",
  reviewer: "Tech Team contact",
  operatingNeeds: "Ongoing maintenance / operating needs",
  openQuestions: "Open questions",
  links: "Implementation links (GitHub issues, discussions)",
  title: "Title",
  sourceUrl: "Source link",
  track: "Track",
  stage: "Review stage",
  delivery: "Delivery",
  decisionRequested: "Decision requested",
  notes: "Discussion notes",
  rationale: "Rationale",
  objections: "Objections",
  unresolved: "Unresolved questions",
  escalation: "Escalation",
}

// Headings follow the acceptance tests in Development Priorities Rules v1; the two server-side criteria groups stay as they are.
export const valueGroupTitle = "Clear value and fits the focus area"
export const readinessGroupTitle = "Tech Team can support it"

export const label = (map: Record<string, string>, value: string): string => map[value] ?? value

export const ideaStatusLabel = (value: string): string => PostStatus.All.find((s) => s.value === value)?.title ?? value
