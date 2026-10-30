import React, { useEffect, useMemo, useState } from "react"
import { Button } from "@fider/components"
import { Candidate, Meeting, Vocabulary, review } from "./api"
import { Badge, Empty, ErrorText, SampleBadge } from "./components"
import { ideaStatusLabel, label, outcomeLabels, stageLabels, trackLabels } from "./labels"

const AddCandidate = (props: { vocab: Vocabulary; onAdded: (id: number) => void }) => {
  const [query, setQuery] = useState("")
  const [results, setResults] = useState<{ number: number; title: string; votesCount: number }[]>([])
  const [title, setTitle] = useState("")
  const [sourceUrl, setSourceUrl] = useState("")
  const [track, setTrack] = useState("product")
  const [error, setError] = useState<string>()

  useEffect(() => {
    if (query.trim().length < 2) {
      setResults([])
      return
    }
    const timer = setTimeout(() => review.searchIdeas(query).then((r) => r.ok && setResults(r.data)), 250)
    return () => clearTimeout(timer)
  }, [query])

  const add = async (input: Parameters<typeof review.addCandidate>[0]) => {
    const result = await review.addCandidate(input)
    if (!result.ok) return setError(result.message)
    setError(undefined)
    setQuery("")
    setTitle("")
    setSourceUrl("")
    props.onAdded(result.data.id)
  }

  return (
    <details className="r-panel">
      <summary>Add candidates</summary>
      <div className="r-add">
        <div className="r-field">
          <label htmlFor="r-idea-search">From the ideas board</label>
          <input id="r-idea-search" type="search" value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Search ideas by title" />
          {results.length > 0 && (
            <ul className="r-list r-list--compact">
              {results.map((idea) => (
                <li key={idea.number}>
                  <Button size="small" variant="tertiary" onClick={() => add({ postNumber: idea.number })}>
                    Add
                  </Button>{" "}
                  #{idea.number} {idea.title} <span className="r-muted">· {idea.votesCount} votes</span>
                </li>
              ))}
            </ul>
          )}
        </div>
        <fieldset className="r-field">
          <legend>Manual entry (initiatives not on the board)</legend>
          <label htmlFor="r-manual-title">Title</label>
          <input id="r-manual-title" type="text" value={title} onChange={(e) => setTitle(e.target.value)} />
          <label htmlFor="r-manual-url">Source link (optional)</label>
          <input id="r-manual-url" type="url" value={sourceUrl} onChange={(e) => setSourceUrl(e.target.value)} placeholder="https://" />
          <label htmlFor="r-manual-track">Track</label>
          <select id="r-manual-track" value={track} onChange={(e) => setTrack(e.target.value)}>
            {props.vocab.tracks.map((t) => (
              <option key={t} value={t}>
                {trackLabels[t]}
              </option>
            ))}
          </select>
          <div>
            <Button variant="primary" size="small" disabled={!title.trim()} onClick={() => add({ title, sourceUrl, track })}>
              Add initiative
            </Button>
          </div>
        </fieldset>
      </div>
      <ErrorText message={error} />
    </details>
  )
}

