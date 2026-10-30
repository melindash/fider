import React, { useEffect, useState } from "react"
import { Action, Dashboard as DashboardData, review, todayISO } from "./api"
import { Badge, Empty, ErrorText } from "./components"
import { fieldLabels } from "./labels"

const ActionList = (props: { actions: Action[]; empty: string }) =>
  props.actions.length === 0 ? (
    <Empty>{props.empty}</Empty>
  ) : (
    <ul className="r-list">
      {props.actions.map((a) => (
        <li key={a.id}>
          <a href={`#/candidate/${a.candidateId}`}>{a.candidateTitle}</a>: {a.nextStep}
          <span className="r-muted">
            {" "}
            · {a.owner || "Unassigned"}
            {a.reviewOn && ` · review ${a.reviewOn}`}
          </span>
          {a.isSample && <Badge tone="sample">Sample</Badge>}
        </li>
      ))}
    </ul>
  )

export const Dashboard = () => {
  const [data, setData] = useState<DashboardData>()
  const [error, setError] = useState<string>()

  useEffect(() => {
    review.dashboard(todayISO()).then((r) => (r.ok ? setData(r.data) : setError(r.message)))
  }, [])

  if (error) return <ErrorText message={error} />
  if (!data) return <p className="r-muted">Loading…</p>

  return (
    <div className="r-dashboard">
      <section>
        <h2>Upcoming meetings</h2>
        {data.upcomingMeetings.length === 0 ? (
          <Empty>
            No meetings scheduled. <a href="#/meetings">Create one</a>.
          </Empty>
        ) : (
          <ul className="r-list">
            {data.upcomingMeetings.map((m) => (
              <li key={m.id}>
                <a href={`#/meeting/${m.id}`}>
                  {m.heldOn}
                  {m.purpose && `: ${m.purpose}`}
                </a>
                <span className="r-muted"> · {m.itemCount} candidates</span>
                {m.isSample && <Badge tone="sample">Sample</Badge>}
              </li>
            ))}
          </ul>
        )}
      </section>
      <section>
        <h2>
          Overdue follow-ups <Badge tone={data.overdue.length ? "warn" : "muted"}>{data.overdue.length}</Badge>
        </h2>
        <ActionList actions={data.overdue} empty="Nothing overdue." />
      </section>
      <section>
        <h2>Reviews in the next 14 days</h2>
        <ActionList actions={data.upcomingReviews} empty="No reviews due soon." />
      </section>
      <section>
        <h2>
          Unassigned actions <Badge tone={data.unassignedActions.length ? "warn" : "muted"}>{data.unassignedActions.length}</Badge>
        </h2>
        <ActionList actions={data.unassignedActions} empty="Every open action has an owner." />
      </section>
      <section className="r-dashboard__wide">
        <h2>Missing information</h2>
        <p className="r-hint">Open candidates with empty card fields. Missing information is not a reason to decline; it shows what to ask for.</p>
        {data.missingInfo.length === 0 ? (
          <Empty>All open candidates have complete cards.</Empty>
        ) : (
          <ul className="r-list">
            {data.missingInfo.map((c) => (
              <li key={c.id}>
                <a href={`#/candidate/${c.id}`}>{c.title}</a>
                <span className="r-muted"> · {c.missing.map((f) => fieldLabels[f] ?? f).join(", ")}</span>
                {c.isSample && <Badge tone="sample">Sample</Badge>}
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  )
}
