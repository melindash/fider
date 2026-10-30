-- Mage-OS fork. Idempotent so it is safe if the migration runner ever replays it.
ALTER TABLE mageos_anonymous_voters ADD COLUMN IF NOT EXISTS email TEXT NULL;

-- One shared author per tenant owns ideas submitted without an account.
ALTER TABLE users ADD COLUMN IF NOT EXISTS mageos_anonymous_author BOOLEAN NOT NULL DEFAULT false;
CREATE UNIQUE INDEX IF NOT EXISTS users_mageos_anonymous_author ON users (tenant_id) WHERE mageos_anonymous_author;

CREATE TABLE IF NOT EXISTS mageos_anonymous_posts (
  tenant_id          INT NOT NULL REFERENCES tenants (id),
  post_id            INT NOT NULL REFERENCES posts (id),
  voter_id           INT NOT NULL REFERENCES mageos_anonymous_voters (id),
  email              TEXT NOT NULL,
  created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  claimed_by_user_id INT NULL REFERENCES users (id),
  claimed_at         TIMESTAMPTZ NULL,
  PRIMARY KEY (tenant_id, post_id)
);
CREATE INDEX IF NOT EXISTS mageos_anonymous_posts_voter ON mageos_anonymous_posts (tenant_id, voter_id);
