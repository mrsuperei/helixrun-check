# HelixRun tRPC-Agent-Go Starter

This starter provides a minimal but extensible setup for HelixRun using
[tRPC-Agent-Go](https://trpc-group.github.io/trpc-agent-go/).

## Features

- Dynamic agent loading from JSON definitions
- Single LLM agent with function tool
- Multi-Agent chain (planner -> writer)
- GraphAgent with 3 nodes (entry -> clarify -> answer)
- Custom `/chat` HTTP endpoint that:
  - Accepts JSON chat requests
  - Spins up an isolated Runner per request
  - Streams all events via Server-Sent Events (SSE)

## Requirements

- Go 1.21+ (1.22 recommended)
- `OPENAI_API_KEY` set in your environment
- Optionally `OPENAI_BASE_URL` if using an OpenAI-compatible proxy
- `DATABASE_URL` pointing to a PostgreSQL instance (needed for CLIProxy integration)

## Quick start

```bash
cd helixrun-starter

# Tidy and download deps
go mod tidy

# Apply migrations (replace with your favourite tooling)
psql "$DATABASE_URL" -f configs/migrations/0001_cliprproxy.sql

# Run HTTP server on :8080
go run ./cmd/server
```

## /chat endpoint

- Method: `POST`
- Path: `/chat`
- Request body:

```json
{
  "agent_id": "simple-tool-agent",
  "message": "What is 2 + 3?",
  "user_id": "demo-user"
}
```

Response is an SSE stream (`text/event-stream`). Each line is:

```text
data: { ... JSON encoded tRPC-Agent-Go event ... }
```

You can consume this with a streaming `fetch()` in the browser or any SSE client
that accepts POST + `text/event-stream`.

## CLIProxy REST endpoints

HelixRun now persists Router CLIProxy state to PostgreSQL. After setting `DATABASE_URL`
and running the migration, the following endpoints become available:

- `GET  /api/cliproxy/keys` - list managed provider keys (secrets are redacted)
- `POST /api/cliproxy/keys` - create a new key (body: `provider`, `secret`, optional metadata)
- `PUT  /api/cliproxy/keys/{id}` - update label/limits/secret
- `DELETE /api/cliproxy/keys/{id}` - remove a key
- `GET  /api/cliproxy/usage` - list usage events, filterable via `provider`, `key_id`, `from`, `to`

These endpoints back the future HelixRun frontend for CLIProxy administration.

For quick manual management (preview UI), open `http://localhost:8080/cliproxy.html`.  
This static page talks to the `/api/cliproxy/*` endpoints to inspect usage and manage keys.
