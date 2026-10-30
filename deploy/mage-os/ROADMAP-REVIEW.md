# Roadmap review tool

Internal tool for the roadmap team at `/review` on the ideas board. Part of the Mage-OS Fider fork.

## Access

| Who | Access |
|---|---|
| Collaborators and administrators | Full use: candidates, assessments, meetings, decisions, actions, export. "Roadmap review" appears in the header. |
| Everyone else | None. `/review` and `/api/v1/mageos/review/*` return 401/403. |

Grant access in **Admin → Members** by making someone a Collaborator. There are no separate review roles.

## Setup

None beyond deploying the fork. Migration `202610301100_mageos_review.sql` runs at startup. No environment variables, no credentials, no external services.

## Data ownership

| Data | Source of truth | Stored in |
|---|---|---|
| Idea title, description, votes, categories, public status | Ideas board (`posts`, votes, tags) | Read live; never copied |
| Review card fields, track, review stage, delivery state | Review tool | `mageos_review_candidates` |
| Ratings and notes per team member | Review tool | `mageos_review_assessments` |
| Meetings, agenda, decisions, finalizer | Review tool | `mageos_review_meetings`, `mageos_review_items` |
| Follow-up actions | Review tool | `mageos_review_actions` |
| Change history (who, what, when) | Review tool | `mageos_review_events` |

Manual candidates (initiatives not on the board) have no linked idea and no votes.

## Rules built in

- One candidate per idea. Adding an idea twice opens the existing candidate.
- Ratings are strong / moderate / weak / unknown per criterion, with a note. Value and delivery readiness are shown separately. No combined score, ranking or automatic approval.
- Effort "Unknown" counts as answered. Missing card fields are listed, never scored.
- Role breakdowns are self-reported and may overlap. Votes are not weighted.
- A decision needs an outcome and a rationale to be finalized. Final decisions are read-only until someone reopens them; both actions are recorded with the person's name.
- No threshold, quorum or approval authority is enforced. Any team member can finalize; record agreed policy in the meeting purpose or rationale until one exists.
- "Prioritize" is separate from delivery state (not scheduled / scheduled / in development / delivered).
- The public status on the board changes only through the confirmed "Update the public status on the board" panel. Its text starts empty; internal notes, ratings and drafts are never copied.
- Sample data is created only by "Add sample candidates for practice", is labelled everywhere, and is removed with "Remove all sample data".

## Meeting flow

1. **Candidates**: add ideas from the board or enter initiatives manually. Fill in cards.
2. **Meetings**: create a meeting. On **Candidates**, select rows and "Add to meeting shortlist".
3. Before the meeting, each member rates candidates on the candidate page ("Your assessment").
4. During the meeting, open the meeting. Use "Compare shortlist" for the side-by-side view. Per candidate: record notes, outcome, rationale, objections, unresolved questions, escalation, and actions; then "Finalize decision".
5. "Copy summary for Google Docs" copies decisions, rationale, owners, next steps and unresolved questions. Discussion notes and ratings are left out.
6. **Follow-ups** lists overdue and upcoming reviews, unassigned actions, and candidates missing information.

GitHub links are entered manually in the card's "Implementation links" field.

## Backups

Review data is in the Fider database, so the nightly `backup.sh` dump covers it. Restore = restore the database. Verify a restore includes the `mageos_review_*` tables.
