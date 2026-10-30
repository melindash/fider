import { http } from "../http"

// Mage-OS fork. Most calls use fetch directly instead of the shared http helper so a 403 or 429
// can be handled in the vote UI rather than as a generic error toast.

export interface MageOSResult<T> {
  ok: boolean
  status: number
  data: T
}

export interface Participant {
  roles: string[]
  rolesPrompted: boolean
  availableRoles: string[]
  isAnonymous: boolean
  email?: string
}

async function send<T>(url: string, method: "GET" | "POST" | "PUT" | "DELETE", body?: unknown): Promise<MageOSResult<T>> {
  const response = await fetch(url, {
    method,
    headers: { Accept: "application/json", "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
    credentials: "same-origin",
  })
  const data = (await response.json().catch(() => ({}))) as T
  return { ok: response.status < 400, status: response.status, data }
}

export const addAnonymousVote = (postNumber: number) =>
  send<{ isNewVoter?: boolean; reason?: string; message?: string }>(`/api/v1/posts/${postNumber}/anonymous-vote`, "POST")

export const removeAnonymousVote = (postNumber: number) => send<unknown>(`/api/v1/posts/${postNumber}/anonymous-vote`, "DELETE")

export const getParticipant = () => send<Participant>("/api/v1/mageos/participant", "GET")

// email is only sent for visitors without an account; leaving it undefined keeps the stored one.
export const setParticipantRoles = (roles: string[], email?: string) => send<{ message?: string }>("/api/v1/mageos/participant/roles", "PUT", { roles, email })

// Goes through the shared http helper so validation errors reach the idea form's fields.
export const createAnonymousPost = (title: string, description: string, tags: string[], email: string) =>
  http.post<{ number: number; slug: string; isApproved: boolean; message?: string }>("/api/v1/mageos/anonymous-posts", { title, description, tags, email })

export interface AnonymousSubmission {
  email: string
  createdAt: string
  claimed: boolean
}

export const getAnonymousSubmission = (postNumber: number) => send<AnonymousSubmission>(`/api/v1/posts/${postNumber}/anonymous-submission`, "GET")

export interface PostLink {
  id: number
  kind: "discussion" | "issue" | "pull_request"
  url: string
  repo: string
  number: number
  isSource: boolean
  attribution?: string
  state?: string
}

export const listPostLinks = (postNumber: number) => send<PostLink[]>(`/api/v1/posts/${postNumber}/links`, "GET")

export const addPostLink = (postNumber: number, url: string, isSource: boolean, attribution: string) =>
  send<PostLink & { message?: string }>(`/api/v1/posts/${postNumber}/links`, "POST", { url, isSource, attribution })

export const deletePostLink = (postNumber: number, linkID: number) => send<unknown>(`/api/v1/posts/${postNumber}/links/${linkID}`, "DELETE")
