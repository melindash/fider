import React, { useState } from "react"
import { Trans } from "@lingui/react/macro"
import { i18n } from "@lingui/core"
import { Button, Form, Input, SignInModal } from "@fider/components"
import { VStack } from "@fider/components/layout"
import { actions, Failure, notify } from "@fider/services"

interface AnonymousSubmitProps {
  title: string
  description: string
  tags: string[]
  hasAttachments: boolean
  onError: (error?: Failure) => void
  onSubmitted: () => void
}

// Replaces the sign-in step of the idea form for visitors without an account.
export const AnonymousSubmit = (props: AnonymousSubmitProps) => {
  const [email, setEmail] = useState("")
  const [error, setError] = useState<Failure | undefined>()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [isSignInOpen, setIsSignInOpen] = useState(false)

  const canSubmit = props.title.replace(/\s+/g, " ").trim().length > 9 && email.trim() !== "" && !props.hasAttachments && !isSubmitting

  const submit = async () => {
    setIsSubmitting(true)
    const result = await actions.createAnonymousPost(props.title, props.description, props.tags, email)
    setIsSubmitting(false)
    if (result.ok) {
      setError(undefined)
      props.onError(undefined)
      notify.success(i18n._({ id: "mageos.submit.success", message: "Thanks! Your idea will appear on the board once a moderator has reviewed it." }))
      props.onSubmitted()
      return
    }
    if (result.data?.message) {
      notify.error(result.data.message)
    }
    setError(result.error)
    props.onError(result.error)
  }

  return (
    <VStack spacing={4}>
      <SignInModal isOpen={isSignInOpen} onClose={() => setIsSignInOpen(false)} />
      <Form error={error}>
        <Input
          field="email"
          type="email"
          autoComplete="email"
          label={i18n._({ id: "mageos.submit.email.label", message: "Your email address" })}
          value={email}
          onChange={setEmail}
          placeholder="you@example.com"
        />
        <p className="text-sm text-muted">
          <Trans id="mageos.submit.email.help">
            No account needed. Moderators use your email only to ask about your idea, and it is never shown publicly. New ideas appear after a moderator reviews
            them.
          </Trans>
        </p>
      </Form>
      {props.hasAttachments && (
        <p className="text-sm text-red-700" role="alert">
          <Trans id="mageos.submit.attachments">Remove the images, or sign in to attach them.</Trans>
        </p>
      )}
      <div className="flex justify-center">
        <Button variant="primary" onClick={submit} disabled={!canSubmit}>
          <Trans id="newpost.modal.submit">Submit your idea</Trans>
        </Button>
      </div>
      <p className="text-sm text-center">
        <Trans id="mageos.submit.signin">Have an account?</Trans>{" "}
        <Button variant="link" onClick={() => setIsSignInOpen(true)}>
          <Trans id="action.signin">Sign in</Trans>
        </Button>
      </p>
    </VStack>
  )
}
