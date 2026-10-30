import React, { useEffect, useId, useState } from "react"
import { ApiResult } from "./api"

type Save = (value: string) => Promise<ApiResult<unknown>>

// Saves on blur so the table stays usable during a meeting; each field reports its own errors.
const useAutosave = (initial: string, save: Save) => {
  const [value, setValue] = useState(initial)
  const [saved, setSaved] = useState(initial)
  const [state, setState] = useState<"idle" | "saving" | "saved" | "error">("idle")
  const [error, setError] = useState<string>()

  useEffect(() => {
    setValue(initial)
    setSaved(initial)
  }, [initial])

  const commit = async (next = value) => {
    if (next === saved) return
    setState("saving")
    const result = await save(next)
    if (result.ok) {
      setSaved(next)
      setState("saved")
      setError(undefined)
    } else {
      setState("error")
      setError(result.message)
    }
  }
  return { value, setValue, commit, state, error }
}

const Status = (props: { state: string; error?: string }) => {
  if (props.state === "error")
    return (
      <span className="r-status r-status--error" role="alert">
        {props.error || "Not saved"}
      </span>
    )
  if (props.state === "saving") return <span className="r-status">Saving…</span>
  if (props.state === "saved") return <span className="r-status r-status--ok">Saved</span>
  return null
}

interface TextFieldProps {
  label: string
  value: string
  onSave: Save
  multiline?: boolean
  missing?: boolean
  hint?: string
  disabled?: boolean
  placeholder?: string
}

export const TextField = (props: TextFieldProps) => {
  const id = useId()
  const f = useAutosave(props.value, props.onSave)
  const common = {
    id,
    value: f.value,
    disabled: props.disabled,
    placeholder: props.placeholder,
    onBlur: () => f.commit(),
    "aria-describedby": props.hint ? `${id}-hint` : undefined,
  }
  return (
    <div className={`r-field ${props.missing && !f.value ? "r-field--missing" : ""}`}>
      <label htmlFor={id}>
        {props.label}
        {props.missing && !f.value && <span className="r-missing-tag">Missing</span>}
        <Status state={f.state} error={f.error} />
      </label>
      {props.hint && (
        <span id={`${id}-hint`} className="r-hint">
          {props.hint}
        </span>
      )}
      {props.multiline ? (
        <textarea {...common} rows={3} onChange={(e) => f.setValue(e.target.value)} />
      ) : (
        <input {...common} type="text" onChange={(e) => f.setValue(e.target.value)} />
      )}
    </div>
  )
}

interface SelectFieldProps {
  label: string
  value: string
  options: string[]
  labels: Record<string, string>
  onSave: Save
  missing?: boolean
  disabled?: boolean
  inline?: boolean
}

export const SelectField = (props: SelectFieldProps) => {
  const id = useId()
  const f = useAutosave(props.value, props.onSave)
  return (
    <div className={`r-field ${props.inline ? "r-field--inline" : ""} ${props.missing && !f.value ? "r-field--missing" : ""}`}>
      <label htmlFor={id}>
        {props.label}
        {props.missing && !f.value && <span className="r-missing-tag">Missing</span>}
        <Status state={f.state} error={f.error} />
      </label>
      <select
        id={id}
        value={f.value}
        disabled={props.disabled}
        onChange={(e) => {
          f.setValue(e.target.value)
          f.commit(e.target.value)
        }}
      >
        {props.options.map((o) => (
          <option key={o} value={o}>
            {props.labels[o] ?? o}
          </option>
        ))}
      </select>
    </div>
  )
}

export const Badge = (props: { children: React.ReactNode; tone?: "sample" | "final" | "draft" | "warn" | "muted" }) => (
  <span className={`r-badge r-badge--${props.tone || "muted"}`}>{props.children}</span>
)

export const SampleBadge = () => <Badge tone="sample">Sample data, not a real commitment</Badge>

export const ErrorText = (props: { message?: string }) =>
  props.message ? (
    <p className="r-error" role="alert">
      {props.message}
    </p>
  ) : null

export const Empty = (props: { children: React.ReactNode }) => <p className="r-empty">{props.children}</p>
