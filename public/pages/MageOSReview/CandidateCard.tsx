import React, { useEffect, useState } from "react"
import { Button } from "@fider/components"
import { PostStatus } from "@fider/models"
import { CandidateDetail, HistoryEvent, Vocabulary, review } from "./api"
import { ActionsPanel } from "./Actions"
import { AssessmentSummary, Demand, MyAssessment } from "./Assessments"
import { Badge, ErrorText, SampleBadge, SelectField, TextField } from "./components"
import { deliveryLabels, effortLabels, fieldLabels, ideaStatusLabel, label, outcomeLabels, stageLabels, trackLabels } from "./labels"

const textFields: { key: string; multiline?: boolean; hint?: string }[] = [
  { key: "problem", multiline: true },
  { key: "beneficiaries", hint: "Merchants, developers, agencies, contributors, event organisers…" },
  { key: "outcome", multiline: true },
  { key: "scope", multiline: true },
  { key: "exclusions", multiline: true },
  { key: "evidence", multiline: true, hint: "Discussions, support threads, data. One link per line." },
  { key: "dependencies", multiline: true },
  { key: "risks", multiline: true },
  { key: "owner", hint: "Who would lead delivery or run the initiative." },
  { key: "reviewer", hint: "Who would review the work." },
  { key: "operatingNeeds", multiline: true, hint: "Maintenance, moderation, hosting, people needed after launch." },
  { key: "openQuestions", multiline: true },
  { key: "links", multiline: true, hint: "GitHub issues or discussions tracking implementation. One per line." },
]

export const HistoryList = (props: { events: HistoryEvent[] }) =>
  props.events.length === 0 ? (
    <p className="r-muted">No changes recorded.</p>
  ) : (
    <ol className="r-history">
      {props.events.map((e) => (
        <li key={e.id}>
          <time dateTime={e.createdAt}>{new Date(e.createdAt).toLocaleString()}</time> <strong>{e.userName}</strong> {e.action.replace("_", " ")} {e.entity}
          {Object.keys(e.changes).length > 0 && (
            <ul>
              {Object.entries(e.changes).map(([field, change]) => (
                <li key={field}>
                  {fieldLabels[field] ?? field}: {change.from ? <del>{change.from}</del> : null} {change.to ? <ins>{change.to}</ins> : <em>cleared</em>}
                </li>
              ))}
            </ul>
          )}
        </li>
      ))}
    </ol>
  )

// Changing the public status is a separate, confirmed step. The public text starts empty so internal
// notes are never published by accident.
const PublicStatus = (props: { detail: CandidateDetail; onChanged: () => void }) => {
  const idea = props.detail.candidate.idea
  const [status, setStatus] = useState(idea?.status ?? "open")
  const [text, setText] = useState("")
  const [confirmed, setConfirmed] = useState(false)
  const [message, setMessage] = useState<string>()
  const [error, setError] = useState<string>()
  if (!idea) return null

  const publish = async () => {
    const result = await review.setPublicStatus(idea.number, status, text)
    if (!result.ok) return setError(result.message)
    await review.logPublicStatus(props.detail.candidate.id, status, text)
    setError(undefined)
    setConfirmed(false)
    setText("")
    setMessage("Public status updated. Followers of the idea are notified.")
    props.onChanged()
  }

  return (
    <details className="r-panel r-public">
      <summary>Update the public status on the board</summary>
      <p className="r-hint">
        Currently <strong>{ideaStatusLabel(idea.status)}</strong>. Everything below is public. Internal notes, ratings and draft decisions are never copied
        here.
      </p>
      <div className="r-field">
        <label htmlFor="r-public-status">New public status</label>
        <select id="r-public-status" value={status} onChange={(e) => setStatus(e.target.value)}>
          {PostStatus.All.filter((s) => s.value !== "duplicate").map((s) => (
            <option key={s.value} value={s.value}>
              {s.title}
            </option>
          ))}
        </select>
      </div>
      <div className="r-field">
        <label htmlFor="r-public-text">Public response (optional)</label>
        <textarea id="r-public-text" rows={3} value={text} onChange={(e) => setText(e.target.value)} />
      </div>
      <label className="r-check">
        <input type="checkbox" checked={confirmed} onChange={(e) => setConfirmed(e.target.checked)} /> I understand this is shown publicly and notifies
        followers.
      </label>
      <Button variant="primary" size="small" disabled={!confirmed} onClick={publish}>
        Update public status
      </Button>
      <ErrorText message={error} />
      {message && <p className="r-status r-status--ok">{message}</p>}
    </details>
  )
}

