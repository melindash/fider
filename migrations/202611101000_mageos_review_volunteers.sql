-- Mage-OS fork: volunteers on a review candidate. Each one is a Fider account, a GitHub username, or both.
CREATE TABLE IF NOT EXISTS mageos_review_volunteers (
  id           SERIAL PRIMARY KEY,
  tenant_id    INT NOT NULL REFERENCES tenants (id),
  candidate_id INT NOT NULL REFERENCES mageos_review_candidates (id) ON DELETE CASCADE,
  user_id      INT NULL REFERENCES users (id),
  github_login TEXT NULL,
  created_by   INT NOT NULL REFERENCES users (id),
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT mageos_review_volunteers_identity CHECK (user_id IS NOT NULL OR github_login IS NOT NULL)
);
CREATE UNIQUE INDEX IF NOT EXISTS mageos_review_volunteers_user ON mageos_review_volunteers (candidate_id, user_id) WHERE user_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS mageos_review_volunteers_github ON mageos_review_volunteers (candidate_id, lower(github_login)) WHERE github_login IS NOT NULL;
