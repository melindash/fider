import React, { useEffect, useState } from "react"
import { Assessment, Candidate, Participation, Vocabulary, review } from "./api"
import { AssessmentSummary } from "./Assessments"
import { ErrorText, SampleBadge } from "./components"
import { criterionLabels, effortLabels, ideaStatusLabel, label, outcomeLabels, roleLabels, stageLabels, trackLabels } from "./labels"

type Column = { candidate: Candidate; assessments: Assessment[]; participation?: Participation }

// Side-by-side view. Rows are criteria; columns are candidates. Value and readiness stay in separate blocks
// and nothing is summed or ranked.
export const Compare = (props: { ids: number[]; vocab: Vocabulary }) => {
  const [columns, setColumns] = useState<Column[]>()
  const [error, setError] = useState<string>()

  useEffect(() => {
    review.compare(props.ids).then((r) => (r.ok ? setColumns(r.data) : setError(r.message)))
  }, [props.ids.join(",")])

  if (!columns) return error ? <ErrorText message={error} /> : <p className="r-muted">Loading…</p>

  const row = (title: string, render: (c: Column) => React.ReactNode) => (
    <tr>
      <th scope="row">{title}</th>
      {columns.map((col) => (
        <td key={col.candidate.id}>{render(col)}</td>
      ))}
    </tr>
  )
  const block = (title: string) => (
    <tr className="r-compare__block">
      <th scope="rowgroup" colSpan={columns.length + 1}>
        {title}
      </th>
    </tr>
  )
  const criteria = (list: string[]) =>
    list.map((criterion) => (
      <React.Fragment key={criterion}>
        {row(criterionLabels[criterion].title, (col) => (
          <AssessmentSummary bare criteria={[criterion]} assessments={col.assessments} compact={columns.length > 3} />
        ))}
      </React.Fragment>
    ))

  return (
    <div className="r-compare">
      <p>
        <a href="#/candidates">← All candidates</a>
      </p>
      <p className="r-hint">Ratings show how many team members chose each level. There is no combined score; the decision is the team&apos;s.</p>
      <div className="r-scroll">
        <table className="r-table r-compare__table">
          <thead>
            <tr>
              <th scope="col">
                <span className="sr-only">Criterion</span>
              </th>
              {columns.map((col) => (
                <th scope="col" key={col.candidate.id}>
                  <a href={`#/candidate/${col.candidate.id}`}>{col.candidate.title}</a>
                  {col.candidate.isSample && <SampleBadge />}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {block("Overview")}
            {row("Track / categories", (c) => [trackLabels[c.candidate.track], ...(c.candidate.idea?.tags ?? [])].join(", "))}
            {row("Board status", (c) => (c.candidate.idea ? ideaStatusLabel(c.candidate.idea.status) : "Not on board"))}
            {row("Review stage", (c) => label(stageLabels, c.candidate.stage))}
            {row("Problem", (c) => c.candidate.problem || <span className="r-missing-tag">Missing</span>)}
            {row("Expected outcome", (c) => c.candidate.outcome || <span className="r-missing-tag">Missing</span>)}
            {row("Last decision", (c) => (c.candidate.latestOutcome ? label(outcomeLabels, c.candidate.latestOutcome) : "None"))}

            {block("Community votes (one input among several)")}
            {row("Votes", (c) => (c.candidate.idea ? c.candidate.idea.votes : "n/a"))}
            {props.vocab.roles.map((role) =>
              row(roleLabels[role], (c) => (c.participation ? (c.participation.byRole.find((r) => r.role === role)?.participants ?? 0) : "n/a"))
            )}
            {row("No role given", (c) => (c.participation ? c.participation.noRole : "n/a"))}

            {block("Value")}
            {criteria(props.vocab.valueCriteria)}

            {block("Delivery readiness")}
            {criteria(props.vocab.readinessCriteria)}
            {row("Rough effort", (c) => effortLabels[c.candidate.effort])}
            {row("Initiative owner", (c) => c.candidate.owner || <span className="r-missing-tag">Missing</span>)}
            {row("Reviewer", (c) => c.candidate.reviewer || <span className="r-missing-tag">Missing</span>)}
            {row("Dependencies", (c) => c.candidate.dependencies || "None recorded")}
            {row("Operating needs", (c) => c.candidate.operatingNeeds || <span className="r-missing-tag">Missing</span>)}
            {row("Open questions", (c) => c.candidate.openQuestions || "None recorded")}
          </tbody>
        </table>
      </div>
      <p className="r-hint">Roles are optional and self-reported; voters can pick several, so role counts can add up to more than the vote total.</p>
    </div>
  )
}
