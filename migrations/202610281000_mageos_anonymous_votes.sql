-- Mage-OS fork. Idempotent so it is safe if the migration runner ever replays it.
CREATE TABLE IF NOT EXISTS mageos_anonymous_voters (
  id                  SERIAL PRIMARY KEY,
  tenant_id           INT NOT NULL REFERENCES tenants (id),
  token_hash          TEXT NOT NULL,
  roles               TEXT[] NOT NULL DEFAULT '{}',
  roles_prompted_at   TIMESTAMPTZ NULL,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_seen_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  merged_into_user_id INT NULL REFERENCES users (id),
  merged_at           TIMESTAMPTZ NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS mageos_anonymous_voters_token ON mageos_anonymous_voters (tenant_id, token_hash);

CREATE TABLE IF NOT EXISTS mageos_anonymous_votes (
  tenant_id    INT NOT NULL REFERENCES tenants (id),
  post_id      INT NOT NULL REFERENCES posts (id),
  voter_id     INT NOT NULL REFERENCES mageos_anonymous_voters (id),
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  network_hash TEXT NULL,
  PRIMARY KEY (tenant_id, post_id, voter_id)
);
CREATE INDEX IF NOT EXISTS mageos_anonymous_votes_post ON mageos_anonymous_votes (tenant_id, post_id);
CREATE INDEX IF NOT EXISTS mageos_anonymous_votes_network ON mageos_anonymous_votes (tenant_id, network_hash, created_at);

ALTER TABLE users ADD COLUMN IF NOT EXISTS mageos_roles TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE users ADD COLUMN IF NOT EXISTS mageos_roles_prompted_at TIMESTAMPTZ NULL;
