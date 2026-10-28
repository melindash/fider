import React from "react"
import { Post, UserRole, PostStatus, UserStatus } from "@fider/models"
import { VoteCounter } from "@fider/components"
import { screen, fireEvent, render } from "@testing-library/react"
import { fiderMock, httpMock, setupModalRoot } from "@fider/services/testing"
import { FiderContext } from "@fider/services"
import { act } from "react"

jest.mock("@fider/services/toastify", () => ({ error: jest.fn(), success: jest.fn() }))

let post: Post

type FetchRoutes = Record<string, [number, unknown]>

const mockFetch = (routes: FetchRoutes) => {
  const participant: [number, unknown] = [200, { roles: [], rolesPrompted: true, availableRoles: [], isAnonymous: true }]
  const fn = jest.fn((url: string, init?: RequestInit) => {
    const [status, body] = routes[`${init?.method || "GET"} ${url}`] || (url === "/api/v1/mageos/participant" ? participant : [200, {}])
    return Promise.resolve({ status, json: () => Promise.resolve(body) } as Response)
  })
  global.fetch = fn as unknown as typeof fetch
  return fn
}

beforeEach(() => {
  setupModalRoot()
  mockFetch({})

  post = {
    id: 1,
    number: 10,
    slug: "add-typescript",
    title: "Add TypeScript",
    description: "",
    createdAt: "",
    status: PostStatus.Started.value,
    user: {
      id: 5,
      name: "John",
      isTrusted: false,
      role: UserRole.Collaborator,
      status: UserStatus.Active,
      avatarURL: "/static/avatars/letter/5/John",
    },
    hasVoted: false,
    response: null,
    votesCount: 5,
    commentsCount: 2,
    tags: [],
    isApproved: true,
  }
})

describe("<VoteCounter />", () => {
  test("when hasVoted === true", () => {
    post.hasVoted = true
    post.votesCount = 9

    const { container } = render(
      <FiderContext.Provider value={fiderMock.authenticated()}>
        <VoteCounter post={post} />
      </FiderContext.Provider>
    )
    const button = container.querySelector("button")

    expect(button).toHaveTextContent("9")
    expect(button).toHaveClass("c-vote-counter__button--voted")
    expect(button).not.toHaveClass("c-vote-counter__button--disabled")
  })

  test("when hasVoted === false", () => {
    post.hasVoted = false
    post.votesCount = 2
    const { container } = render(
      <FiderContext.Provider value={fiderMock.authenticated()}>
        <VoteCounter post={post} />
      </FiderContext.Provider>
    )
    const button = container.querySelector("button")
    expect(button).toHaveTextContent("2")
    expect(button).not.toHaveClass("c-vote-counter__button--voted")
    expect(button).not.toHaveClass("c-vote-counter__button--disabled")
  })

  test("when post is closed", () => {
    post.status = PostStatus.Completed.value
    const { container } = render(
      <FiderContext.Provider value={fiderMock.authenticated()}>
        <VoteCounter post={post} />
      </FiderContext.Provider>
    )
    const button = container.querySelector("button")
    expect(button).toHaveTextContent("5")
    expect(button).toHaveClass("c-vote-counter__button--disabled")
    expect(button).not.toHaveClass("c-vote-counter__button--voted")
  })

  test("click when unauthenticated votes anonymously", async () => {
    const fetchMock = mockFetch({ "POST /api/v1/posts/10/anonymous-vote": [200, { isNewVoter: true }] })

    const { container } = render(
      <FiderContext.Provider value={fiderMock.notAuthenticated()}>
        <VoteCounter post={post} />
      </FiderContext.Provider>
    )
    const button = container.querySelector("button") || fail("button not found")
    await act(async () => {
      fireEvent.click(button)
    })

    expect(fetchMock).toHaveBeenCalledWith("/api/v1/posts/10/anonymous-vote", expect.objectContaining({ method: "POST" }))
    expect(button).toHaveTextContent("6")
    expect(button).toHaveAttribute("aria-pressed", "true")
  })

  test("click when unauthenticated and rate limited keeps the count", async () => {
    mockFetch({ "POST /api/v1/posts/10/anonymous-vote": [429, { reason: "rate", message: "Please sign in to vote." }] })

    const { container } = render(
      <FiderContext.Provider value={fiderMock.notAuthenticated()}>
        <VoteCounter post={post} />
      </FiderContext.Provider>
    )
    const button = container.querySelector("button") || fail("button not found")
    await act(async () => {
      fireEvent.click(button)
    })

    expect(button).toHaveTextContent("5")
    expect(button).toHaveAttribute("aria-pressed", "false")
  })

  test("click when anonymous voting is disabled opens sign in", async () => {
    mockFetch({ "POST /api/v1/posts/10/anonymous-vote": [403, {}] })

    const { container } = render(
      <FiderContext.Provider value={fiderMock.notAuthenticated()}>
        <VoteCounter post={post} />
      </FiderContext.Provider>
    )
    const button = container.querySelector("button") || fail("button not found")
    await act(async () => {
      fireEvent.click(button)
    })

    expect(screen.queryByTestId("modal")).toBeInTheDocument()
    expect(button).toHaveTextContent("5")
  })

  test("button has an accessible label", () => {
    const { container } = render(
      <FiderContext.Provider value={fiderMock.notAuthenticated()}>
        <VoteCounter post={post} />
      </FiderContext.Provider>
    )
    expect(container.querySelector("button")).toHaveAttribute("aria-label", "Vote for this idea. 5 votes")
  })

  test("click when authenticated and hasVoted === false", async () => {
    const mock = httpMock.alwaysOk()

    const { container } = render(
      <FiderContext.Provider value={fiderMock.authenticated()}>
        <VoteCounter post={post} />
      </FiderContext.Provider>
    )

    const button = container.querySelector("button") || fail("button not found")
    expect(button).toHaveTextContent("5")
    await act(async () => {
      fireEvent.click(button)
    })

    expect(mock.post).toHaveBeenCalledWith("/api/v1/posts/10/votes")
    expect(mock.post).toHaveBeenCalledTimes(1)
    expect(button).toHaveTextContent("6")
  })

  test("click when authenticated and hasVoted === true", async () => {
    post.hasVoted = true

    const mock = httpMock.alwaysOk()

    const { container } = render(
      <FiderContext.Provider value={fiderMock.authenticated()}>
        <VoteCounter post={post} />
      </FiderContext.Provider>
    )

    const button = container.querySelector("button") || fail("button not found")
    expect(button).toHaveTextContent("5")
    await act(async () => {
      fireEvent.click(button)
    })

    expect(mock.delete).toHaveBeenCalledWith("/api/v1/posts/10/votes")
    expect(mock.delete).toHaveBeenCalledTimes(1)
    expect(button).toHaveTextContent("4")
  })
})
