import React, { useState } from "react"
import { Post, PostStatus } from "@fider/models"
import { Button, Icon, SignInModal } from "@fider/components"
import { ParticipantRolesModal } from "@fider/components/mageos/ParticipantRolesModal"
import { PostLinks } from "@fider/components/mageos/PostLinks"
import { AnonymousSubmitter } from "@fider/components/mageos/AnonymousSubmitter"
import { useFider, useMageOSVote } from "@fider/hooks"
import IconThumbsUp from "@fider/assets/images/heroicons-thumbsup.svg"
import IconCheck from "@fider/assets/images/heroicons-check.svg"
import { Trans, Plural } from "@lingui/react/macro"
import { HStack, VStack } from "@fider/components/layout"

interface VoteSectionProps {
  post: Post
  votes: number
  onDataChanged?: () => void
}

export const VoteSection = (props: VoteSectionProps) => {
  const fider = useFider()
  const vote = useMageOSVote(props.post, props.votes)
  const [isEditingRoles, setIsEditingRoles] = useState(false)

  const voteOrUndo = async () => {
    if (await vote.voteOrUndo()) {
      props.onDataChanged?.()
    }
  }

  const status = PostStatus.Get(props.post.status)
  const isDisabled = status.closed || fider.isReadOnly
  const historicalVotes = props.post.historicalVotesCount || 0
  const votes = vote.votesCount

  const buttonText = vote.hasVoted ? <Trans id="action.voted">Voted!</Trans> : <Trans id="action.vote">Vote for this idea</Trans>
  const icon = vote.hasVoted ? IconCheck : IconThumbsUp

  return (
    <>
      <SignInModal isOpen={vote.isSignInModalOpen} onClose={vote.closeSignIn} />
      <ParticipantRolesModal isOpen={vote.isRolesModalOpen} mode="prompt" onClose={vote.closeRoles} />
      <ParticipantRolesModal isOpen={isEditingRoles} mode="edit" onClose={() => setIsEditingRoles(false)} />
      <VStack spacing={4}>
        <div className="align-self-start">
          <Button variant="primary" onClick={voteOrUndo} disabled={isDisabled} aria-pressed={vote.hasVoted} style={{ minWidth: "180px" }}>
            <HStack spacing={2} justify="center" className="w-full">
              <Icon sprite={icon} /> <span>{buttonText}</span>
            </HStack>
          </Button>
        </div>
        {!fider.session.isAuthenticated && !isDisabled && (
          <p className="text-sm text-muted">
            <Trans id="mageos.vote.noaccount">No account needed to vote. You can remove your vote later from this browser.</Trans>
          </p>
        )}
        {vote.notice && (
          <p className="text-sm text-red-700" role="alert">
            {vote.notice}{" "}
            <Button variant="link" onClick={vote.openSignIn}>
              <Trans id="action.signin">Sign in</Trans>
            </Button>
          </p>
        )}
        <HStack align="center" spacing={2}>
          <span className="text-semibold text-2xl" style={{ fontSize: "32px", minHeight: "48px" }} aria-live="polite">
            {votes}
          </span>
          <span className="text-semibold text-lg">
            <Plural id="label.votecount" value={votes} one="Vote" other="Votes" />
          </span>
        </HStack>
        {historicalVotes > 0 && (
          <p className="text-sm text-muted">
            <Trans id="mageos.vote.historical">Includes {historicalVotes} votes carried over from the original GitHub discussion.</Trans>{" "}
            {props.post.historicalSource && (
              <a href={props.post.historicalSource} target="_blank" rel="noopener noreferrer" className="text-link">
                <Trans id="mageos.vote.historical.link">View discussion</Trans>
              </a>
            )}
          </p>
        )}
        <Button variant="link" onClick={() => setIsEditingRoles(true)} className="align-self-start text-sm">
          <Trans id="mageos.roles.edit">Your roles (optional)</Trans>
        </Button>
        <AnonymousSubmitter postNumber={props.post.number} />
        <PostLinks postNumber={props.post.number} />
      </VStack>
    </>
  )
}
