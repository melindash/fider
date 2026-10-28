#!/usr/bin/env bash
# Nightly encrypted database backup. BLOB_STORAGE=sql keeps uploads in the database,
# so the dump is a complete backup. Run from the deployment directory.
set -euo pipefail
cd "$(dirname "$0")"
set -a; . ./.env; set +a

umask 077
mkdir -p backups
stamp=$(date -u +%Y%m%dT%H%M%SZ)
out="backups/fider-${stamp}.dump.age"

docker compose exec -T db pg_dump -U fider -d fider --format=custom --no-owner --no-acl \
  | age -r "$BACKUP_AGE_RECIPIENT" > "$out.partial"
test -s "$out.partial"
mv "$out.partial" "$out"

# Keep 14 local copies. Off-host copy: see README "Backups".
ls -1t backups/fider-*.dump.age | tail -n +15 | xargs -r rm --
echo "backup ok: $out"
