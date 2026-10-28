#!/usr/bin/env node

// Generates the one-time SQL that converts the demo's placeholder "Historical GitHub vote NN
// (imported)" accounts into a per-post historical baseline (posts.mageos_historical_votes).
//
// The SQL runs in one transaction, refuses to run twice (mageos_data_migrations marker),
// and aborts if any post's total vote count would change or if the placeholder votes on a
// post differ from the manifest. Run it on a staging restore before production.
//
// Usage: node scripts/mageos/historical-votes-sql.mjs [manifest.json] > convert.sql
//        psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f convert.sql

import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const manifestPath = process.argv[2] || resolve(dirname(fileURLToPath(import.meta.url)), 'demo-import-manifest.json')
const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'))

const quote = (value) => `'${String(value).replaceAll("'", "''")}'`
const rows = manifest.posts
  .map((post) => `  (${Number(post.fiderPost)}, ${Number(post.historicalVotes)}, ${quote(post.githubURL)})`)
  .join(',\n')

process.stdout.write(`-- Generated from ${manifest.source} snapshot ${manifest.snapshotDate}: ${manifest.posts.length} posts, ${manifest.historicalVotes} historical votes.
BEGIN;

CREATE TABLE IF NOT EXISTS mageos_data_migrations (name TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now(), details JSONB);

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM mageos_data_migrations WHERE name = 'historical-github-votes') THEN
    RAISE EXCEPTION 'historical-github-votes was already applied; refusing to run twice';
  END IF;
END $$;

CREATE TEMP TABLE manifest (post_number INT PRIMARY KEY, historical_votes INT NOT NULL, source TEXT NOT NULL) ON COMMIT DROP;
INSERT INTO manifest VALUES
${rows};

CREATE TEMP TABLE placeholder_users ON COMMIT DROP AS
  SELECT id, tenant_id FROM users WHERE name ~ '^Historical GitHub vote [0-9]+ \\(imported\\)$';

CREATE TEMP TABLE before_totals ON COMMIT DROP AS
  SELECT p.id, p.number, p.mageos_historical_votes + COUNT(v.user_id) AS total
  FROM posts p LEFT JOIN post_votes v ON v.post_id = p.id AND v.tenant_id = p.tenant_id
  GROUP BY p.id;

DO $$
DECLARE mismatch TEXT;
BEGIN
  IF (SELECT COUNT(*) FROM placeholder_users) <> 10 THEN
    RAISE EXCEPTION 'expected 10 placeholder users, found %', (SELECT COUNT(*) FROM placeholder_users);
  END IF;
  SELECT string_agg(format('#%s manifest=%s placeholders=%s', m.post_number, m.historical_votes, COALESCE(c.n, 0)), ', ') INTO mismatch
  FROM manifest m
  JOIN posts p ON p.number = m.post_number AND p.tenant_id = (SELECT DISTINCT tenant_id FROM placeholder_users)
  LEFT JOIN (
    SELECT v.post_id, COUNT(*) AS n FROM post_votes v JOIN placeholder_users u ON u.id = v.user_id GROUP BY v.post_id
  ) c ON c.post_id = p.id
  WHERE COALESCE(c.n, 0) <> m.historical_votes;
  IF mismatch IS NOT NULL THEN
    RAISE EXCEPTION 'placeholder votes differ from the manifest: %', mismatch;
  END IF;
END $$;

UPDATE posts p
SET mageos_historical_votes = p.mageos_historical_votes + m.historical_votes,
    mageos_historical_source = m.source
FROM manifest m
WHERE p.number = m.post_number AND p.tenant_id = (SELECT DISTINCT tenant_id FROM placeholder_users);

DELETE FROM post_votes v USING placeholder_users u WHERE v.user_id = u.id AND v.tenant_id = u.tenant_id;

DO $$
DECLARE changed TEXT;
BEGIN
  SELECT string_agg(format('#%s %s->%s', b.number, b.total, a.total), ', ') INTO changed
  FROM before_totals b
  JOIN (
    SELECT p.id, p.mageos_historical_votes + COUNT(v.user_id) AS total
    FROM posts p LEFT JOIN post_votes v ON v.post_id = p.id AND v.tenant_id = p.tenant_id
    GROUP BY p.id
  ) a ON a.id = b.id
  WHERE a.total <> b.total;
  IF changed IS NOT NULL THEN
    RAISE EXCEPTION 'vote totals would change: %', changed;
  END IF;
END $$;

INSERT INTO mageos_data_migrations (name, details)
VALUES ('historical-github-votes', jsonb_build_object('posts', ${manifest.posts.length}, 'votes', ${Number(manifest.historicalVotes)}, 'snapshot', ${quote(manifest.snapshotDate)}));

COMMIT;
`)
