#!/usr/bin/env bash
# Fider applies only migrations newer than the database's last version. An upstream
# migration dated earlier than our newest fork migration would therefore be skipped
# silently on any database that already ran the fork. Run after every upstream merge.
set -euo pipefail
cd "$(dirname "$0")/../.."

newest_fork=$(ls migrations | grep '_mageos_' | sort | tail -1 | cut -c1-12)
late=$(git diff --name-only --diff-filter=A "${1:-ORIG_HEAD}" HEAD -- migrations \
  | xargs -r -n1 basename | { grep -v '_mageos_' || true; } | awk -v n="$newest_fork" 'substr($0,1,12) < n')

if [ -n "$late" ]; then
  echo "These upstream migrations sort before fork migration $newest_fork and would be skipped:" >&2
  echo "$late" >&2
  echo "Databases that already ran the fork will never apply them. Add a fork migration numbered" >&2
  echo "after $newest_fork that re-applies each one idempotently (IF NOT EXISTS / guarded DO blocks)," >&2
  echo "keep the upstream files for fresh installs, and make sure running both is safe." >&2
  exit 1
fi
echo "migration order ok (newest fork migration $newest_fork)"
