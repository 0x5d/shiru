## Shiru MVP Engineering Plan

This plan turns the product spec into an implementation-ready MVP for the current repository.

## 1) Scope

- No authentication/authorization in MVP.
- Single-user mode in backend (`default_user`) to keep architecture ready for future multi-user support.
Core flows:
- Add/import vocab.
- Generate topics.
- Generate story from topic + user settings + vocab-derived tags.
- Read story with word interactions.
- Play cached TTS audio.
- Search stories by full text.

## 2) High-Level Architecture

`Go API`:
- REST endpoints.
- Orchestration for Anthropic, WaniKani, dictionary API, and ElevenLabs.
- Persistence in Postgres.
- Full-text indexing in Elasticsearch.

`Postgres`:
- Source of truth for users/settings/vocab/stories/jobs.

`Elasticsearch`:
- Story full-text retrieval.
- Topic/tag-assisted query support (optional in MVP, keep mapping extensible).

`React app`:
- Home (topics, generation, story reading).
- Settings (vocab, WaniKani, JLPT level, story length).

## 3) Data Model (Postgres)

All tables include `created_at`, `updated_at` (`timestamptz`).

### `users`

- `id uuid pk`
- `handle text unique not null` (seed with `default_user`)

### `user_settings`

- `user_id uuid pk references users(id)`
- `jlpt_level text not null check (jlpt_level in ('N5','N4','N3','N2','N1')) default 'N5'`
- `story_word_target int not null default 100 check (story_word_target between 50 and 500)`
- `wanikani_api_key text null`
- `wanikani_last_synced_at timestamptz null`

### `vocab_entries`

- `id uuid pk`
- `user_id uuid not null references users(id)`
- `surface text not null` (word/phrase as entered)
- `normalized_surface text not null` (NFKC/lower normalization for dedupe)
- `meaning text null` (dictionary result cache)
- `reading text null` (furigana/reading cache)
- `source text not null check (source in ('manual','wanikani'))`
- `source_ref text null` (e.g., wani item/subject id)
- Unique key: `(user_id, normalized_surface)` to enforce merged duplicates.

### `tags`

- `id uuid pk`
- `user_id uuid not null references users(id)`
- `name text not null`
- Unique key: `(user_id, name)`

### `vocab_entry_tags`

- `vocab_entry_id uuid not null references vocab_entries(id) on delete cascade`
- `tag_id uuid not null references tags(id) on delete cascade`
- `rank smallint not null check (rank between 1 and 3)`
- PK: `(vocab_entry_id, tag_id)`

### `stories`

- `id uuid pk`
- `user_id uuid not null references users(id)`
- `topic text not null`
- `tone text not null check (tone in ('funny','shocking'))`
- `jlpt_level text not null`
- `target_word_count int not null`
- `actual_word_count int not null`
- `content text not null`
- `used_vocab_count int not null`
- `source_tag_names text[] not null default '{}'`

### `story_vocab_entries`

- `story_id uuid not null references stories(id) on delete cascade`
- `vocab_entry_id uuid not null references vocab_entries(id) on delete cascade`
- PK: `(story_id, vocab_entry_id)`

### `story_audio`

- `story_id uuid pk references stories(id) on delete cascade`
- `voice_id text not null`
- `audio_format text not null` (e.g., `mp3`)
- `storage_path text not null` (local/object storage path)
- `duration_ms int null`
- `checksum text not null`

### `topic_snapshots`

- `id uuid pk`
- `user_id uuid not null references users(id)`
- `topics text[] not null`
- `prompt_version text not null`

## 4) Elasticsearch Indexes

### `stories_v1`

- `story_id keyword`
- `user_id keyword`
- `topic text`
- `tone keyword`
- `content text` (Japanese analyzer)
- `jlpt_level keyword`
- `created_at date`

Index operation:
- Upsert on story creation.
- Delete on story deletion (if deletion is added later).

## 5) External Integrations

### Anthropic

Use for:
- Tag generation (`<=3` tags per vocab item).
- Topic generation (3 topics).
- Tag-to-topic relevance ranking (top 3 tags).
- Story generation.

### WaniKani

- Store API token in `user_settings.wanikani_api_key`.
- Pull whichever endpoints are needed to import unlocked vocab and keep `source='wanikani'` with `source_ref`.
- Incremental sync based on provider capabilities + `wanikani_last_synced_at`.

### Dictionary API

- On demand for long-press meaning and reading enrichment.
- Cache `meaning` and `reading` into `vocab_entries` once fetched.

### ElevenLabs

- One configured voice ID from backend config.
- Generate once per story, cache metadata in `story_audio`, and serve cached audio on replay.

## 6) Backend API (REST)

All endpoints are user-scoped internally to `default_user` for MVP.

### Settings

- `GET /api/v1/settings`
- `PUT /api/v1/settings`
- body: `{ "jlpt_level": "N3", "story_word_target": 120, "wanikani_api_key": "..." }`

### Vocab

