-- Mage-OS fork: roadmap review tool. Idempotent so it is safe if the migration runner ever replays it.
-- Ideas are never copied here: linked candidates point at posts and read title, votes and status live.
CREATE TABLE IF NOT EXISTS mageos_review_candidates (
  id              SERIAL PRIMARY KEY,
  tenant_id       INT NOT NULL REFERENCES tenants (id),
  post_id         INT NULL REFERENCES posts (id),
  title           TEXT NOT NULL DEFAULT '',
  source_url      TEXT NOT NULL DEFAULT '',
  track           TEXT NOT NULL DEFAULT 'product',
  stage           TEXT NOT NULL DEFAULT 'new',
  delivery        TEXT NOT NULL DEFAULT 'none',
  problem         TEXT NOT NULL DEFAULT '',
  beneficiaries   TEXT NOT NULL DEFAULT '',
  outcome         TEXT NOT NULL DEFAULT '',
  scope           TEXT NOT NULL DEFAULT '',
  exclusions      TEXT NOT NULL DEFAULT '',
  evidence        TEXT NOT NULL DEFAULT '',
  dependencies    TEXT NOT NULL DEFAULT '',
  risks           TEXT NOT NULL DEFAULT '',
  effort          TEXT NOT NULL DEFAULT '',
  owner           TEXT NOT NULL DEFAULT '',
  reviewer        TEXT NOT NULL DEFAULT '',
  operating_needs TEXT NOT NULL DEFAULT '',
  open_questions  TEXT NOT NULL DEFAULT '',
  links           TEXT NOT NULL DEFAULT '',
  is_sample       BOOLEAN NOT NULL DEFAULT false,
  created_by      INT NOT NULL REFERENCES users (id),
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS mageos_review_candidates_post ON mageos_review_candidates (tenant_id, post_id) WHERE post_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS mageos_review_assessments (
  tenant_id    INT NOT NULL REFERENCES tenants (id),
  candidate_id INT NOT NULL REFERENCES mageos_review_candidates (id) ON DELETE CASCADE,
  user_id      INT NOT NULL REFERENCES users (id),
  criterion    TEXT NOT NULL,
  rating       TEXT NOT NULL,
  note         TEXT NOT NULL DEFAULT '',
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, candidate_id, user_id, criterion)
);

CREATE TABLE IF NOT EXISTS mageos_review_meetings (
  id         SERIAL PRIMARY KEY,
  tenant_id  INT NOT NULL REFERENCES tenants (id),
  held_on    DATE NOT NULL,
  purpose    TEXT NOT NULL DEFAULT '',
  is_sample  BOOLEAN NOT NULL DEFAULT false,
  created_by INT NOT NULL REFERENCES users (id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS mageos_review_items (
  id                 SERIAL PRIMARY KEY,
  tenant_id          INT NOT NULL REFERENCES tenants (id),
  meeting_id         INT NOT NULL REFERENCES mageos_review_meetings (id) ON DELETE CASCADE,
  candidate_id       INT NOT NULL REFERENCES mageos_review_candidates (id) ON DELETE CASCADE,
  position           INT NOT NULL DEFAULT 0,
  decision_requested TEXT NOT NULL DEFAULT '',
  notes              TEXT NOT NULL DEFAULT '',
  outcome            TEXT NOT NULL DEFAULT '',
  rationale          TEXT NOT NULL DEFAULT '',
  objections         TEXT NOT NULL DEFAULT '',
  unresolved         TEXT NOT NULL DEFAULT '',
  escalation         TEXT NOT NULL DEFAULT 'none',
  finalized_by       INT NULL REFERENCES users (id),
  finalized_at       TIMESTAMPTZ NULL,
  UNIQUE (meeting_id, candidate_id)
);

CREATE TABLE IF NOT EXISTS mageos_review_actions (
  id           SERIAL PRIMARY KEY,
  tenant_id    INT NOT NULL REFERENCES tenants (id),
  candidate_id INT NOT NULL REFERENCES mageos_review_candidates (id) ON DELETE CASCADE,
  item_id      INT NULL REFERENCES mageos_review_items (id) ON DELETE SET NULL,
  owner        TEXT NOT NULL DEFAULT '',
  next_step    TEXT NOT NULL,
  review_on    DATE NULL,
  done_at      TIMESTAMPTZ NULL,
  created_by   INT NOT NULL REFERENCES users (id),
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS mageos_review_actions_candidate ON mageos_review_actions (tenant_id, candidate_id);

-- No foreign key to the changed row, so history survives when a sample candidate is removed.
CREATE TABLE IF NOT EXISTS mageos_review_events (
  id           SERIAL PRIMARY KEY,
  tenant_id    INT NOT NULL REFERENCES tenants (id),
  candidate_id INT NULL,
  meeting_id   INT NULL,
  user_id      INT NOT NULL REFERENCES users (id),
  entity       TEXT NOT NULL,
  entity_id    INT NOT NULL,
  action       TEXT NOT NULL,
  changes      JSONB NOT NULL DEFAULT '{}',
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS mageos_review_events_candidate ON mageos_review_events (tenant_id, candidate_id, created_at);
CREATE INDEX IF NOT EXISTS mageos_review_events_meeting ON mageos_review_events (tenant_id, meeting_id, created_at);
