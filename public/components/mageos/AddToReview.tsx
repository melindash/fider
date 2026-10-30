import React, { useState } from "react"
import { Button } from "@fider/components"
import { useFider } from "@fider/hooks"

// Lets team members put an idea on the roadmap review list from the idea page.
export const AddToReview = (props: { postNumber: number }) => {
  const fider = useFider()
  const [message, setMessage] = useState<string>()
  if (!fider.session.isAuthenticated || !fider.session.user.isCollaborator) return null

  const add = async () => {
    const response = await fetch("/api/v1/mageos/review/candidates", {
      method: "POST",
      headers: { Accept: "application/json", "Content-Type": "application/json" },
      body: JSON.stringify({ postNumber: props.postNumber }),
      credentials: "same-origin",
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) return setMessage(data.message || "Could not add the idea.")
    location.href = `/review#/candidate/${data.id}`
  }

  return (
    <div>
      <Button variant="tertiary" size="small" onClick={add}>
        Open in roadmap review
      </Button>
      {message && (
        <p className="text-sm text-red-700" role="alert">
          {message}
        </p>
      )}
    </div>
  )
}
