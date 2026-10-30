import React from "react"
import { useFider } from "@fider/hooks"

// Header link to the roadmap review tool, shown only to collaborators and administrators.
export const MageOSReviewNavLink = (props: { pathname: string }) => {
  const fider = useFider()
  if (!fider.session.isAuthenticated || !fider.session.user.isCollaborator) return null
  const active = props.pathname?.startsWith("/review")
  return (
    <a href="/review" className={`c-header__nav-link ${active ? "c-header__nav-link--active" : ""}`}>
      Roadmap review
    </a>
  )
}
