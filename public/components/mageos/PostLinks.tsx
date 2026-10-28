import React, { useEffect, useState } from "react"
import { Trans } from "@lingui/react/macro"
import { Button, Checkbox, Input } from "@fider/components"
import { HStack, VStack } from "@fider/components/layout"
import { useFider } from "@fider/hooks"
import { actions } from "@fider/services"
import type { PostLink } from "@fider/services/actions/mageos"

const kindLabel = (link: PostLink): JSX.Element => {
  if (link.isSource) return <Trans id="mageos.links.source">Original discussion</Trans>
  switch (link.kind) {
    case "pull_request":
      return <Trans id="mageos.links.pr">Pull request</Trans>
    case "issue":
      return <Trans id="mageos.links.issue">Issue</Trans>
    default:
      return <Trans id="mageos.links.discussion">Discussion</Trans>
  }
}

export const PostLinks = (props: { postNumber: number }) => {
  const fider = useFider()
  const canEdit = fider.session.isAuthenticated && fider.session.user.isCollaborator
  const [links, setLinks] = useState<PostLink[]>([])
  const [url, setUrl] = useState("")
  const [isSource, setIsSource] = useState(false)
  const [attribution, setAttribution] = useState("")
  const [error, setError] = useState<string | undefined>()

  const load = () => actions.listPostLinks(props.postNumber).then((r) => r.ok && setLinks(r.data))
  useEffect(() => {
    load()
  }, [props.postNumber])

  const add = async () => {
    const result = await actions.addPostLink(props.postNumber, url, isSource, attribution)
    if (result.ok) {
      setUrl("")
      setAttribution("")
      setError(undefined)
      load()
    } else {
      setError(result.data.message || "The link could not be added.")
    }
  }

  const remove = async (id: number) => {
    if ((await actions.deletePostLink(props.postNumber, id)).ok) load()
  }

  if (links.length === 0 && !canEdit) return null

  return (
    <section aria-labelledby="mageos-links-title">
      <h3 id="mageos-links-title" className="text-semibold mb-2">
        <Trans id="mageos.links.title">On GitHub</Trans>
      </h3>
      <ul className="mb-2">
        {links.map((link) => (
          <li key={link.id} className="text-sm">
            <HStack spacing={2}>
              <span className="text-muted">{kindLabel(link)}:</span>
              <a href={link.url} target="_blank" rel="noopener noreferrer" className="text-link">
                {link.repo}#{link.number}
              </a>
              {link.attribution && <span className="text-muted">({link.attribution})</span>}
              {link.state && <span className="text-muted">· {link.state}</span>}
              {canEdit && (
                <Button variant="link" size="small" onClick={() => remove(link.id)} aria-label={`Remove link ${link.repo}#${link.number}`}>
                  <Trans id="action.remove">Remove</Trans>
                </Button>
              )}
            </HStack>
          </li>
        ))}
      </ul>
      {canEdit && (
        <VStack spacing={2}>
          <Input field="mageos-link-url" label="GitHub link" placeholder="https://github.com/owner/repo/pull/123" value={url} onChange={setUrl} />
          <Checkbox field="mageos-link-source" checked={isSource} onChange={setIsSource}>
            <Trans id="mageos.links.issource">This is where the idea was first proposed</Trans>
          </Checkbox>
          {isSource && <Input field="mageos-link-attribution" label="Original author" value={attribution} onChange={setAttribution} />}
          {error && (
            <p className="text-sm text-red-700" role="alert">
              {error}
            </p>
          )}
          <Button variant="secondary" size="small" onClick={add} disabled={!url}>
            <Trans id="mageos.links.add">Add link</Trans>
          </Button>
        </VStack>
      )}
    </section>
  )
}
