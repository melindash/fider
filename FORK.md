# Mage-OS Fider fork

Fork of [getfider/fider](https://github.com/getfider/fider) for the Mage-OS ideas board. Branch `mage-os`, based on upstream `v0.38.2` (`e492af4a`).

## Customizations

| Feature | Where | Notes |
|---|---|---|
| **Under review** status (`underreview`, id 7) | `app/models/enum/post_status.go`, `public/models/post.ts`, `ShowPostResponse.tsx` | Sits between Open and Planned. Not shown on the roadmap. |
| Anonymous voting | `app/pkg/mageos/`, `*mageos*.go`, `app/middlewares/mageos_voter.go`, `public/hooks/use-mageos-vote.ts`, `public/components/mageos/` | Cookie `mageos_voter`; DB stores SHA-256 only. Unique (post, voter) prevents duplicate votes. |
| Rate limits and abuse tools | `app/pkg/mageos/ratelimit.go`, `app/handlers/apiv1/mageos.go` | Per-network limits; admin burst report and void endpoint. |
| Sign-in merge | `app/middlewares/mageos_voter.go` | On the first signed-in request carrying a voter cookie: votes and roles move to the account, duplicates collapse. |
| Participant roles | `ParticipantRolesModal.tsx`, `users.mageos_roles`, `mageos_anonymous_voters.roles` | Multi-select, optional, prompted once, editable from any idea page. Never weights votes. |
| Historical GitHub votes | `posts.mageos_historical_votes`, `mageos_historical_source` | Set per idea when importing GitHub discussions. Counted in totals and ranking; shown separately on the idea page. |
| GitHub discussion import | `scripts/mageos/import-github-discussions.mjs`, `PUT /api/v1/posts/:number/historical-votes`, `GET /api/v1/mageos/links?url=` | Dry run by default; `--apply` to write. Skips discussions already linked as a source. Read-only towards GitHub. |
| GitHub links and webhook | `app/handlers/apiv1/mageos_links.go`, `app/handlers/webhooks/mageos_github.go`, `PostLinks.tsx` | Read-only towards GitHub. |
| Accessible vote button | `public/components/mageos/VoteCounter.tsx` | `aria-label`, `aria-pressed`. |
| Dark theme by default | `views/base.html`, `public/components/ThemeSwitcher.tsx`, `app/pkg/web/testdata/*.html` | Fallback theme is `dark`; a visitor's saved choice still wins. The renderer fixtures embed the base template script, so they change with it. |
| Out-of-order migrations | `app/pkg/dbx/migrate_mageos.go` | Applies every migration missing from `migrations_history`. Upstream applies only versions above the newest applied one, which would skip upstream migrations dated before a fork migration. |

Upstream files touched only at integration seams: `routes.go` (middleware, webhook route, `mageosRoutes(r)`), `server.go` and `postgres.go` (registration), `migrate.go` (one condition), `post.go` (vote aggregation and `has_voted`), `common.go` (status filters), `actions/post.go` (status validation), entity/db Post structs, `post_status.go`, `public/components/index.tsx` and `PostDetails.tsx` (point at `components/mageos/VoteCounter` and `VoteSection`), `ShowPostResponse.tsx`, `models/post.ts`, hook and action index files. Upstream `VoteCounter.tsx` and `VoteSection.tsx` are kept unchanged and unused.

## Configuration

| Variable | Default | Purpose |
|---|---|---|
| `MAGEOS_ANONYMOUS_VOTING` | `true` | `false` turns anonymous voting off (abuse kill switch). Existing votes stay. |
| `MAGEOS_CLIENT_IP_HEADER` | empty | Header holding the client IP. Set `CF-Connecting-IP` behind Cloudflare Tunnel. Only set it when nothing else can reach the app. |
| `MAGEOS_NEW_VOTERS_PER_NETWORK_PER_HOUR` | 60 | New anonymous voters per network (IPv4 /24, IPv6 /48). |
| `MAGEOS_VOTES_PER_NETWORK_PER_HOUR` | 600 | Anonymous votes per network. |
| `MAGEOS_VOTES_PER_VOTER_PER_MINUTE` | 30 | Votes per anonymous voter. |
| `MAGEOS_VOTES_PER_NETWORK_PER_POST_PER_DAY` | 25 | Anonymous votes on one idea from one network per 24 h. Stored in the database; survives restarts. |
| `MAGEOS_GITHUB_WEBHOOK_SECRET` | empty | Enables `POST /webhooks/github`. Empty = endpoint returns 404. |
| `MAGEOS_GITHUB_AUTO_STATUS` | `false` | Allows Planned → Started when a linked issue or PR opens. |

## Anonymous voting: what it does and does not prevent

| Prevents | Does not prevent |
|---|---|
| Same browser voting twice on an idea (database unique key) | Repeat votes after clearing cookies, private windows or another browser |
| Scripted floods from one network (per-network limits and daily per-idea cap) | Distributed abuse across many networks |
| Double counting when an anonymous voter signs in | One person voting anonymously on one device and signed in on another |

- An IP address is never treated as a person. Limits apply per network block and are set high enough for offices and universities. Over a limit, the visitor is asked to sign in.
- No fingerprinting. Stored per anonymous vote: voter ID, post, time, and a keyed hash of the network block. The network hash is cleared after 30 days by `MageOSPurgeNetworkHashesJob`.
- Abuse response: `GET /api/v1/mageos/vote-bursts?hours=24&min=10` (collaborators) lists clusters; `POST /api/v1/mageos/vote-bursts/void` (administrators) deletes one cluster's anonymous votes. Signed-in votes are never touched. Set `MAGEOS_ANONYMOUS_VOTING=false` for a sustained attack.
- Role counts: `GET /api/v1/mageos/participation[?post=N]` (collaborators). Totals only. Multi-role participants are counted in each group.

## GitHub status mapping

| GitHub event on a linked item | Link state | Idea status change |
|---|---|---|
| Issue or PR opened / reopened | `open` | Planned → Started, only if `MAGEOS_GITHUB_AUTO_STATUS=true` |
| PR merged | `merged` | None |
| Issue closed, PR closed unmerged | `closed` | None |
| Discussion links | not synced | None |

Completed and Declined are always set by a maintainer. Webhook deliveries are verified with `X-Hub-Signature-256`, deduplicated by `X-GitHub-Delivery` (a failed delivery rolls back and can be retried), and logged with their outcome in `mageos_github_deliveries`. Fider never writes to GitHub.

GitHub webhook setup (after approval): repository or organization → Settings → Webhooks → payload URL `https://<board>/webhooks/github`, content type `application/json`, secret = `MAGEOS_GITHUB_WEBHOOK_SECRET`, events **Issues** and **Pull requests** only. No GitHub token is needed.

## Updating from upstream

`.github/workflows/mageos-upstream.yml` trial-merges the latest upstream release and upstream `main` into `mage-os` every Monday, on every push to `mage-os`, and on demand, then runs UI lint, UI tests and server tests on the result. It pushes nothing. A red `latest-release` run means the next merge needs work; `main` is an early warning and never fails the workflow. The step summary lists conflicting files.

1. `git fetch upstream --tags && git switch mage-os && git merge <upstream-tag>`.
2. Resolve conflicts at the seams listed above.
3. Run tests the way the Makefile does: `docker compose up -d pgtest`, `make build-server build-ssr build-ui`, `godotenv -f .test.env ./fider migrate`, `godotenv -f .test.env go test ./... -short`, `TZ=GMT npx jest ./public`, `make lint`.
4. Rehearse on a staging restore of production before deploying. Record old and new image tags.

Rules that keep merges small:

- Put fork code in fork-owned files (`*mageos*`, `mageos/` directories). Edit upstream files only to call into them.
- Never run `gofmt -w` on a directory; format only files you changed. Newer gofmt versions realign upstream files.
- Add fork fields at the end of upstream structs after a blank line, so alignment of upstream lines does not change.
- Fork migrations are named `*_mageos_*.sql` and must be idempotent.
- Lingui extraction adds new message IDs to every `locale/*/client.json`. Commit those changes.

## Known test environment gap

`app/services/blob` S3 tests need the `minio/minio:RELEASE.2021-07-27…` image, which is no longer pullable. They fail locally before and after the fork changes; production uses `BLOB_STORAGE=sql`.
