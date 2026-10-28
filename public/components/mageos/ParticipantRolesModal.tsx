import React, { useEffect, useState } from "react"
import { Trans } from "@lingui/react/macro"
import { Button, Checkbox, Modal } from "@fider/components"
import { CloseIcon } from "@fider/components/common"
import { HStack, VStack } from "@fider/components/layout"
import { actions } from "@fider/services"

interface ParticipantRolesModalProps {
  isOpen: boolean
  // "prompt" appears once after a first vote; closing it records a skip so it never returns.
  // "edit" is opened on purpose, so closing it changes nothing.
  mode: "prompt" | "edit"
  onClose: (roles?: string[]) => void
}

const roleLabel = (role: string): JSX.Element => {
  switch (role) {
    case "merchant":
      return <Trans id="mageos.role.merchant">Merchant / e-commerce manager</Trans>
    case "developer":
      return <Trans id="mageos.role.developer">Developer</Trans>
    case "agency":
      return <Trans id="mageos.role.agency">Agency</Trans>
    case "marketing":
      return <Trans id="mageos.role.marketing">Marketing</Trans>
    default:
      return <Trans id="mageos.role.other">Other</Trans>
  }
}

export const ParticipantRolesModal = (props: ParticipantRolesModalProps) => {
  const [available, setAvailable] = useState<string[]>([])
  const [selected, setSelected] = useState<string[]>([])
  const [loaded, setLoaded] = useState(false)
  const [error, setError] = useState(false)

  useEffect(() => {
    if (!props.isOpen) return
    setLoaded(false)
    actions.getParticipant().then((result) => {
      if (result.ok) {
        setAvailable(result.data.availableRoles)
        setSelected(result.data.roles)
      }
      setLoaded(true)
    })
  }, [props.isOpen])

  const toggle = (role: string, checked: boolean) => {
    setSelected(checked ? [...selected.filter((r) => r !== role), role] : selected.filter((r) => r !== role))
  }

  const save = async (roles: string[]) => {
    const ordered = available.filter((role) => roles.includes(role))
    const result = await actions.setParticipantRoles(ordered)
    if (result.ok) {
      setError(false)
      props.onClose(ordered)
    } else {
      setError(true)
    }
  }

  const dismiss = () => (props.mode === "prompt" ? save(selected) : props.onClose())

  return (
    <Modal.Window isOpen={props.isOpen} onClose={dismiss}>
      <Modal.Header>
        <HStack justify="between">
          <h2 className="text-title" id="mageos-roles-title">
            <Trans id="mageos.roles.title">Which of these describe you?</Trans>
          </h2>
          <CloseIcon closeModal={dismiss} />
        </HStack>
      </Modal.Header>
      <Modal.Content>
        <VStack spacing={4}>
          <p className="text-muted">
            <Trans id="mageos.roles.explanation">
              Optional. Choose any that apply. Roles help maintainers see which groups are taking part. They are self-reported, only ever shown as totals, and
              never change how much a vote counts.
            </Trans>
          </p>
          {loaded && (
            <fieldset aria-labelledby="mageos-roles-title">
              {available.map((role) => (
                <Checkbox key={role} field={`mageos-role-${role}`} checked={selected.includes(role)} onChange={(checked) => toggle(role, checked)}>
                  {roleLabel(role)}
                </Checkbox>
              ))}
            </fieldset>
          )}
          {error && (
            <p className="text-red-700" role="alert">
              <Trans id="mageos.roles.error">Your roles could not be saved. Please try again.</Trans>
            </p>
          )}
        </VStack>
      </Modal.Content>
      <Modal.Footer>
        <Button variant="primary" onClick={() => save(selected)} disabled={!loaded}>
          <Trans id="action.save">Save</Trans>
        </Button>
        {props.mode === "prompt" ? (
          <Button variant="tertiary" onClick={() => save([])} disabled={!loaded}>
            <Trans id="mageos.roles.skip">Skip</Trans>
          </Button>
        ) : (
          <Button variant="tertiary" onClick={() => props.onClose()}>
            <Trans id="action.cancel">Cancel</Trans>
          </Button>
        )}
      </Modal.Footer>
    </Modal.Window>
  )
}