export const Candidates = (props: { vocab: Vocabulary }) => {
  const [candidates, setCandidates] = useState<Candidate[]>()
  const [meetings, setMeetings] = useState<Meeting[]>([])
  const [error, setError] = useState<string>()
  const [notice, setNotice] = useState<string>()
  const [selected, setSelected] = useState<number[]>([])
  const [text, setText] = useState("")
  const [track, setTrack] = useState("")
  const [stage, setStage] = useState("")
  const [owner, setOwner] = useState("")
  const [readiness, setReadiness] = useState("")
  const [meetingId, setMeetingId] = useState("")

  const load = () => {
    review.candidates().then((r) => (r.ok ? setCandidates(r.data) : setError(r.message)))
    review.meetings().then((r) => r.ok && setMeetings(r.data))
  }
  useEffect(load, [])

  const owners = useMemo(() => Array.from(new Set((candidates || []).map((c) => c.owner).filter(Boolean))).sort(), [candidates])
  const areas = useMemo(() => Array.from(new Set((candidates || []).flatMap((c) => c.idea?.tags || []))).sort(), [candidates])

  const visible = (candidates || []).filter((c) => {
    const needle = text.toLowerCase()
    if (needle && !`${c.title} ${c.problem} ${c.idea?.number ?? ""}`.toLowerCase().includes(needle)) return false
    if (track && !(c.track === track || c.idea?.tags.includes(track))) return false
    if (stage && c.stage !== stage) return false
    if (owner === "-" ? c.owner !== "" : owner && c.owner !== owner) return false
    if (readiness === "complete" && c.missing.length > 0) return false
    if (readiness === "missing" && c.missing.length === 0) return false
    return true
  })

  const toggle = (id: number) => setSelected(selected.includes(id) ? selected.filter((s) => s !== id) : [...selected, id])

  const shortlist = async () => {
    const result = await review.addItems(Number(meetingId), selected)
    if (!result.ok) return setError(result.message)
    setNotice(`Added ${selected.length} to the meeting.`)
    setSelected([])
    load()
  }

  const addSample = async () => {
    const result = await review.addSample()
    if (!result.ok) return setError(result.message)
    load()
  }
  const removeSample = async () => {
    const result = await review.removeSample()
    if (!result.ok) return setError(result.message)
    setNotice(`Removed ${result.data.removed} sample candidates and their sample meetings.`)
    load()
  }

  if (!candidates) return error ? <ErrorText message={error} /> : <p className="r-muted">Loading…</p>
  const hasSample = candidates.some((c) => c.isSample)

  return (
    <div>
      <AddCandidate vocab={props.vocab} onAdded={(id) => (location.hash = `#/candidate/${id}`)} />

      <div className="r-filters" role="search">
        <input type="search" aria-label="Search candidates" placeholder="Search" value={text} onChange={(e) => setText(e.target.value)} />
        <select aria-label="Track or category" value={track} onChange={(e) => setTrack(e.target.value)}>
          <option value="">All tracks and categories</option>
          {props.vocab.tracks.map((t) => (
            <option key={t} value={t}>
              {trackLabels[t]}
            </option>
          ))}
          {areas.map((a) => (
            <option key={a} value={a}>
              Board category: {a}
            </option>
          ))}
        </select>
        <select aria-label="Review stage" value={stage} onChange={(e) => setStage(e.target.value)}>
          <option value="">All stages</option>
          {props.vocab.stages
            .filter((s) => s !== "archived")
            .map((s) => (
              <option key={s} value={s}>
                {stageLabels[s]}
              </option>
            ))}
        </select>
        <select aria-label="Owner" value={owner} onChange={(e) => setOwner(e.target.value)}>
          <option value="">Any owner</option>
          <option value="-">No owner</option>
          {owners.map((o) => (
            <option key={o} value={o}>
              {o}
            </option>
          ))}
        </select>
        <select aria-label="Readiness" value={readiness} onChange={(e) => setReadiness(e.target.value)}>
          <option value="">Any readiness</option>
          <option value="complete">Card complete</option>
          <option value="missing">Missing information</option>
        </select>
      </div>

      <div className="r-toolbar" aria-live="polite">
        <span>{selected.length} selected</span>
        <Button
          size="small"
          disabled={selected.length === 0}
          onClick={() => {
            location.hash = `#/compare/${selected.join(",")}`
          }}
        >
          Compare
        </Button>
        <select aria-label="Meeting to add to" value={meetingId} onChange={(e) => setMeetingId(e.target.value)}>
          <option value="">Choose a meeting…</option>
          {meetings.map((m) => (
            <option key={m.id} value={m.id}>
              {m.heldOn} {m.purpose}
            </option>
          ))}
        </select>
        <Button size="small" disabled={selected.length === 0 || !meetingId} onClick={shortlist}>
          Add to meeting shortlist
        </Button>
        {notice && <span className="r-status r-status--ok">{notice}</span>}
      </div>
      <ErrorText message={error} />

      {visible.length === 0 ? (
        <Empty>No candidates match. Add ideas from the board or enter an initiative above.</Empty>
      ) : (
        <table className="r-table r-table--stack">
          <thead>
            <tr>
              <th scope="col">
                <span className="sr-only">Select</span>
              </th>
              <th scope="col">Candidate</th>
              <th scope="col">Track / categories</th>
              <th scope="col">Votes</th>
              <th scope="col">Board status</th>
              <th scope="col">Review stage</th>
              <th scope="col">Owner</th>
              <th scope="col">Card</th>
              <th scope="col">Last decision</th>
            </tr>
          </thead>
          <tbody>
            {visible.map((c) => (
              <tr key={c.id}>
                <td data-label="Select">
                  <input type="checkbox" aria-label={`Select ${c.title}`} checked={selected.includes(c.id)} onChange={() => toggle(c.id)} />
                </td>
                <td data-label="Candidate">
                  <a href={`#/candidate/${c.id}`}>{c.title}</a>
                  {c.idea && <span className="r-muted"> #{c.idea.number}</span>}
                  {c.isSample && <SampleBadge />}
                </td>
                <td data-label="Track / categories">
                  {trackLabels[c.track]}
                  {c.idea && c.idea.tags.length > 0 && <div className="r-muted">{c.idea.tags.join(", ")}</div>}
                </td>
                <td data-label="Votes">{c.idea ? c.idea.votes : <span className="r-muted">n/a</span>}</td>
                <td data-label="Board status">{c.idea ? ideaStatusLabel(c.idea.status) : <span className="r-muted">not on board</span>}</td>
                <td data-label="Review stage">{label(stageLabels, c.stage)}</td>
                <td data-label="Owner">{c.owner || <span className="r-muted">none</span>}</td>
                <td data-label="Card">
                  {c.missing.length === 0 ? <Badge tone="final">Complete</Badge> : <Badge tone="warn">{c.missing.length} missing</Badge>}
                </td>
                <td data-label="Last decision">
                  {c.latestOutcome ? `${outcomeLabels[c.latestOutcome]} (${c.latestDecidedAt})` : <span className="r-muted">none</span>}
                  {c.latestOutcome === "prioritize" && <div className="r-muted">{c.delivery === "none" ? "Not scheduled" : c.delivery.replace("_", " ")}</div>}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <p className="r-footnote">
        {hasSample ? (
          <Button size="small" variant="tertiary" onClick={removeSample}>
            Remove all sample data
          </Button>
        ) : (
          <Button size="small" variant="tertiary" onClick={addSample}>
            Add sample candidates for practice
          </Button>
        )}
      </p>
    </div>
  )
}
