## Shiru MVP Tracker

Last updated: 2026-02-21

## Current Status

- Done: Product spec clarified and persisted.
- Done: Implementation plan documented.
- Done: Milestone 0 (Local Infrastructure).
- Done: Milestone 1 (Foundation).
- Done: Milestone 2 (Generation Core).
- Next: Milestone 3 (Search + Reading).

## Completed Work

- Updated `spec.md` with resolved Q&A decisions.
- Added implementation plan in `docs/mvp-engineering-plan.md`.
- Linked `spec.md` to the implementation plan.
- Milestone 0: Added `docker-compose.yml`, `Dockerfile` (backend), `web/Dockerfile` (frontend), `web/nginx.conf`, and `docs/local-dev.md`.
- Milestone 1: Added DB migrations, domain types, config, Postgres repositories (settings + vocab with duplicate merge), REST endpoints (`GET/PUT /settings`, `GET/POST /vocab`), mocks, and unit tests.
- Milestone 2: Added config, Anthropic client, story model/repository/service, mocks, and tests.

## Milestone Checklist

- [x] Milestone 0: Local Infrastructure
- [x] Milestone 1: Foundation
- [x] Milestone 2: Generation Core
- [ ] Milestone 3: Search + Reading
- [ ] Milestone 4: Audio + WaniKani
- [ ] Milestone 5: Frontend Integration

## Milestone 0: Local Infrastructure

Status: `done`

Definition of done:
- `docker-compose.yml` exists for local development.
- Includes local infra components only:
- `postgres`
- `elasticsearch`
- optional local storage service for audio artifacts (for example, `minio`) or local volume strategy.
- Includes the local app containers:
- backend service
- frontend service
- Includes healthchecks and `depends_on` wiring so startup order is predictable.
- Local dev boot command is documented in `README` (or dedicated local-dev doc).
- No third-party APIs are containerized (`Anthropic`, `WaniKani`, `ElevenLabs`, dictionary provider).

## Milestone 1: Foundation

Status: `done`

Definition of done:
- Postgres migrations exist for core schema (`users`, `user_settings`, `vocab_entries`, `tags`, `vocab_entry_tags`, `stories`, `story_vocab_entries`, `story_audio`, `topic_snapshots`).
- `default_user` seed exists.
- Go repository layer supports settings and vocab CRUD with duplicate merge by normalized surface.
- Basic REST endpoints are wired: `GET/PUT /api/v1/settings`, `GET/POST /api/v1/vocab`.
- Unit tests cover duplicate merge and settings update behavior.

## Milestone 2: Generation Core

Status: `done`

Definition of done:
- Anthropic client integrated for topic generation, tag generation, tag ranking, and story generation.
- `POST /api/v1/topics/generate` works.
- `POST /api/v1/stories` works end-to-end with persistence.
- Story generation enforces max 50 candidate vocab entries.
- Tone is randomly selected (`funny` or `shocking`).

## Milestone 3: Search + Reading

Status: `pending`

Definition of done:
- Story indexing to Elasticsearch on creation.
- `GET /api/v1/stories/search` returns full-text matches.
- `GET /api/v1/stories/{storyID}/tokens` supports highlighting offsets.
- `GET /api/v1/vocab/{vocabID}/details` fetches meaning/reading and caches dictionary results.

## Milestone 4: Audio + WaniKani

Status: `pending`

Definition of done:
- `POST /api/v1/stories/{storyID}/audio` returns cached audio when present.
- ElevenLabs generation stores audio metadata in `story_audio`.
- WaniKani import endpoint implemented: `POST /api/v1/vocab/import/wanikani`.
- Incremental sync updates `wanikani_last_synced_at`.

## Milestone 5: Frontend Integration

Status: `pending`

Definition of done:
- Home page wired to topics + story generation APIs.
- Story reading experience supports highlighting, furigana click, long-press meaning tooltip, and TTS play.
- Settings page wired for JLPT, story length, manual vocab, and WaniKani key/import.
- Story history + search integrated.

## Outstanding Decisions

- None blocking MVP start.
- Any remaining unspecified details should default to `docs/mvp-engineering-plan.md`.

## Session Resume Notes

- Resume source of truth: `spec.md`, `docs/mvp-engineering-plan.md`, `docs/mvp-tracker.md`.
- When work starts, move milestone status from `pending` -> `in progress` -> `done`.
- Update this file in every PR that changes milestone status.
