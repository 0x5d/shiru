-- Backfill script: reassign all default-user data to a real Google user.
--
-- Required psql variables:
--   target_user_id   UUID of the target Google user
--   commit_or_rollback   COMMIT or ROLLBACK (for dry-run support)
--
-- Usage: psql -v target_user_id='<UUID>' -v commit_or_rollback='COMMIT' -f backfill-default-user.sql

\set ON_ERROR_STOP on

BEGIN;

-- Store params in a temp table so DO blocks (PL/pgSQL) can access them.
CREATE TEMP TABLE _bp (default_uid uuid, target_uid uuid);
INSERT INTO _bp VALUES (
  '00000000-0000-0000-0000-000000000001',
  :'target_user_id'
);

-- Validate target user exists and is a Google-authenticated user.
DO $$
DECLARE
  v_target uuid;
  v_sub text;
BEGIN
  SELECT target_uid INTO v_target FROM _bp;
  SELECT google_sub INTO v_sub FROM users WHERE id = v_target;
  IF NOT FOUND THEN
    RAISE EXCEPTION 'target user % does not exist', v_target;
  END IF;
  IF v_sub IS NULL THEN
    RAISE EXCEPTION 'target user % has no google_sub (not a Google user)', v_target;
  END IF;
  RAISE NOTICE 'validated target user %', v_target;
END;
$$;

-- ── Step 1: Merge conflicting tags ──────────────────────────────────────────
-- If the target user already has a tag with the same name as a default-user
-- tag, remap vocab_entry_tags to the target's tag, then delete the duplicate.

INSERT INTO vocab_entry_tags (vocab_entry_id, tag_id, rank)
SELECT vet.vocab_entry_id, tt.id, vet.rank
FROM vocab_entry_tags vet
JOIN tags dt ON dt.id = vet.tag_id
JOIN tags tt ON tt.name = dt.name AND tt.user_id = (SELECT target_uid FROM _bp)
WHERE dt.user_id = (SELECT default_uid FROM _bp)
ON CONFLICT DO NOTHING;

DELETE FROM vocab_entry_tags vet
USING tags dt
WHERE vet.tag_id = dt.id
AND dt.user_id = (SELECT default_uid FROM _bp)
AND dt.name IN (SELECT name FROM tags WHERE user_id = (SELECT target_uid FROM _bp));

DELETE FROM tags
WHERE user_id = (SELECT default_uid FROM _bp)
AND name IN (SELECT name FROM tags WHERE user_id = (SELECT target_uid FROM _bp));

-- ── Step 2: Merge conflicting vocab entries ─────────────────────────────────
-- If the target user already has a vocab entry with the same normalized_surface,
-- remap story_vocab_entries and vocab_entry_tags, then delete the duplicate.

-- Remap story_vocab_entries for conflicting vocab
INSERT INTO story_vocab_entries (story_id, vocab_entry_id)
SELECT sve.story_id, tv.id
FROM story_vocab_entries sve
JOIN vocab_entries dv ON dv.id = sve.vocab_entry_id
JOIN vocab_entries tv ON tv.normalized_surface = dv.normalized_surface
  AND tv.user_id = (SELECT target_uid FROM _bp)
WHERE dv.user_id = (SELECT default_uid FROM _bp)
ON CONFLICT DO NOTHING;

DELETE FROM story_vocab_entries sve
USING vocab_entries dv
WHERE sve.vocab_entry_id = dv.id
AND dv.user_id = (SELECT default_uid FROM _bp)
AND dv.normalized_surface IN (
  SELECT normalized_surface FROM vocab_entries
  WHERE user_id = (SELECT target_uid FROM _bp)
);

-- Remap vocab_entry_tags for conflicting vocab
INSERT INTO vocab_entry_tags (vocab_entry_id, tag_id, rank)
SELECT tv.id, vet.tag_id, vet.rank
FROM vocab_entry_tags vet
JOIN vocab_entries dv ON dv.id = vet.vocab_entry_id
JOIN vocab_entries tv ON tv.normalized_surface = dv.normalized_surface
  AND tv.user_id = (SELECT target_uid FROM _bp)
WHERE dv.user_id = (SELECT default_uid FROM _bp)
ON CONFLICT DO NOTHING;

DELETE FROM vocab_entry_tags vet
USING vocab_entries dv
WHERE vet.vocab_entry_id = dv.id
AND dv.user_id = (SELECT default_uid FROM _bp)
AND dv.normalized_surface IN (
  SELECT normalized_surface FROM vocab_entries
  WHERE user_id = (SELECT target_uid FROM _bp)
);

-- Delete duplicate vocab entries (CASCADE removes any remaining links)
DELETE FROM vocab_entries
WHERE user_id = (SELECT default_uid FROM _bp)
AND normalized_surface IN (
  SELECT normalized_surface FROM vocab_entries
  WHERE user_id = (SELECT target_uid FROM _bp)
);

-- ── Step 3: Reassign remaining non-conflicting records ──────────────────────

UPDATE tags
SET user_id = (SELECT target_uid FROM _bp)
WHERE user_id = (SELECT default_uid FROM _bp);

UPDATE vocab_entries
SET user_id = (SELECT target_uid FROM _bp)
WHERE user_id = (SELECT default_uid FROM _bp);

UPDATE stories
SET user_id = (SELECT target_uid FROM _bp)
WHERE user_id = (SELECT default_uid FROM _bp);

UPDATE topic_snapshots
SET user_id = (SELECT target_uid FROM _bp)
WHERE user_id = (SELECT default_uid FROM _bp);

-- ── Step 4: Merge user_settings ─────────────────────────────────────────────
-- The default user's settings were the "real" settings; the target user's
-- settings were auto-created with defaults on first Google login.
-- Overwrite the target's settings with the default user's values.

INSERT INTO user_settings (user_id, jlpt_level, story_word_target, wanikani_api_key, wanikani_last_synced_at)
SELECT (SELECT target_uid FROM _bp), jlpt_level, story_word_target, wanikani_api_key, wanikani_last_synced_at
FROM user_settings
WHERE user_id = (SELECT default_uid FROM _bp)
ON CONFLICT (user_id) DO UPDATE SET
  jlpt_level = EXCLUDED.jlpt_level,
  story_word_target = EXCLUDED.story_word_target,
  wanikani_api_key = EXCLUDED.wanikani_api_key,
  wanikani_last_synced_at = EXCLUDED.wanikani_last_synced_at;

DELETE FROM user_settings WHERE user_id = (SELECT default_uid FROM _bp);

-- ── Step 5: Delete default user ─────────────────────────────────────────────

DELETE FROM users WHERE id = (SELECT default_uid FROM _bp);

-- ── Cleanup ─────────────────────────────────────────────────────────────────

DROP TABLE _bp;

:commit_or_rollback;
