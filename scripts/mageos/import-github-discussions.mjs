#!/usr/bin/env node

// Imports GitHub discussions into the board as ideas. Read-only towards GitHub.
//
// Each discussion becomes one idea with its original body, a credit line, a source link
// (isSource, attribution @author) and its GitHub upvotes as the separate historical baseline.
// Discussions already linked as a source are skipped, so the import is safe to re-run.
//
// Usage:
//   FIDER_URL=https://ideas.example FIDER_API_KEY=... \
//     node scripts/mageos/import-github-discussions.mjs [--repo mage-os-lab/.github] [--category Ideas] [--apply]
//
// Without --apply it only prints the plan. GitHub auth: GITHUB_TOKEN, else `gh auth token`.
// FIDER_API_KEY must belong to an administrator (Profile → API key).

import { execFileSync } from 'node:child_process'

const args = process.argv.slice(2)
const option = (name, fallback) => {
  const i = args.indexOf(name)
  return i >= 0 ? args[i + 1] : fallback
}
const apply = args.includes('--apply')
const [owner, repo] = option('--repo', 'mage-os-lab/.github').split('/')
const category = option('--category', 'Ideas')
const fiderURL = (process.env.FIDER_URL || '').replace(/\/$/, '')
const fiderKey = process.env.FIDER_API_KEY

if (!fiderURL || !fiderKey) {
  console.error('Set FIDER_URL and FIDER_API_KEY.')
  process.exit(2)
}

const githubToken = process.env.GITHUB_TOKEN || execFileSync('gh', ['auth', 'token'], { encoding: 'utf8' }).trim()

async function fetchDiscussions() {
  const discussions = []
  let after = null
  do {
    const response = await fetch('https://api.github.com/graphql', {
      method: 'POST',
      headers: { Authorization: `Bearer ${githubToken}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({
        query: `query($owner: String!, $repo: String!, $after: String) {
          repository(owner: $owner, name: $repo) {
            discussions(first: 50, after: $after, orderBy: {field: CREATED_AT, direction: ASC}) {
              pageInfo { hasNextPage endCursor }
              nodes { number title body url upvoteCount createdAt author { login } category { name } }
            }
          }
        }`,
        variables: { owner, repo, after },
      }),
    })
    const json = await response.json()
    if (!response.ok || json.errors) throw new Error(`GitHub query failed: ${JSON.stringify(json.errors || response.status)}`)
    const page = json.data.repository.discussions
    discussions.push(...page.nodes)
    after = page.pageInfo.hasNextPage ? page.pageInfo.endCursor : null
  } while (after)
  return discussions.filter((d) => !category || d.category?.name === category)
}

async function fider(method, path, body) {
  const response = await fetch(`${fiderURL}${path}`, {
    method,
    headers: { Authorization: `Bearer ${fiderKey}`, 'Content-Type': 'application/json', Accept: 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  const data = await response.json().catch(() => ({}))
  return { ok: response.ok, status: response.status, data }
}

function describe(d) {
  const login = d.author?.login || 'ghost'
  const date = d.createdAt.slice(0, 10)
  return `${d.body.trim()}\n\n---\n\nOriginally proposed by [@${login}](https://github.com/${login}) in [${owner} discussion #${d.number}](${d.url}) on ${date}. Votes from GitHub are counted separately from votes on this board.`
}

function titleFor(d) {
  const title = d.title.trim().replace(/\s+/g, ' ')
  return title.length >= 10 && title.length <= 100 ? title : `${title.slice(0, 85)} (GitHub #${d.number})`
}

async function importOne(d) {
  const existing = await fider('GET', `/api/v1/mageos/links?url=${encodeURIComponent(d.url)}`)
  if (!existing.ok) throw new Error(`lookup failed (${existing.status}) for ${d.url}`)
  if (existing.data.postNumber) return `skip   #${d.number} already imported as idea ${existing.data.postNumber}`

  const plan = `create #${d.number} "${titleFor(d)}" historical=${d.upvoteCount} by @${d.author?.login}`
  if (!apply) return plan

  let created = await fider('POST', '/api/v1/posts', { title: titleFor(d), description: describe(d) })
  if (!created.ok && created.data.errors?.some((e) => e.field === 'title')) {
    created = await fider('POST', '/api/v1/posts', { title: `${d.title.trim().slice(0, 85)} (GitHub #${d.number})`, description: describe(d) })
  }
  if (!created.ok) throw new Error(`create failed for ${d.url}: ${JSON.stringify(created.data)}`)
  const number = created.data.number

  // Link first: the link is what makes a re-run skip this discussion.
  const link = await fider('POST', `/api/v1/posts/${number}/links`, { url: d.url, isSource: true, attribution: `@${d.author?.login || 'ghost'}` })
  if (!link.ok) throw new Error(`idea ${number} created but source link failed; add it by hand before re-running: ${JSON.stringify(link.data)}`)

  // Fider votes for a post's author automatically; the importing admin is not a supporter.
  const unvote = await fider('DELETE', `/api/v1/posts/${number}/votes`)
  if (!unvote.ok) throw new Error(`idea ${number}: could not remove the importer's automatic vote`)

  const votes = await fider('PUT', `/api/v1/posts/${number}/historical-votes`, { votes: d.upvoteCount, source: d.url })
  if (!votes.ok) throw new Error(`idea ${number}: could not set historical votes: ${JSON.stringify(votes.data)}`)

  return `${plan} -> idea ${number}`
}

const discussions = await fetchDiscussions()
console.log(`${discussions.length} discussions in ${owner}/${repo}${category ? ` (category ${category})` : ''}; ${apply ? 'APPLYING' : 'dry run'}`)
let failed = 0
for (const d of discussions) {
  try {
    console.log(await importOne(d))
  } catch (error) {
    failed++
    console.error(`ERROR  #${d.number}: ${error.message}`)
    if (apply) break
  }
}
const historical = discussions.reduce((sum, d) => sum + d.upvoteCount, 0)
console.log(`historical votes in source: ${historical}; errors: ${failed}`)
process.exit(failed ? 1 : 0)
