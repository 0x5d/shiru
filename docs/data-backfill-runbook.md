# Data Backfill Runbook

## Purpose

The initial Shiru deployment used a hard-coded seed user (`00000000-0000-0000-0000-000000000001`) for all data. After enabling Google authentication, existing data must be reassigned to a real Google-authenticated user.

This is a **one-time** operation per deployment.

## Prerequisites

1. PostgreSQL client (`psql`) installed.
2. The target user has already logged in via Google at least once (their `users` row must have a non-null `google_sub`).
3. Database connection URL available.
4. **Stop the application** before running the migration. The script does not lock tables; concurrent writes from the running app could cause partial or inconsistent results.
5. **Take a database backup** before running the live migration:

```bash
pg_dump "$DATABASE_URL" > shiru-pre-backfill-$(date +%Y%m%d%H%M%S).sql
```

## Finding the Target User ID

Query the database for the user you want to receive the data:

```sql
SELECT id, email, name, google_sub FROM users WHERE google_sub IS NOT NULL;
```

## Pre-Flight Verification

Check how much data will be moved:

```sql
SELECT 'vocab_entries' AS tbl, count(*) FROM vocab_entries WHERE user_id = '00000000-0000-0000-0000-000000000001'
UNION ALL
SELECT 'stories', count(*) FROM stories WHERE user_id = '00000000-0000-0000-0000-000000000001'
UNION ALL
SELECT 'tags', count(*) FROM tags WHERE user_id = '00000000-0000-0000-0000-000000000001'
UNION ALL
SELECT 'topic_snapshots', count(*) FROM topic_snapshots WHERE user_id = '00000000-0000-0000-0000-000000000001'
UNION ALL
SELECT 'user_settings', count(*) FROM user_settings WHERE user_id = '00000000-0000-0000-0000-000000000001';
```

Record the counts so you can verify the target user's totals after migration.

## Dry Run

Always perform a dry run first. The script runs the full migration inside a transaction, then rolls back:

```bash
./scripts/backfill-default-user.sh \
  -t <TARGET_USER_UUID> \
  -d 'postgres://shiru:shiru@localhost:5432/shiru?sslmode=disable' \
  --dry-run
```

Verify the output shows no errors and the NOTICE messages confirm validation passed.

## Live Run

```bash
./scripts/backfill-default-user.sh \
  -t <TARGET_USER_UUID> \
  -d 'postgres://shiru:shiru@localhost:5432/shiru?sslmode=disable'
```

## What the Script Does

1. **Validates** the target user exists, has a `google_sub`, and is not the default user. Also checks the default user still exists (catches accidental re-runs).
2. **Merges conflicting tags** — if the target user already has a tag with the same name, remaps `vocab_entry_tags` references and deletes the duplicate.
3. **Merges conflicting vocab entries** — if the target user already has a vocab entry with the same `normalized_surface`, remaps `story_vocab_entries` and `vocab_entry_tags` references, fills missing `meaning`/`reading` from the default user's entry, and deletes the duplicate.
4. **Reassigns remaining records** — updates `user_id` on `tags`, `vocab_entries`, `stories`, and `topic_snapshots`.
5. **Merges user settings** — overwrites the target user's settings with the default user's customized values (JLPT level, word target, WaniKani key).
6. **Deletes the default user** row and its settings.

All steps run inside a single transaction.

## Merge Policy

When both users have the same vocab entry or tag, the **target user's row is kept** and the default user's duplicate is deleted. For vocab entries, `meaning` and `reading` are preserved from whichever row has a non-NULL value (target takes precedence via `COALESCE`).

## Tables Affected

| Table | Action |
|---|---|
| `tags` | Reassign or merge by name |
| `vocab_entries` | Reassign or merge by `normalized_surface` |
| `vocab_entry_tags` | Remap during merge, then cascade |
| `story_vocab_entries` | Remap during merge |
| `stories` | Reassign `user_id` |
| `topic_snapshots` | Reassign `user_id` |
| `user_settings` | Merge into target, delete default |
| `users` | Delete default user row |

`story_audio` is not affected — it references `stories(id)` which keeps its primary key.

## Rollback

If the live run completes but the result is wrong, restore from the backup taken before running:

```bash
psql "$DATABASE_URL" < shiru-pre-backfill-YYYYMMDDHHMMSS.sql
```

**There is no automatic reverse migration.** The merge steps (deduplication of tags and vocab) are lossy — the default user's duplicate entries are deleted in favor of the target user's existing entries. A backup is the only safe rollback path.

## Post-Migration Verification

```sql
-- Confirm no data remains under the default user
SELECT 'vocab_entries' AS tbl, count(*) FROM vocab_entries WHERE user_id = '00000000-0000-0000-0000-000000000001'
UNION ALL
SELECT 'stories', count(*) FROM stories WHERE user_id = '00000000-0000-0000-0000-000000000001'
UNION ALL
SELECT 'tags', count(*) FROM tags WHERE user_id = '00000000-0000-0000-0000-000000000001'
UNION ALL
SELECT 'topic_snapshots', count(*) FROM topic_snapshots WHERE user_id = '00000000-0000-0000-0000-000000000001'
UNION ALL
SELECT 'user_settings', count(*) FROM user_settings WHERE user_id = '00000000-0000-0000-0000-000000000001'
UNION ALL
SELECT 'users', count(*) FROM users WHERE id = '00000000-0000-0000-0000-000000000001';

-- Confirm target user owns the data (counts should match pre-flight + any prior target data)
SELECT 'vocab_entries' AS tbl, count(*) FROM vocab_entries WHERE user_id = '<TARGET_USER_UUID>'
UNION ALL
SELECT 'stories', count(*) FROM stories WHERE user_id = '<TARGET_USER_UUID>'
UNION ALL
SELECT 'tags', count(*) FROM tags WHERE user_id = '<TARGET_USER_UUID>';
```

After verifying, restart the application.
