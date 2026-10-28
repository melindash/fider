// Mage-OS fork. Uses fetch directly instead of the shared http helper so a 403 or 429
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

export const setParticipantRoles = (roles: string[]) => send<{ message?: string }>("/api/v1/mageos/participant/roles", "PUT", { roles })
