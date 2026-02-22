#!/usr/bin/env bash
set -euo pipefail

# Reassign all default-user data to a real Google-authenticated user.
# See docs/data-backfill-runbook.md for full instructions.

usage() {
  cat <<EOF
Usage: $0 -t TARGET_USER_ID [-d DATABASE_URL] [--dry-run]

Reassigns all data owned by the seed default user
(00000000-0000-0000-0000-000000000001) to the specified Google user.

Options:
  -t    Target user UUID (must be an existing Google-authenticated user)
  -d    Database connection URL (default: \$DATABASE_URL)
  --dry-run  Execute inside a transaction then ROLLBACK (no changes persisted)

Examples:
  $0 -t 550e8400-e29b-41d4-a716-446655440000 --dry-run
  $0 -t 550e8400-e29b-41d4-a716-446655440000 -d 'postgres://shiru:shiru@localhost:5432/shiru?sslmode=disable'
EOF
  exit 1
}

TARGET_USER_ID=""
DB_URL="${DATABASE_URL:-}"
DRY_RUN=false

while [[ $# -gt 0 ]]; do
  case $1 in
    -t) TARGET_USER_ID="$2"; shift 2 ;;
    -d) DB_URL="$2"; shift 2 ;;
    --dry-run) DRY_RUN=true; shift ;;
    -h|--help) usage ;;
    *) echo "Unknown option: $1"; usage ;;
  esac
done

if [[ -z "$TARGET_USER_ID" ]]; then
  echo "ERROR: -t TARGET_USER_ID is required"
  usage
fi

if [[ -z "$DB_URL" ]]; then
  echo "ERROR: -d DATABASE_URL is required (or set DATABASE_URL env var)"
  usage
fi

uuid_re='^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$'
if ! [[ "$TARGET_USER_ID" =~ $uuid_re ]]; then
  echo "ERROR: invalid UUID format: $TARGET_USER_ID"
  exit 1
fi

COMMIT_OR_ROLLBACK="COMMIT"
if $DRY_RUN; then
  COMMIT_OR_ROLLBACK="ROLLBACK"
  echo "=== DRY RUN MODE — transaction will be rolled back ==="
  echo ""
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SQL_FILE="$SCRIPT_DIR/backfill-default-user.sql"

DEFAULT_USER_ID="00000000-0000-0000-0000-000000000001"
if [[ "$TARGET_USER_ID" == "$DEFAULT_USER_ID" ]]; then
  echo "ERROR: target user must not be the default user ($DEFAULT_USER_ID)"
  exit 1
fi

# Redact password from DB_URL for display (postgres://user:pass@host → postgres://user:***@host)
DISPLAY_URL=$(echo "$DB_URL" | sed -E 's|(://[^:]+:)[^@]+(@)|\1***\2|')

echo "Target user:  $TARGET_USER_ID"
echo "Database:     $DISPLAY_URL"
echo "Mode:         $(if $DRY_RUN; then echo 'dry-run'; else echo 'LIVE'; fi)"
echo ""

psql "$DB_URL" \
  -v target_user_id="$TARGET_USER_ID" \
  -v commit_or_rollback="$COMMIT_OR_ROLLBACK" \
  -f "$SQL_FILE"

if $DRY_RUN; then
  echo ""
  echo "Dry run complete — no changes were persisted."
else
  echo ""
  echo "Backfill complete — all default-user data reassigned to $TARGET_USER_ID."
fi
