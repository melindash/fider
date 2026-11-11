import React, { useEffect, useMemo, useState } from "react"
import { Button } from "@fider/components"
import { Candidate, Meeting, RatingCount, Vocabulary, review } from "./api"
import { Badge, Empty, ErrorText, SampleBadge } from "./components"
import { ideaStatusLabel, label, outcomeLabels, stageLabels, trackLabels } from "./labels"

type BoardIdea = { number: number; title: string; votesCount: number }

const AddCandidate = (props: { vocab: Vocabulary; listed: Set<number>; removed: Set<number>; onAdded: (title: string) => void }) => {
  const [query, setQuery] = useState("")
  const [results, setResults] = useState<BoardIdea[]>()
  const [title, setTitle] = useState("")
  const [sourceUrl, setSourceUrl] = useState("")
  const [track, setTrack] = useState("product")
  const [error, setError] = useState<string>()

  // With no search text the panel lists the whole board by votes, so candidates can be picked by browsing.
  useEffect(() => {
    const q = query.trim()
    if (q.length === 1) return
    const timer = setTimeout(() => review.searchIdeas(q).then((r) => (r.ok ? setResults(r.data) : setError(r.message))), q ? 250 : 0)
    return () => clearTimeout(timer)
  }, [query])

  const add = async (input: Parameters<typeof review.addCandidate>[0], label: string) => {
    const result = await review.addCandidate(input)
    if (!result.ok) return setError(result.message)
    setError(undefined)
    setTitle("")
    setSourceUrl("")
    props.onAdded(label)
  }

  const available = (results || []).filter((idea) => !props.listed.has(idea.number))

  return (
    <div className="r-panel">
      <div className="r-add">
        <div className="r-field">
          <label htmlFor="r-idea-search">From the ideas board</label>
          <input id="r-idea-search" type="search" value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Filter ideas" autoFocus />
          {!results ? (
            <p className="r-muted">Loading ideas…</p>
          ) : available.length === 0 ? (
            <p className="r-muted">{query.trim() ? "No matching ideas that aren't already candidates." : "Every idea on the board is already a candidate."}</p>
          ) : (
            <ul className="r-list r-list--compact r-browse">
              {available.map((idea) => (
                <li key={idea.number}>
                  <Button size="small" variant="tertiary" onClick={() => add({ postNumber: idea.number }, idea.title)}>
                    {props.removed.has(idea.number) ? "Add back" : "Add"}
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
            <Button variant="primary" size="small" disabled={!title.trim()} onClick={() => add({ title, sourceUrl, track }, title)}>
              Add initiative
            </Button>
          </div>
        </fieldset>
      </div>
      <ErrorText message={error} />
    </div>
  )
}

const Tally = (props: { counts: RatingCount }) => {
  const { strong, moderate, weak } = props.counts
  if (strong + moderate + weak === 0) return <span className="r-muted">Not rated</span>
  return (
    <span className="r-tally">
      {strong > 0 && <span className="r-tally__strong">{strong} strong</span>}
      {moderate > 0 && <span className="r-tally__moderate">{moderate} moderate</span>}
      {weak > 0 && <span className="r-tally__weak">{weak} weak</span>}
    </span>
  )
}

const CandidateStatus = (props: { c: Candidate }) => {
  const { c } = props
  if (!c.latestOutcome) return <Badge>{label(stageLabels, c.stage)}</Badge>
  return (
    <>
      <Badge tone={c.latestOutcome === "prioritize" ? "final" : "muted"}>{label(outcomeLabels, c.latestOutcome)}</Badge>
      {c.latestOutcome === "prioritize" && c.idea && <div className="r-muted">{ideaStatusLabel(c.idea.status)}</div>}
    </>
  )
}

// Prioritized and declined are settled; every other outcome (defer, investigate, …) still comes back for a decision.
const isSettled = (c: Candidate) => c.latestOutcome === "prioritize" || c.latestOutcome === "decline"

const views = [
  { key: "open", title: "Needs a decision", test: (c: Candidate) => !isSettled(c) },
  { key: "prioritized", title: "Prioritized", test: (c: Candidate) => c.latestOutcome === "prioritize" },
  { key: "all", title: "All", test: () => true },
]

// Manual initiatives have no votes, so they sort after board ideas rather than tying with zero-vote ideas.
const byVotes = (a: Candidate, b: Candidate) => (b.idea?.votes ?? -1) - (a.idea?.votes ?? -1) || a.title.localeCompare(b.title)

export const Candidates = (props: { vocab: Vocabulary }) => {
  const [everything, setEverything] = useState<Candidate[]>()
  const [undo, setUndo] = useState<{ id: number; title: string; stage: string }>()
  const [meetings, setMeetings] = useState<Meeting[]>([])
  const [error, setError] = useState<string>()
  const [notice, setNotice] = useState<string>()
  const [selected, setSelected] = useState<number[]>([])
  const [adding, setAdding] = useState(false)
  const [view, setView] = useState("open")
  const [text, setText] = useState("")
  const [track, setTrack] = useState("")
  const [owner, setOwner] = useState("")
  const [card, setCard] = useState("")
  const [meetingId, setMeetingId] = useState("")

  const load = () => {
    review.candidates(true).then((r) => (r.ok ? setEverything(r.data) : setError(r.message)))
    review.meetings().then((r) => r.ok && setMeetings(r.data))
  }
  useEffect(load, [])

  const candidates = useMemo(() => everything?.filter((c) => c.stage !== "archived"), [everything])
  const listed = useMemo(() => new Set((candidates || []).flatMap((c) => (c.idea ? [c.idea.number] : []))), [candidates])
  const removed = useMemo(() => new Set((everything || []).flatMap((c) => (c.stage === "archived" && c.idea ? [c.idea.number] : []))), [everything])

  // Removing archives the candidate, which keeps its ratings, decisions and history if it is added back.
  const remove = async (c: Candidate) => {
    const result = await review.updateCandidate(c.id, { stage: "archived" })
    if (!result.ok) return setError(result.message)
    setSelected(selected.filter((s) => s !== c.id))
    setNotice(undefined)
    setUndo({ id: c.id, title: c.title, stage: c.stage })
    load()
  }
  const restore = async () => {
    if (!undo) return
    const result = await review.updateCandidate(undo.id, { stage: undo.stage })
    if (!result.ok) return setError(result.message)
    setUndo(undefined)
    load()
  }

  const owners = useMemo(() => Array.from(new Set((candidates || []).map((c) => c.owner).filter(Boolean))).sort(), [candidates])
  const areas = useMemo(() => Array.from(new Set((candidates || []).flatMap((c) => c.idea?.tags || []))).sort(), [candidates])

  const matchesFilters = (c: Candidate) => {
    const needle = text.toLowerCase()
    if (needle && !`${c.title} ${c.problem} ${c.idea?.number ?? ""}`.toLowerCase().includes(needle)) return false
    if (track && !(c.track === track || c.idea?.tags.includes(track))) return false
    if (owner === "-" ? c.owner !== "" : owner && c.owner !== owner) return false
    if (card === "complete" && c.missing.length > 0) return false
    if (card === "missing" && c.missing.length === 0) return false
    return true
  }

  const filtered = (candidates || []).filter(matchesFilters)
  const current = views.find((v) => v.key === view) || views[0]
  const visible = filtered.filter(current.test).sort(byVotes)
  const extraFilters = [track, owner, card].filter(Boolean).length

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
      <div className="r-filters" role="search">
        <input type="search" aria-label="Search candidates" placeholder="Search" value={text} onChange={(e) => setText(e.target.value)} />
        <div className="r-seg" role="group" aria-label="Show">
          {views.map((v) => (
            <button key={v.key} type="button" aria-pressed={view === v.key} onClick={() => setView(v.key)}>
              {v.title} <span className="r-seg__count">{filtered.filter(v.test).length}</span>
            </button>
          ))}
        </div>
        <span className="r-filters__end">
          <Button size="small" variant={adding ? "secondary" : "primary"} onClick={() => setAdding(!adding)}>
            {adding ? "Close" : "Add candidates"}
          </Button>
        </span>
      </div>

      <details className="r-more" open={extraFilters > 0 || undefined}>
        <summary>More filters{extraFilters > 0 && ` (${extraFilters})`}</summary>
        <div className="r-filters">
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
          <select aria-label="Owner" value={owner} onChange={(e) => setOwner(e.target.value)}>
            <option value="">Any owner</option>
            <option value="-">No owner</option>
            {owners.map((o) => (
              <option key={o} value={o}>
                {o}
              </option>
            ))}
          </select>
          <select aria-label="Card completeness" value={card} onChange={(e) => setCard(e.target.value)}>
            <option value="">Any card</option>
            <option value="complete">Card complete</option>
            <option value="missing">Missing information</option>
          </select>
        </div>
      </details>

      {adding && (
        <AddCandidate
          vocab={props.vocab}
          listed={listed}
          removed={removed}
          onAdded={(title) => {
            setUndo(undefined)
            setNotice(`Added "${title}".`)
            load()
          }}
        />
      )}

      {selected.length > 0 && (
        <div className="r-toolbar r-toolbar--sticky" aria-live="polite">
          <span>{selected.length} selected</span>
          <Button
            size="small"
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
          <Button size="small" disabled={!meetingId} onClick={shortlist}>
            Add to meeting shortlist
          </Button>
          <Button size="small" variant="tertiary" onClick={() => setSelected([])}>
            Clear
          </Button>
        </div>
      )}
      {notice && <p className="r-status r-status--ok">{notice}</p>}
      {undo && (
        <p className="r-status r-status--ok" aria-live="polite">
          Removed &quot;{undo.title}&quot; from candidates.{" "}
          <Button size="small" variant="tertiary" onClick={restore}>
            Undo
          </Button>
        </p>
      )}
      <ErrorText message={error} />

      {visible.length === 0 ? (
        <Empty>{candidates.length === 0 ? "No candidates yet. Use Add candidates to pull ideas from the board." : "No candidates match."}</Empty>
      ) : (
        <table className="r-table r-table--stack r-table--candidates">
          <thead>
            <tr>
              <th scope="col">
                <span className="sr-only">Select</span>
              </th>
              <th scope="col">Candidate</th>
              <th scope="col">Votes</th>
              <th scope="col">Team value</th>
              <th scope="col">Readiness</th>
              <th scope="col">Status</th>
              <th scope="col">
                <span className="sr-only">Remove</span>
              </th>
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
                  <div className="r-hint">{c.idea && c.idea.tags.length > 0 ? c.idea.tags.join(", ") : trackLabels[c.track]}</div>
                </td>
                <td data-label="Votes">{c.idea ? c.idea.votes : <span className="r-muted">n/a</span>}</td>
                <td data-label="Team value">
                  <Tally counts={c.valueRatings} />
                </td>
                <td data-label="Readiness">
                  <Tally counts={c.readinessRatings} />
                </td>
                <td data-label="Status">
                  <CandidateStatus c={c} />
                </td>
                <td data-label="">
                  <Button size="small" variant="tertiary" onClick={() => remove(c)}>
                    Remove
                  </Button>
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
