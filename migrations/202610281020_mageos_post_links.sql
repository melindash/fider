-- Mage-OS fork. Links from ideas to GitHub discussions, issues and pull requests.
CREATE TABLE IF NOT EXISTS mageos_post_links (
  id             SERIAL PRIMARY KEY,
  tenant_id      INT NOT NULL REFERENCES tenants (id),
  post_id        INT NOT NULL REFERENCES posts (id),
  kind           TEXT NOT NULL,
  url            TEXT NOT NULL,
  repo           TEXT NOT NULL,
  number         INT NOT NULL,
  is_source      BOOLEAN NOT NULL DEFAULT false,
  attribution    TEXT NULL,
  state          TEXT NULL,
  created_by_id  INT NULL REFERENCES users (id),
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_synced_at TIMESTAMPTZ NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS mageos_post_links_url ON mageos_post_links (tenant_id, post_id, url);
CREATE INDEX IF NOT EXISTS mageos_post_links_target ON mageos_post_links (tenant_id, repo, number, kind);

CREATE TABLE IF NOT EXISTS mageos_github_deliveries (
  delivery_id TEXT PRIMARY KEY,
  event       TEXT NOT NULL,
  received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  outcome     TEXT NULL
);
