import React, { useEffect, useState } from "react"
import { Button } from "@fider/components"
import { HistoryEvent, Item, Meeting, Vocabulary, review, todayISO } from "./api"
import { ActionsPanel } from "./Actions"
import { AssessmentSummary, Demand } from "./Assessments"
import { HistoryList } from "./CandidateCard"
import { Badge, Empty, ErrorText, SampleBadge, SelectField, TextField } from "./components"
import { escalationLabels, fieldLabels, label, outcomeLabels } from "./labels"

export const Meetings = () => {
  const [meetings, setMeetings] = useState<Meeting[]>()
  const [heldOn, setHeldOn] = useState(todayISO())
  const [purpose, setPurpose] = useState("")
  const [error, setError] = useState<string>()

  useEffect(() => {
    review.meetings().then((r) => (r.ok ? setMeetings(r.data) : setError(r.message)))
  }, [])

  const create = async () => {
    const result = await review.createMeeting(heldOn, purpose)
    if (!result.ok) return setError(result.message)
    location.hash = `#/meeting/${result.data.id}`
  }

  return (
    <div>
      <fieldset className="r-panel r-inline-form">
        <legend>New meeting</legend>
        <div className="r-field">
          <label htmlFor="r-meeting-date">Date</label>
          <input id="r-meeting-date" type="date" value={heldOn} onChange={(e) => setHeldOn(e.target.value)} />
        </div>
        <div className="r-field r-grow">
          <label htmlFor="r-meeting-purpose">Purpose</label>
          <input id="r-meeting-purpose" type="text" value={purpose} onChange={(e) => setPurpose(e.target.value)} placeholder="e.g. Pick Q1 priorities" />
        </div>
        <Button variant="primary" size="small" disabled={!heldOn} onClick={create}>
          Create meeting
        </Button>
      </fieldset>
      <ErrorText message={error} />
      {!meetings ? (
        <p className="r-muted">Loading…</p>
      ) : meetings.length === 0 ? (
        <Empty>No meetings yet.</Empty>
      ) : (
        <table className="r-table">
          <thead>
            <tr>
              <th scope="col">Date</th>
              <th scope="col">Purpose</th>
              <th scope="col">Shortlist</th>
              <th scope="col">Final decisions</th>
            </tr>
          </thead>
          <tbody>
            {meetings.map((m) => (
              <tr key={m.id}>
                <td>
                  <a href={`#/meeting/${m.id}`}>{m.heldOn}</a>
                </td>
                <td>
                  {m.purpose} {m.isSample && <SampleBadge />}
                </td>
                <td>{m.itemCount}</td>
                <td>
                  {m.finalCount} of {m.itemCount}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}

const needsFollowUp = (outcome: string) => outcome !== "" && outcome !== "decline"

const Decision = (props: { item: Item; vocab: Vocabulary; onChanged: () => void }) => {
  const { item } = props
  const c = item.candidate
  const [error, setError] = useState<string>()
  const isFinal = item.finalizedAt !== ""
  if (!c) return null

  const save = (field: string) => async (value: string) => {
    const result = await review.updateItem(item.id, { [field]: value })
    if (result.ok) props.onChanged()
    return result
  }
  const run = async (fn: () => Promise<{ ok: boolean; message?: string }>) => {
    const result = await fn()
    if (!result.ok) return setError(result.message)
    setError(undefined)
    props.onChanged()
  }
  const openActions = (item.actions || []).filter((a) => !a.isDone)

  return (
    <section className={`r-decision ${isFinal ? "r-decision--final" : ""}`} aria-labelledby={`r-item-${item.id}`}>
      <header className="r-decision__header">
        <h3 id={`r-item-${item.id}`}>
          <a href={`#/candidate/${c.id}`}>{c.title}</a> {c.isSample && <SampleBadge />}
        </h3>
        {isFinal ? (
          <Badge tone="final">
            Final · {label(outcomeLabels, item.outcome)} · recorded by {item.finalizedBy}
          </Badge>
        ) : (
          <Badge tone="draft">Draft recommendation</Badge>
        )}
      </header>

      <div className="r-decision__grid">
        <div>
          <TextField
            label={fieldLabels.decisionRequested}
            value={item.decisionRequested}
            disabled={isFinal}
            placeholder="What should the team decide?"
            onSave={save("decisionRequested")}
          />
          <p className="r-quote">{c.problem || <span className="r-missing-tag">Problem missing</span>}</p>
          {c.missing.length > 0 && <p className="r-hint">Card missing: {c.missing.map((f) => fieldLabels[f] ?? f).join(", ")}</p>}
          <Demand candidate={c} roles={[]} />
          <details>
            <summary>Team assessments ({c.assessors} people)</summary>
            <h4>Value</h4>
            <AssessmentSummary criteria={props.vocab.valueCriteria} assessments={item.assessments || []} />
            <h4>Delivery readiness</h4>
            <AssessmentSummary criteria={props.vocab.readinessCriteria} assessments={item.assessments || []} />
          </details>
        </div>
        <div>
          <TextField
            label={fieldLabels.notes}
            value={item.notes}
            multiline
            disabled={isFinal}
            onSave={save("notes")}
            hint="Internal. Not included in the exported summary."
          />
          <SelectField label="Outcome" value={item.outcome} options={props.vocab.outcomes} labels={outcomeLabels} disabled={isFinal} onSave={save("outcome")} />
          {item.outcome === "prioritize" && (
            <p className="r-hint">Prioritized is not the same as scheduled. Set Delivery on the candidate when work is scheduled or started.</p>
          )}
          <TextField
            label={fieldLabels.rationale}
            value={item.rationale}
            multiline
            disabled={isFinal}
            missing={item.outcome !== ""}
            onSave={save("rationale")}
          />
          <TextField label={fieldLabels.objections} value={item.objections} multiline disabled={isFinal} onSave={save("objections")} />
          <TextField label={fieldLabels.unresolved} value={item.unresolved} multiline disabled={isFinal} onSave={save("unresolved")} />
          <SelectField
            label={fieldLabels.escalation}
            value={item.escalation}
            options={props.vocab.escalations}
            labels={escalationLabels}
            disabled={isFinal}
            onSave={save("escalation")}
          />
        </div>
      </div>

      <h4>Follow-up actions</h4>
      {needsFollowUp(item.outcome) && openActions.length === 0 && (
        <p className="r-hint r-warn">This outcome usually needs an owner, next step and review date.</p>
      )}
      <ActionsPanel candidateId={c.id} itemId={item.id} actions={item.actions || []} onChanged={props.onChanged} />

      <div className="r-decision__footer">
        {isFinal ? (
          <Button size="small" onClick={() => run(() => review.reopen(item.id))}>
            Reopen as draft
          </Button>
        ) : (
          <>
            <Button variant="primary" size="small" onClick={() => run(() => review.finalize(item.id))}>
              Finalize decision
            </Button>
            <Button variant="tertiary" size="small" onClick={() => run(() => review.removeItem(item.id))}>
              Remove from agenda
            </Button>
          </>
        )}
      </div>
      <ErrorText message={error} />
    </section>
  )
}

// Copies both HTML and plain text so Google Docs keeps headings and lists.
const copySummary = async (meetingId: number): Promise<string> => {
  const result = await review.summary(meetingId)
  if (!result.ok) return result.message || "Could not build the summary."
  try {
    if (typeof ClipboardItem !== "undefined") {
      await navigator.clipboard.write([
        new ClipboardItem({
          "text/html": new Blob([result.data.html], { type: "text/html" }),
          "text/plain": new Blob([result.data.text], { type: "text/plain" }),
        }),
      ])
    } else {
      await navigator.clipboard.writeText(result.data.text)
    }
    return "Summary copied. Paste it into Google Docs."
  } catch {
    return "Your browser blocked clipboard access. Use the text below."
  }
}

export const MeetingView = (props: { id: number; vocab: Vocabulary }) => {
  const [meeting, setMeeting] = useState<Meeting>()
  const [items, setItems] = useState<Item[]>([])
  const [history, setHistory] = useState<HistoryEvent[]>([])
  const [error, setError] = useState<string>()
  const [copyMessage, setCopyMessage] = useState<string>()
  const [summaryText, setSummaryText] = useState<string>()

  const load = () =>
    review.meeting(props.id).then((r) => {
      if (!r.ok) return setError(r.message)
      setMeeting(r.data.meeting)
      setItems(r.data.items)
      setHistory(r.data.history)
    })
  useEffect(() => {
    load()
  }, [props.id])

  if (!meeting) return error ? <ErrorText message={error} /> : <p className="r-muted">Loading…</p>

  const exportSummary = async () => {
    setCopyMessage(await copySummary(meeting.id))
    const r = await review.summary(meeting.id)
    if (r.ok) setSummaryText(r.data.text)
  }

  return (
    <div className="r-meeting">
      <p>
        <a href="#/meetings">← All meetings</a>
      </p>
      <div className="r-inline-form">
        <TextField
          label="Date (YYYY-MM-DD)"
          value={meeting.heldOn}
          onSave={(v) => review.updateMeeting(meeting.id, v, meeting.purpose).then((r) => (load(), r))}
        />
        <div className="r-grow">
          <TextField label="Purpose" value={meeting.purpose} onSave={(v) => review.updateMeeting(meeting.id, meeting.heldOn, v).then((r) => (load(), r))} />
        </div>
        {meeting.isSample && <SampleBadge />}
      </div>
      <div className="r-toolbar">
        <span>
          {meeting.finalCount} of {meeting.itemCount} decisions final
        </span>
        <Button
          size="small"
          disabled={items.length === 0}
          onClick={() => {
            location.hash = `#/compare/${items.map((i) => i.candidateId).join(",")}`
          }}
        >
          Compare shortlist
        </Button>
        <Button size="small" variant="primary" disabled={items.length === 0} onClick={exportSummary}>
          Copy summary for Google Docs
        </Button>
        {copyMessage && <span className="r-status r-status--ok">{copyMessage}</span>}
      </div>
      {summaryText && (
        <details className="r-panel">
          <summary>Summary as plain text</summary>
          <textarea className="r-summary" readOnly rows={12} value={summaryText} aria-label="Meeting summary" />
        </details>
      )}
      <ErrorText message={error} />

      {items.length === 0 ? (
        <Empty>
          The shortlist is empty. Select candidates on the <a href="#/candidates">Candidates</a> tab and add them to this meeting.
        </Empty>
      ) : (
        items.map((item) => <Decision key={item.id} item={item} vocab={props.vocab} onChanged={load} />)
      )}

      <details className="r-panel">
        <summary>Meeting history ({history.length})</summary>
        <HistoryList events={history} />
      </details>
    </div>
  )
}
