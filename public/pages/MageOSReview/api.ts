// Mage-OS fork: client for the roadmap review API. Errors come back as values so each view can show them inline.

export interface ApiResult<T> {
  ok: boolean
  data: T
  message?: string
}

export async function call<T>(method: "GET" | "POST" | "PUT" | "DELETE", url: string, body?: unknown): Promise<ApiResult<T>> {
  try {
    const response = await fetch(url, {
      method,
      headers: { Accept: "application/json", "Content-Type": "application/json" },
      body: body === undefined ? undefined : JSON.stringify(body),
      credentials: "same-origin",
    })
    const data = await response.json().catch(() => ({}))
    if (response.ok) return { ok: true, data: data as T }
    const message = (data && (data.message || data.errors?.[0]?.message)) || `Request failed (${response.status}).`
    return { ok: false, data: data as T, message }
  } catch {
    return { ok: false, data: {} as T, message: "Network error. Check your connection and try again." }
  }
}

export interface Vocabulary {
  tracks: string[]
  stages: string[]
  delivery: string[]
  efforts: string[]
  ratings: string[]
  outcomes: string[]
  escalations: string[]
  valueCriteria: string[]
  readinessCriteria: string[]
  requiredFields: string[]
  roles: string[]
}

export interface Idea {
  id: number
  number: number
  title: string
  slug: string
  description: string
  status: string
  votes: number
  historicalVotes: number
  tags: string[]
  isApproved: boolean
}

export interface Candidate {
  id: number
  idea?: Idea
  title: string
  sourceUrl: string
  track: string
  stage: string
  delivery: string
  problem: string
  beneficiaries: string
  outcome: string
  scope: string
  exclusions: string
  evidence: string
  dependencies: string
  risks: string
  effort: string
  owner: string
  reviewer: string
  operatingNeeds: string
  openQuestions: string
  links: string
  isSample: boolean
  updatedAt: string
  missing: string[]
  latestOutcome: string
  latestDecidedAt: string
  openActions: number
  assessors: number
}

export interface Assessment {
  candidateId: number
  userId: number
  userName: string
  criterion: string
  rating: string
  note: string
}

export interface Action {
  id: number
  candidateId: number
  candidateTitle: string
  isSample: boolean
  itemId: number
  owner: string
  nextStep: string
  reviewOn: string
  isDone: boolean
}

export interface Meeting {
  id: number
  heldOn: string
  purpose: string
  isSample: boolean
  itemCount: number
  finalCount: number
}

export interface Item {
  id: number
  meetingId: number
  meetingHeldOn: string
  candidateId: number
  decisionRequested: string
  notes: string
  outcome: string
  rationale: string
  objections: string
  unresolved: string
  escalation: string
  finalizedBy: string
  finalizedAt: string
  candidate?: Candidate
  assessments?: Assessment[]
  actions?: Action[]
}

export interface HistoryEvent {
  id: number
  userName: string
  entity: string
  action: string
  changes: Record<string, { from: string; to: string }>
  createdAt: string
}

export interface Participation {
  totalVoters: number
  noRole: number
  byRole: { role: string; participants: number }[]
  historicalVotes: number
}

export interface CandidateDetail {
  candidate: Candidate
  assessments: Assessment[]
  actions: Action[]
  decisions: Item[]
  history: HistoryEvent[]
  participation?: Participation
}

export interface Dashboard {
  missingInfo: Candidate[]
  unassignedActions: Action[]
  upcomingReviews: Action[]
  overdue: Action[]
  upcomingMeetings: Meeting[]
}

const base = "/api/v1/mageos/review"

export const review = {
  vocabulary: () => call<Vocabulary>("GET", `${base}/vocabulary`),
  dashboard: (today: string) => call<Dashboard>("GET", `${base}/dashboard?today=${today}`),
  candidates: () => call<Candidate[]>("GET", `${base}/candidates`),
  addCandidate: (input: { postNumber?: number; title?: string; sourceUrl?: string; track?: string }) =>
    call<{ id: number; existed: boolean }>("POST", `${base}/candidates`, input),
  candidate: (id: number) => call<CandidateDetail>("GET", `${base}/candidates/${id}`),
  updateCandidate: (id: number, fields: Record<string, string>) => call<unknown>("PUT", `${base}/candidates/${id}`, { fields }),
  assess: (id: number, criterion: string, rating: string, note: string) =>
    call<unknown>("PUT", `${base}/candidates/${id}/assessment`, { criterion, rating, note }),
  addAction: (id: number, input: { itemId?: number; owner: string; nextStep: string; reviewOn: string }) =>
    call<Action>("POST", `${base}/candidates/${id}/actions`, input),
  updateAction: (id: number, fields: Record<string, string>) => call<unknown>("PUT", `${base}/actions/${id}`, { fields }),
  logPublicStatus: (id: number, status: string, text: string) => call<unknown>("POST", `${base}/candidates/${id}/public-status-log`, { status, text }),
  compare: (ids: number[]) =>
    call<{ candidate: Candidate; assessments: Assessment[]; participation?: Participation }[]>("GET", `${base}/compare?ids=${ids.join(",")}`),
  meetings: () => call<Meeting[]>("GET", `${base}/meetings`),
  createMeeting: (heldOn: string, purpose: string) => call<{ id: number }>("POST", `${base}/meetings`, { heldOn, purpose }),
  updateMeeting: (id: number, heldOn: string, purpose: string) => call<unknown>("PUT", `${base}/meetings/${id}`, { heldOn, purpose }),
  meeting: (id: number) => call<{ meeting: Meeting; items: Item[]; history: HistoryEvent[] }>("GET", `${base}/meetings/${id}`),
  addItems: (meetingId: number, candidateIds: number[]) => call<unknown>("POST", `${base}/meetings/${meetingId}/items`, { candidateIds }),
  summary: (meetingId: number) => call<{ html: string; text: string }>("GET", `${base}/meetings/${meetingId}/summary`),
  updateItem: (id: number, fields: Record<string, string>) => call<unknown>("PUT", `${base}/items/${id}`, { fields }),
  removeItem: (id: number) => call<unknown>("DELETE", `${base}/items/${id}`),
  finalize: (id: number) => call<{ missing?: string[] }>("POST", `${base}/items/${id}/finalize`),
  reopen: (id: number) => call<unknown>("POST", `${base}/items/${id}/reopen`),
  addSample: () => call<{ ids: number[] }>("POST", `${base}/sample`),
  removeSample: () => call<{ removed: number }>("DELETE", `${base}/sample`),
  searchIdeas: (q: string) =>
    call<{ number: number; title: string; votesCount: number; status: string }[]>(
      "GET",
      `/api/v1/posts?query=${encodeURIComponent(q)}&view=most-wanted&limit=10`
    ),
  setPublicStatus: (postNumber: number, status: string, text: string) => call<unknown>("PUT", `/api/v1/posts/${postNumber}/status`, { status, text }),
}

export const todayISO = (): string => {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`
}
