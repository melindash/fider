import "./Review.page.scss"

import React, { useEffect, useState } from "react"
import { Header } from "@fider/components"
import { Vocabulary, review } from "./api"
import { CandidateCard } from "./CandidateCard"
import { Candidates } from "./Candidates"
import { Compare } from "./Compare"
import { ErrorText } from "./components"
import { Dashboard } from "./Dashboard"
import { MeetingView, Meetings } from "./Meetings"

// Mage-OS fork: roadmap review tool for collaborators. Views live in the URL hash so links can be shared in chat.
const useRoute = () => {
  const read = () => (typeof window === "undefined" ? "" : window.location.hash.replace(/^#\/?/, ""))
  const [route, setRoute] = useState(read())
  useEffect(() => {
    const onChange = () => {
      setRoute(read())
      window.scrollTo(0, 0)
    }
    window.addEventListener("hashchange", onChange)
    return () => window.removeEventListener("hashchange", onChange)
  }, [])
  return route.split("/")
}

const tabs = [
  { key: "dashboard", title: "Follow-ups" },
  { key: "candidates", title: "Candidates" },
  { key: "meetings", title: "Meetings" },
]

const ReviewPage = () => {
  const [vocab, setVocab] = useState<Vocabulary>()
  const [error, setError] = useState<string>()
  const [view, param] = useRoute()

  useEffect(() => {
    review.vocabulary().then((r) => (r.ok ? setVocab(r.data) : setError(r.message)))
  }, [])

  const active = view === "candidate" || view === "compare" ? "candidates" : view === "meeting" ? "meetings" : view || "dashboard"

  const body = () => {
    if (!vocab) return error ? <ErrorText message={error} /> : <p className="r-muted">Loading…</p>
    switch (view) {
      case "candidates":
        return <Candidates vocab={vocab} />
      case "candidate":
        return <CandidateCard id={Number(param)} vocab={vocab} />
      case "compare":
        return <Compare ids={(param || "").split(",").map(Number).filter(Boolean)} vocab={vocab} />
      case "meetings":
        return <Meetings />
      case "meeting":
        return <MeetingView id={Number(param)} vocab={vocab} />
      default:
        return <Dashboard />
    }
  }

  return (
    <>
      <Header />
      <div id="p-mageos-review" className="page container">
        <div className="r-title">
          <h1>Roadmap review</h1>
          <p className="r-hint">Internal to the roadmap team. Nothing here is public until someone updates an idea&apos;s status on the board.</p>
        </div>
        <nav className="r-tabs" aria-label="Roadmap review">
          {tabs.map((t) => (
            <a key={t.key} href={`#/${t.key}`} className={active === t.key ? "r-tabs__active" : ""} aria-current={active === t.key ? "page" : undefined}>
              {t.title}
            </a>
          ))}
        </nav>
        <main>{body()}</main>
      </div>
    </>
  )
}

export default ReviewPage
