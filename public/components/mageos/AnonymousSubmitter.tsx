import React, { useEffect, useState } from "react"
import { Trans } from "@lingui/react/macro"
import { useFider } from "@fider/hooks"
import { actions } from "@fider/services"
import type { AnonymousSubmission } from "@fider/services/actions/mageos"

// Shows collaborators the contact email left with an idea submitted without an account.
export const AnonymousSubmitter = (props: { postNumber: number }) => {
  const fider = useFider()
  const canSee = fider.session.isAuthenticated && fider.session.user.isCollaborator
  const [submission, setSubmission] = useState<AnonymousSubmission | undefined>()

  useEffect(() => {
    if (canSee) actions.getAnonymousSubmission(props.postNumber).then((r) => setSubmission(r.ok ? r.data : undefined))
  }, [props.postNumber, canSee])

  if (!submission || submission.claimed) return null

  return (
    <p className="text-sm text-muted">
      <Trans id="mageos.submitter">
        Submitted without an account. Contact:{" "}
        <a className="text-link" href={`mailto:${submission.email}`}>
          {submission.email}
        </a>{" "}
        (unverified, visible to moderators only)
      </Trans>
    </p>
  )
}