export const CandidateCard = (props: { id: number; vocab: Vocabulary }) => {
  const [detail, setDetail] = useState<CandidateDetail>()
  const [error, setError] = useState<string>()

  const load = () => review.candidate(props.id).then((r) => (r.ok ? setDetail(r.data) : setError(r.message)))
  useEffect(() => {
    load()
  }, [props.id])

  if (!detail) return error ? <ErrorText message={error} /> : <p className="r-muted">Loading…</p>
  const c = detail.candidate
  const save = (field: string) => async (value: string) => {
    const result = await review.updateCandidate(c.id, { [field]: value })
    if (result.ok) load()
    return result
  }
  const isMissing = (field: string) => c.missing.includes(field)

  return (
    <article className="r-card">
      <header className="r-card__header">
        <p>
          <a href="#/candidates">← All candidates</a>
        </p>
        <h2>
          {c.title} {c.isSample && <SampleBadge />}
        </h2>
        <p className="r-muted">
          {c.idea ? (
            <>
              Idea{" "}
              <a href={`/posts/${c.idea.number}/${c.idea.slug}`} target="_blank" rel="noopener noreferrer">
                #{c.idea.number}
              </a>{" "}
              · {ideaStatusLabel(c.idea.status)} on the board
              {c.idea.tags.length > 0 && ` · ${c.idea.tags.join(", ")}`}
              {!c.idea.isApproved && <Badge tone="warn">Awaiting moderation</Badge>}
            </>
          ) : (
            <>
              Manual entry
              {c.sourceUrl && (
                <>
                  {" · "}
                  <a href={c.sourceUrl} target="_blank" rel="noopener noreferrer">
                    source
                  </a>
                </>
              )}
            </>
          )}
        </p>
        <div className="r-card__meta">
          <SelectField inline label="Track" value={c.track} options={props.vocab.tracks} labels={trackLabels} onSave={save("track")} />
          <SelectField inline label="Review stage" value={c.stage} options={props.vocab.stages} labels={stageLabels} onSave={save("stage")} />
          <SelectField inline label="Delivery" value={c.delivery} options={props.vocab.delivery} labels={deliveryLabels} onSave={save("delivery")} />
          {c.latestOutcome && (
            <span>
              Last decision: <strong>{label(outcomeLabels, c.latestOutcome)}</strong> ({c.latestDecidedAt})
              {c.latestOutcome === "prioritize" && c.delivery === "none" && <Badge tone="draft">Prioritized, not scheduled</Badge>}
            </span>
          )}
        </div>
      </header>

      <div className="r-card__grid">
        <section aria-labelledby="r-card-proposal">
          <h3 id="r-card-proposal">Proposal</h3>
          {c.missing.length > 0 && <p className="r-hint">Incomplete cards can still be discussed. Missing fields show what to ask for.</p>}
          {!c.idea && <TextField label={fieldLabels.title} value={c.title} onSave={save("title")} />}
          {!c.idea && <TextField label={fieldLabels.sourceUrl} value={c.sourceUrl} onSave={save("sourceUrl")} />}
          {c.idea?.description && (
            <details>
              <summary>Description on the board</summary>
              <p className="r-quote">{c.idea.description}</p>
            </details>
          )}
          {textFields.map((f) => (
            <TextField
              key={f.key}
              label={fieldLabels[f.key]}
              value={(c as unknown as Record<string, string>)[f.key]}
              multiline={f.multiline}
              hint={f.hint}
              missing={isMissing(f.key)}
              onSave={save(f.key)}
            />
          ))}
          <SelectField
            label={fieldLabels.effort}
            value={c.effort}
            options={props.vocab.efforts}
            labels={effortLabels}
            missing={isMissing("effort")}
            onSave={save("effort")}
          />
        </section>

        <section aria-labelledby="r-card-assess">
          <h3 id="r-card-assess">Demand</h3>
          <Demand candidate={c} participation={detail.participation} roles={props.vocab.roles} />
          <h3>Your assessment</h3>
          <MyAssessment candidateId={c.id} vocab={props.vocab} assessments={detail.assessments} onSaved={load} />
          <h3>Team assessments</h3>
          <h4>Value</h4>
          <AssessmentSummary criteria={props.vocab.valueCriteria} assessments={detail.assessments} />
          <h4>Delivery readiness</h4>
          <AssessmentSummary criteria={props.vocab.readinessCriteria} assessments={detail.assessments} />
        </section>
      </div>

      <section>
        <h3>Actions</h3>
        <ActionsPanel candidateId={c.id} actions={detail.actions} onChanged={load} />
      </section>

      <section>
        <h3>Decisions</h3>
        {detail.decisions.length === 0 ? (
          <p className="r-muted">Not on any meeting agenda yet.</p>
        ) : (
          <ul className="r-list">
            {detail.decisions.map((d) => (
              <li key={d.id}>
                <a href={`#/meeting/${d.meetingId}`}>{d.meetingHeldOn}</a>: {label(outcomeLabels, d.outcome)}{" "}
                {d.finalizedAt ? <Badge tone="final">Final · {d.finalizedBy}</Badge> : <Badge tone="draft">Draft</Badge>}
                {d.rationale && <div className="r-muted">{d.rationale}</div>}
              </li>
            ))}
          </ul>
        )}
      </section>

      <PublicStatus detail={detail} onChanged={load} />

      <details className="r-panel">
        <summary>History ({detail.history.length})</summary>
        <HistoryList events={detail.history} />
      </details>
    </article>
  )
}
