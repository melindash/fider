import { useState } from "react"
import { Post } from "@fider/models"
import { actions, notify } from "@fider/services"
import { useFider } from "./use-fider"

// Prompt for roles at most once per page load; the server remembers dismissals across visits.
let rolesCheckedThisPage = false

export const useMageOSVote = (post: Post, votes: number) => {
  const fider = useFider()
  const [hasVoted, setHasVoted] = useState(post.hasVoted)
  const [votesCount, setVotesCount] = useState(votes)
  const [isSignInModalOpen, setIsSignInModalOpen] = useState(false)
  const [isRolesModalOpen, setIsRolesModalOpen] = useState(false)
  const [notice, setNotice] = useState<string | undefined>()

  const promptForRolesOnce = async () => {
    if (rolesCheckedThisPage) return
    rolesCheckedThisPage = true
    const participant = await actions.getParticipant()
    if (participant.ok && !participant.data.rolesPrompted) {
      setIsRolesModalOpen(true)
    }
  }

  const applyVote = () => {
    setVotesCount(votesCount + (hasVoted ? -1 : 1))
    setHasVoted(!hasVoted)
    if (!hasVoted) {
      void promptForRolesOnce()
    }
  }

  const voteOrUndo = async (): Promise<boolean> => {
    setNotice(undefined)

    if (fider.session.isAuthenticated) {
      const response = await (hasVoted ? actions.removeVote : actions.addVote)(post.number)
      if (response.ok) applyVote()
      return response.ok
    }

    const response = await (hasVoted ? actions.removeAnonymousVote : actions.addAnonymousVote)(post.number)
    if (response.ok) {
      applyVote()
      return true
    }
    if (response.status === 429) {
      const data = response.data as { message?: string }
      setNotice(data.message)
      if (data.message) notify.error(data.message)
    } else {
      setIsSignInModalOpen(true)
    }
    return false
  }

  return {
    hasVoted,
    votesCount,
    voteOrUndo,
    notice,
    isSignInModalOpen,
    openSignIn: () => setIsSignInModalOpen(true),
    closeSignIn: () => setIsSignInModalOpen(false),
    isRolesModalOpen,
    openRoles: () => setIsRolesModalOpen(true),
    closeRoles: () => setIsRolesModalOpen(false),
  }
}
