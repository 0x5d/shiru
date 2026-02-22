## Local Development

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and Docker Compose v2+
- [ko](https://ko.build/install/) for building the Go backend image
- API keys for external services (optional for infra-only testing):
  - `ANTHROPIC_API_KEY`
  - `ELEVENLABS_API_KEY` / `ELEVENLABS_VOICE_ID`

### Quick Start

Build the backend image with ko and start all services:

```bash
KO_DOCKER_REPO=ko.local/shiru ko build . --bare
docker compose up
```

| Service         | URL                      |
|-----------------|--------------------------|
| Frontend        | http://localhost:3000     |
| Backend API     | http://localhost:8080     |
| Postgres        | localhost:5432            |
| Elasticsearch   | http://localhost:9200     |
| MinIO API       | http://localhost:9000     |
| MinIO Console   | http://localhost:9001     |

The frontend proxies `/api/` requests to the backend automatically.

### External API Keys

Third-party APIs (Anthropic, ElevenLabs, WaniKani, dictionary) are **not** containerized. Pass keys via environment variables or a `.env` file in the project root:

```bash
ANTHROPIC_API_KEY=sk-...
ANTHROPIC_MODEL=claude-sonnet-4-20250514
ELEVENLABS_API_KEY=...
ELEVENLABS_VOICE_ID=...
S3_ENDPOINT=minio:9000
S3_BUCKET=shiru-audio
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin
S3_USE_SSL=false
```

MinIO is used as an S3-compatible object store for audio files. The `createbuckets` init container automatically creates the `shiru-audio` bucket on startup. Log in to the MinIO Console at http://localhost:9001 with `minioadmin` / `minioadmin`.

### Stopping

```bash
docker compose down
```

To also remove persisted data (Postgres, Elasticsearch):

```bash
docker compose down -v
```
