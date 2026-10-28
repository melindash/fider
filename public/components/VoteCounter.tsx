import "./VoteCounter.scss"

import React from "react"
import { Post, PostStatus } from "@fider/models"
import { classSet } from "@fider/services"
import { Icon, SignInModal } from "@fider/components"
import { ParticipantRolesModal } from "@fider/components/mageos/ParticipantRolesModal"
import { useFider, useMageOSVote } from "@fider/hooks"
import { t } from "@lingui/core/macro"
import ChevronUp from "@fider/assets/images/chevron-up.svg"

export interface VoteCounterProps {
  post: Post
  size?: "default" | "large"
}

export const VoteCounter = (props: VoteCounterProps) => {
  const fider = useFider()
  const { size = "default" } = props
  const vote = useMageOSVote(props.post, props.post.votesCount)

  const status = PostStatus.Get(props.post.status)
  const isDisabled = status.closed || fider.isReadOnly

  const className = classSet({
    "c-vote-counter__button": true,
    "c-vote-counter__button--voted": !status.closed && vote.hasVoted,
    "c-vote-counter__button--disabled": isDisabled,
    "c-vote-counter__button--large": size === "large",
  })

  const votes = vote.votesCount
  const label = vote.hasVoted
    ? t({ id: "mageos.vote.remove.label", message: `Remove your vote. ${votes} votes` })
    : t({ id: "mageos.vote.add.label", message: `Vote for this idea. ${votes} votes` })

  return (
    <>
      <SignInModal isOpen={vote.isSignInModalOpen} onClose={vote.closeSignIn} />
      <ParticipantRolesModal isOpen={vote.isRolesModalOpen} mode="prompt" onClose={vote.closeRoles} />
      <div className="c-vote-counter">
        <button
          className={className}
          onClick={isDisabled ? undefined : vote.voteOrUndo}
          aria-label={label}
          aria-pressed={vote.hasVoted}
          aria-disabled={isDisabled}
        >
          {!vote.hasVoted && <Icon sprite={ChevronUp} className="c-vote-counter__icon" />}
          <span className="c-vote-counter__count">{vote.votesCount}</span>
        </button>
      </div>
    </>
  )
}
