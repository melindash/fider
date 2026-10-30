import React from "react"
import { useFider } from "@fider/hooks"
import { Assessment, Candidate, Participation, Vocabulary, review } from "./api"
import { Badge, SelectField, TextField } from "./components"
import { criterionLabels, ratingLabels, roleLabels } from "./labels"

// Shows each rating with who gave it. Counts per rating only; ratings are never combined into a score.
export const AssessmentSummary = (props: { criteria: string[]; assessments: Assessment[]; compact?: boolean; bare?: boolean }) => (
  <dl className="r-assess">
    {props.criteria.map((criterion) => {
      const rows = props.assessments.filter((a) => a.criterion === criterion)
      const counts = ["strong", "moderate", "weak", "unknown"].map((r) => [r, rows.filter((a) => a.rating === r).length] as const).filter(([, n]) => n > 0)
      return (
        <div key={criterion} className="r-assess__row">
          <dt className={props.bare ? "sr-only" : ""}>{criterionLabels[criterion]?.title ?? criterion}</dt>
          <dd>
            {rows.length === 0 ? (
              <span className="r-muted">Not assessed</span>
            ) : (
              counts.map(([rating, n]) => (
                <Badge key={rating} tone={rating === "unknown" ? "muted" : "draft"}>
                  {ratingLabels[rating]} × {n}
                </Badge>
              ))
            )}
            {!props.compact && rows.length > 0 && (
              <ul className="r-list r-list--compact">
                {rows.map((a) => (
                  <li key={a.userId}>
                    <strong>{a.userName}</strong>: {ratingLabels[a.rating]}
                    {a.note && <span className="r-muted"> · {a.note}</span>}
                  </li>
                ))}
              </ul>
            )}
          </dd>
        </div>
      )
    })}
  </dl>
)

// The signed-in team member's own ratings, editable before or during a meeting.
export const MyAssessment = (props: { candidateId: number; vocab: Vocabulary; assessments: Assessment[]; onSaved: () => void }) => {
  const fider = useFider()
  const mine = (criterion: string) => props.assessments.find((a) => a.criterion === criterion && a.userId === fider.session.user.id)
  const group = (title: string, criteria: string[]) => (
    <fieldset className="r-myassess">
      <legend>{title}</legend>
      {criteria.map((criterion) => {
        const current = mine(criterion)
        const save = async (rating: string, note: string) => {
          const result = await review.assess(props.candidateId, criterion, rating, note)
          if (result.ok) props.onSaved()
          return result
        }
        return (
          <div key={criterion} className="r-myassess__row">
            <SelectField
              label={criterionLabels[criterion].title}
              value={current?.rating ?? ""}
              options={["", ...props.vocab.ratings]}
              labels={{ "": "Not rated", ...ratingLabels }}
              onSave={(rating) => (rating ? save(rating, current?.note ?? "") : Promise.resolve({ ok: true, data: {} }))}
            />
            <TextField
              label="Why (short)"
              hint={criterionLabels[criterion].hint}
              value={current?.note ?? ""}
              disabled={!current}
              placeholder={current ? "" : "Rate first"}
              onSave={(note) => save(current?.rating ?? "unknown", note)}
            />
          </div>
        )
      })}
    </fieldset>
  )
  return (
    <div className="r-myassess-groups">
      {group("Value", props.vocab.valueCriteria)}
      {group("Delivery readiness", props.vocab.readinessCriteria)}
    </div>
  )
}

export const Demand = (props: { candidate: Candidate; participation?: Participation; roles: string[] }) => {
  const idea = props.candidate.idea
  if (!idea) return <p className="r-muted">Not on the ideas board, so there are no community votes.</p>
  const p = props.participation
  return (
    <div className="r-demand">
      <p>
        <strong>{idea.votes}</strong> community votes
        {idea.historicalVotes > 0 && <span className="r-muted"> (includes {idea.historicalVotes} carried over from GitHub, no role data)</span>}
      </p>
      {p && p.totalVoters > 0 && (
        <>
          <table className="r-table r-table--compact">
            <caption className="r-hint">
              Self-reported roles of voters on the board. Optional and multi-select, so groups overlap. Votes are not weighted.
            </caption>
            <tbody>
              {props.roles.map((role) => (
                <tr key={role}>
                  <th scope="row">{roleLabels[role]}</th>
                  <td>{p.byRole.find((r) => r.role === role)?.participants ?? 0}</td>
                </tr>
              ))}
              <tr>
                <th scope="row">No role given</th>
                <td>{p.noRole}</td>
              </tr>
            </tbody>
          </table>
        </>
      )}
    </div>
  )
}
