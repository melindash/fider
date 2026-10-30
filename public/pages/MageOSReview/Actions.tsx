import React, { useState } from "react"
import { Button } from "@fider/components"
import { Action, review } from "./api"
import { ErrorText, TextField } from "./components"

// Follow-ups for one candidate. itemId ties new actions to the meeting decision that created them.
export const ActionsPanel = (props: { candidateId: number; itemId?: number; actions: Action[]; onChanged: () => void }) => {
  const [owner, setOwner] = useState("")
  const [nextStep, setNextStep] = useState("")
  const [reviewOn, setReviewOn] = useState("")
  const [error, setError] = useState<string>()

  const add = async () => {
    const result = await review.addAction(props.candidateId, { itemId: props.itemId, owner, nextStep, reviewOn })
    if (!result.ok) return setError(result.message)
    setError(undefined)
    setOwner("")
    setNextStep("")
    setReviewOn("")
    props.onChanged()
  }

  const update = async (id: number, fields: Record<string, string>) => {
    const result = await review.updateAction(id, fields)
    if (result.ok) props.onChanged()
    return result
  }

  const prefix = `r-action-${props.candidateId}-${props.itemId ?? 0}`
  return (
    <div className="r-actions">
      {props.actions.length > 0 && (
        <table className="r-table r-table--compact">
          <thead>
            <tr>
              <th scope="col">Done</th>
              <th scope="col">Next step</th>
              <th scope="col">Owner</th>
              <th scope="col">Review date</th>
            </tr>
          </thead>
          <tbody>
            {props.actions.map((a) => (
              <tr key={a.id} className={a.isDone ? "r-done" : ""}>
                <td>
                  <input
                    type="checkbox"
                    aria-label={`Mark "${a.nextStep}" done`}
                    checked={a.isDone}
                    onChange={(e) => update(a.id, { done: String(e.target.checked) })}
                  />
                </td>
                <td>
                  <TextField label="Next step" value={a.nextStep} onSave={(v) => update(a.id, { nextStep: v })} />
                </td>
                <td>
                  <TextField label="Owner" value={a.owner} placeholder="Unassigned" missing={!a.isDone} onSave={(v) => update(a.id, { owner: v })} />
                </td>
                <td>
                  <label className="sr-only" htmlFor={`${prefix}-date-${a.id}`}>
                    Review date
                  </label>
                  <input
                    id={`${prefix}-date-${a.id}`}
                    type="date"
                    defaultValue={a.reviewOn}
                    onBlur={(e) => e.target.value !== a.reviewOn && update(a.id, { reviewOn: e.target.value })}
                  />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      <div className="r-action-add">
        <div className="r-field">
          <label htmlFor={`${prefix}-step`}>New next step</label>
          <input id={`${prefix}-step`} type="text" value={nextStep} onChange={(e) => setNextStep(e.target.value)} />
        </div>
        <div className="r-field">
          <label htmlFor={`${prefix}-owner`}>Owner</label>
          <input id={`${prefix}-owner`} type="text" value={owner} onChange={(e) => setOwner(e.target.value)} placeholder="Name or team" />
        </div>
        <div className="r-field">
          <label htmlFor={`${prefix}-date`}>Review date</label>
          <input id={`${prefix}-date`} type="date" value={reviewOn} onChange={(e) => setReviewOn(e.target.value)} />
        </div>
        <Button size="small" disabled={!nextStep.trim()} onClick={add}>
          Add action
        </Button>
      </div>
      <ErrorText message={error} />
    </div>
  )
}