- `GET /api/v1/vocab?query=&limit=&offset=`
- `POST /api/v1/vocab`
- body: `{ "entries": ["花", "走る"] }`
- behavior: merge duplicates by `normalized_surface`, generate/update tags via Anthropic.
- `POST /api/v1/vocab/import/wanikani`
- behavior: import unlocked vocab and merge duplicates.

### Topics

- `POST /api/v1/topics/generate`
- response: `{ "topics": ["...","...","..."] }`

### Stories

- `POST /api/v1/stories`
- body: `{ "topic": "夏祭り" }`
Flow:
- Choose top 3 related tags via LLM ranking.
- Select matching vocab (max 50).
- Random tone (`funny|shocking`).
- Generate story via Anthropic.
- Persist story + join rows.
- Index in Elasticsearch.
- response includes story payload.
- `GET /api/v1/stories?limit=&offset=`
- `GET /api/v1/stories/{storyID}`
- `GET /api/v1/stories/search?q=&limit=&offset=` (Elasticsearch full text)

### Story reading helpers

- `GET /api/v1/stories/{storyID}/tokens`
- response: tokenized story with offsets and matched vocab entry ids for highlight.
- `GET /api/v1/vocab/{vocabID}/details`
- response: `{ "surface": "花", "meaning": "flower", "reading": "はな" }`
- fills cache from dictionary API on miss.

### Audio

- `POST /api/v1/stories/{storyID}/audio`
- behavior: return cached audio if present, else generate via ElevenLabs and cache.
- response: stream URL or bytes depending on serving strategy.

## 7) LLM Prompt Contracts

Use versioned prompt constants and persist version strings in records where relevant.

### Topic generation output schema

```json
{ "topics": ["string", "string", "string"] }
```

### Tag generation output schema

```json
{ "tags": ["string", "string", "string"] }
```

Rules:
- Return 1-3 tags.
- Short noun phrases.
- No duplicates.

### Tag ranking output schema

```json
{ "top_tags": ["string", "string", "string"] }
```

### Story generation output schema

```json
{ "title": "string", "story": "string" }
```

Rules:
- Japanese only.
- Match requested JLPT level.
- Aim for configured word count.
- Tone must be either funny or shocking.
- Use many supplied vocab items naturally.

## 8) Frontend MVP Screens

### Home

- Auto-load 3 generated topics.
- Topic cards + regenerate topics button.
- Generate story CTA.
Story reader features:
- vocab highlighting.
- click on kanji for furigana.
- long-press for tooltip meaning.
- play button for TTS.

### Settings

- JLPT slider (`N5..N1`).
- Story length number input.
- WaniKani API key field.
- Import/sync WaniKani action.
- Manual vocab add (single and batch lines).

### Stories list/search

- History list of persisted stories.
- Search input using backend full-text endpoint.

## 9) Background Jobs (Optional in MVP, but recommended)

- `wanikani_sync_job` (manual trigger now; cron later).
- `dictionary_backfill_job` for unresolved reading/meaning entries.

## 10) Config (`env`)

- `DATABASE_URL`
- `ELASTICSEARCH_URL`
- `ANTHROPIC_API_KEY`
- `ANTHROPIC_MODEL`
- `ELEVENLABS_API_KEY`
- `ELEVENLABS_VOICE_ID`
- `DICTIONARY_API_BASE_URL`
- `WANIKANI_API_BASE_URL` (override for testing)

## 11) Local Development Environment

- Add `docker-compose.yml` to run local components for fast iteration.
- Include local services: backend, frontend, postgres, elasticsearch.
- Optionally include local object storage (for cached story audio artifacts) or mount a local volume.
- Exclude third-party APIs from compose (Anthropic, WaniKani, ElevenLabs, dictionary API).
- Add healthchecks and startup dependencies so `docker compose up` is stable.

## 12) Delivery Milestones

1. Milestone 0: Local Infrastructure
- Add and validate `docker-compose.yml` for local dev loop.

2. Milestone 1: Foundation
- DB migrations + repositories + settings/vocab CRUD.

3. Milestone 2: Generation Core
- Anthropic clients (topics/tags/ranking/story) + story persistence.

4. Milestone 3: Search + Reading
- Elasticsearch indexing/search + token/highlight endpoint + dictionary lookup cache.

5. Milestone 4: Audio + WaniKani
- ElevenLabs cached audio endpoint + WaniKani import/sync.

6. Milestone 5: Frontend Integration
- Home/settings/history/search wired to API.

## 13) Acceptance Criteria (MVP)

- Users can add vocab and duplicates are merged.
- WaniKani import adds unlocked vocab and merges existing entries.
- Home shows 3 dynamically generated topics.
- Story generation uses LLM-ranked top 3 tags and max 50 candidate vocab items.
- Story is persisted and appears in history.
- Story content is indexed and retrievable via full-text search.
- Reading view supports highlighting, furigana-on-click, and long-press meaning tooltip.
- Play returns cached audio after first generation for that story.
