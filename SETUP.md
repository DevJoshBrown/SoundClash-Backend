# BeatBattler Backend — Local Setup

Steps to get the Go backend running locally, written for someone who doesn't
use Go day-to-day.

## Prerequisites

Install these first:

1. **Go** (1.26.1 or newer) — https://go.dev/dl. Running the server fetches all
   Go dependencies automatically; there's no manual install step for those.
2. **Docker Desktop** — used to run Postgres via the included `docker-compose.yml`.
3. **goose** (database migration tool) — install once with:
   ```bash
   go install github.com/pressly/goose/v3/cmd/goose@latest
   ```
4. **ffmpeg** — only needed to test *audio uploads* (the `/submit` endpoint
   transcodes with it). The server runs fine without it otherwise.
   - macOS: `brew install ffmpeg`
   - Debian/Ubuntu: `sudo apt install ffmpeg`

## 1. Create your `.env`

The `.env` file holds secrets and is **not** committed to git. Copy the example
and fill in the values:

```bash
cp .env.example .env
```

Then set:

- `CLERK_SECRET_KEY` — **required**, the server won't start without it. Ask Josh
  for the dev key (share via a password manager, never commit it). It must come
  from the **same Clerk application** as the frontend's publishable key, or auth
  tokens won't validate.
- `POSTGRES_PASSWORD` — any value; it just needs to match between `.env` and the
  database container (docker-compose reads it from `.env`).
- Make sure `POSTGRES_PORT` and the port inside `DATABASE_URL` are the same.

## 2. Start Postgres

From the repo root:

```bash
docker compose up -d
```

This starts a Postgres container using the credentials from your `.env`.

## 3. Run the database migrations

This builds all the tables. Use the same connection string as your
`DATABASE_URL` (matching the password and port you set in `.env`):

```bash
goose postgres -dir migrations "postgres://beatbattler:<PASSWORD>@localhost:<PORT>/beatbattler?sslmode=disable" up
```

## 4. Run the server

```bash
go run ./cmd/server
```

The server listens on `http://localhost:8080` (or whatever `PORT` you set).
Watch the logs — on a healthy start you'll see the database connection confirmed.

## Connecting the frontend

The frontend (`beatbattler-web`) needs its own `.env.local` with:

- `VITE_API_URL=http://localhost:8080`
- The Clerk **publishable** key from the same Clerk app as the backend secret key.

## Useful extras

- `jq` makes the curl examples in `TESTING.md` readable (`brew install jq`).
- To stop Postgres: `docker compose down` (add `-v` to also wipe the data volume).
