import React, { useEffect, useState } from "react"
import { Button } from "@fider/components"
import { ApiResult, Volunteer, review } from "./api"
import { Badge, ErrorText } from "./components"

export const volunteerLabel = (v: Volunteer): string => {
  if (v.userName && v.githubLogin) return `${v.userName} (@${v.githubLogin})`
  return v.userName || `@${v.githubLogin}`
}

const FiderAccountPicker = (props: { onPick: (userId: number) => void }) => {
  const [q, setQ] = useState("")
  const [users, setUsers] = useState<{ id: number; name: string }[]>([])

  useEffect(() => {
    if (q.trim().length < 2) return setUsers([])
    const timer = setTimeout(() => review.searchUsers(q.trim()).then((r) => setUsers(r.ok ? (r.data.users ?? []) : [])), 250)
    return () => clearTimeout(timer)
  }, [q])

  return (
    <div className="r-field">
      <label htmlFor="r-volunteer-user">Fider account</label>
      <input id="r-volunteer-user" type="search" value={q} placeholder="Type at least 2 letters of a name" onChange={(e) => setQ(e.target.value)} />
      {users.length > 0 && (
        <ul className="r-list r-list--compact">
          {users.map((u) => (
            <li key={u.id}>
              {u.name}{" "}
              <Button size="small" onClick={() => props.onPick(u.id)}>
                Add
              </Button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

// Volunteers are optional and either kind of identity is enough. The task owner stays the card's Owner text,
// so "Make task owner" copies the volunteer's name into it rather than keeping a second owner flag.
export const Volunteers = (props: {
  candidateId: number
  volunteers: Volunteer[]
  owner: string
  onChanged: () => void
  onSetOwner: (owner: string) => Promise<ApiResult<unknown>>
}) => {
  const [kind, setKind] = useState<"github" | "fider">("github")
  const [github, setGitHub] = useState("")
  const [error, setError] = useState<string>()

  const add = async (input: { userId?: number; github?: string }) => {
    const result = await review.addVolunteer(props.candidateId, input)
    if (!result.ok) return setError(result.message)
    setError(undefined)
    setGitHub("")
    props.onChanged()
  }
  const remove = async (id: number) => {
    const result = await review.removeVolunteer(id)
    if (!result.ok) return setError(result.message)
    setError(undefined)
    props.onChanged()
  }
  const makeOwner = async (v: Volunteer) => {
    const result = await props.onSetOwner(volunteerLabel(v))
    if (!result.ok) setError(result.message)
  }

  return (
    <div className="r-volunteers">
      {props.volunteers.length === 0 ? (
        <p className="r-muted">No volunteers yet.</p>
      ) : (
        <ul className="r-list">
          {props.volunteers.map((v) => (
            <li key={v.id}>
              {v.userName && <strong>{v.userName}</strong>}{" "}
              {v.githubLogin && (
                <a href={`https://github.com/${v.githubLogin}`} target="_blank" rel="noopener noreferrer">
                  @{v.githubLogin}
                </a>
              )}{" "}
              {props.owner === volunteerLabel(v) ? (
                <Badge tone="final">Task owner</Badge>
              ) : (
                <Button size="small" variant="tertiary" onClick={() => makeOwner(v)}>
                  Make task owner
                </Button>
              )}{" "}
              <Button size="small" variant="tertiary" onClick={() => remove(v.id)}>
                Remove
              </Button>
            </li>
          ))}
        </ul>
      )}

      <fieldset className="r-volunteers__add">
        <legend>Add a volunteer</legend>
        <div className="r-seg" role="group" aria-label="Identify the volunteer by">
          <Button size="small" variant={kind === "github" ? "primary" : "secondary"} onClick={() => setKind("github")}>
            GitHub username
          </Button>
          <Button size="small" variant={kind === "fider" ? "primary" : "secondary"} onClick={() => setKind("fider")}>
            Fider account
          </Button>
        </div>
        {kind === "github" ? (
          <div className="r-inline-form">
            <div className="r-field">
              <label htmlFor="r-volunteer-github">GitHub username</label>
              <input
                id="r-volunteer-github"
                value={github}
                placeholder="octocat or https://github.com/octocat"
                onChange={(e) => setGitHub(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && github.trim()) add({ github })
                }}
              />
            </div>
            <Button size="small" variant="primary" disabled={!github.trim()} onClick={() => add({ github })}>
              Add
            </Button>
          </div>
        ) : (
          <FiderAccountPicker onPick={(userId) => add({ userId })} />
        )}
      </fieldset>
      <ErrorText message={error} />
    </div>
  )
}
