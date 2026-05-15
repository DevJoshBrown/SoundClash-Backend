# Beat Arena — Project Brief

## Overview

Beat Arena is a real-time beat battle platform for music producers. Producers join lobbies, receive sample packs, produce beats in their local DAWs, upload submissions, vote on each other's work, and receive ELO-based rankings. Think multiplayer game lobby, not content platform. Audio is ephemeral — it exists only for the duration of a battle session and is purged on lobby close.

This platform replaces beat-battle.net, which suffers from extreme slowness/downtime, limited genres (trap/EDM only), poor quality samples, clunky UI, and an unreliable ranking system.

---

## Tech Stack

- **Language:** Go (primary backend language)
- **Router:** Chi or Gin
- **Database:** PostgreSQL via pgx (pure Go driver, no ORM)
- **Cache / Real-time state:** Redis (session state, live vote tallies, leaderboard via sorted sets)
- **Permanent file storage:** Cloudflare R2 (sample packs only — zero egress fees)
- **Ephemeral audio:** Server local disk / temp directory (battle submissions only, purged on lobby close)
- **CDN:** Cloudflare (edge caching for samples and static assets)
- **Audio processing:** FFmpeg called from Go (transcode uploads to 128k AAC for streaming, LUFS normalisation)
- **Auth:** Clerk or Supabase Auth (do not build from scratch)
- **Real-time:** WebSockets (gorilla/websocket or nhooyr/websocket) for live battle events, vote updates, lobby state
- **Frontend:** Next.js + React + Tailwind CSS (built by a separate frontend developer against the Go API)
- **Hosting:** Vercel (frontend), Railway or Fly.io (Go backend)

---

## Architecture

### Layers

1. **Frontend (Next.js + React)** — UI, audio player with Web Audio API, waveform display, voting interface, lobby management
2. **API Gateway (Go)** — Single entry point. REST for CRUD operations, WebSocket upgrade for real-time lobby events. Handles auth middleware, rate limiting, request routing
3. **Backend Services (Go, internal packages):**
   - `audio` — Upload validation, FFmpeg transcoding, temp file management, streaming, cleanup
   - `battle` — Lobby creation, joining, timer management, submission handling, vote collection, results
   - `user` — Profile management, auth integration, stats
   - `ranking` — ELO calculation, leaderboard management, genre-specific rankings
4. **Data Layer:**
   - PostgreSQL — Users, battle metadata, vote records, ELO history, sample pack metadata
   - Redis — Active lobby session state, live vote counts, leaderboard sorted sets, caching
   - Cloudflare R2 — Curated sample packs (permanent storage)
5. **External Services:** Cloudflare CDN, FFmpeg, Clerk Auth, WebSocket connections

### Data Lifecycle

- **Permanent:** Sample packs (R2), battle metadata/results (PostgreSQL), ELO scores and history (PostgreSQL), user profiles (PostgreSQL)
- **Ephemeral:** Battle audio submissions — stored in server temp directory for battle duration only, streamed to lobby participants, purged immediately on lobby close. Producers already have their beats locally from their DAW. No need to persist.

---

## Project Structure

```
beat-arena/
├── cmd/
│   └── server/
│       └── main.go              # Entry point
├── internal/
│   ├── audio/
│   │   ├── handler.go           # Upload/stream HTTP handlers
│   │   ├── transcoder.go        # FFmpeg integration
│   │   ├── storage.go           # Temp file management + R2 client for samples
│   │   └── cleanup.go           # Purge temp files on lobby close
│   ├── battle/
│   │   ├── handler.go           # Battle/lobby REST + WebSocket handlers
│   │   ├── lobby.go             # Lobby session state machine
│   │   ├── timer.go             # Battle countdown management
│   │   └── voting.go            # Vote collection and tallying
│   ├── user/
│   │   ├── handler.go           # Profile endpoints
│   │   └── repository.go        # User DB queries
│   └── ranking/
│       ├── elo.go               # ELO calculation algorithm
│       ├── leaderboard.go       # Redis sorted set operations
│       └── handler.go           # Ranking/leaderboard endpoints
├── pkg/
│   ├── middleware/
│   │   ├── auth.go              # Clerk auth middleware
│   │   ├── ratelimit.go         # Rate limiting
│   │   └── logging.go           # Request logging
│   ├── storage/
│   │   ├── postgres.go          # DB connection pool
│   │   ├── redis.go             # Redis client
│   │   └── r2.go                # Cloudflare R2 client
│   └── websocket/
│       └── hub.go               # WebSocket connection manager
├── migrations/
│   ├── 001_users.sql
│   ├── 002_battles.sql
│   ├── 003_entries.sql
│   ├── 004_votes.sql
│   └── 005_elo_history.sql
├── config/
│   └── config.go                # Environment/config loading
├── go.mod
└── go.sum
```

---

## Core Database Schema

### users
- `id` (UUID, PK)
- `clerk_id` (string, unique — external auth reference)
- `username` (string, unique)
- `display_name` (string)
- `elo_rating` (integer, default 1000)
- `battles_played` (integer, default 0)
- `battles_won` (integer, default 0)
- `created_at` (timestamp)
- `updated_at` (timestamp)

### battles
- `id` (UUID, PK)
- `creator_id` (UUID, FK → users)
- `genre` (string — e.g. drill, afrobeats, jersey_club, trap, edm, footwork, ambient, soul, jazz)
- `status` (enum: waiting, in_progress, voting, completed, cancelled)
- `duration_minutes` (integer, 10-30)
- `max_participants` (integer)
- `sample_pack_id` (UUID, FK → sample_packs, nullable)
- `started_at` (timestamp, nullable)
- `completed_at` (timestamp, nullable)
- `created_at` (timestamp)

### entries
- `id` (UUID, PK)
- `battle_id` (UUID, FK → battles)
- `user_id` (UUID, FK → users)
- `temp_audio_key` (string — local temp file path, cleared on purge)
- `submitted_at` (timestamp)

### votes
- `id` (UUID, PK)
- `battle_id` (UUID, FK → battles)
- `voter_id` (UUID, FK → users)
- `entry_id` (UUID, FK → entries)
- `cast_at` (timestamp)
- Unique constraint on (battle_id, voter_id) — one vote per person per battle

### elo_history
- `id` (UUID, PK)
- `user_id` (UUID, FK → users)
- `battle_id` (UUID, FK → battles)
- `elo_before` (integer)
- `elo_after` (integer)
- `recorded_at` (timestamp)

### sample_packs
- `id` (UUID, PK)
- `name` (string)
- `genre` (string)
- `r2_key` (string — path in Cloudflare R2)
- `uploaded_by` (UUID, FK → users, nullable — for admin/curator tracking)
- `created_at` (timestamp)

---

## Battle Lobby Session Flow

1. **Create lobby** — A user creates a battle, selects genre, sets duration (10-30 min), optionally assigns a sample pack. Status: `waiting`.
2. **Join lobby** — Other producers join via lobby browser or invite link. WebSocket connection established per participant.
3. **Battle starts** — Creator starts the battle (or auto-start when full). Timer begins. Sample pack URL distributed to participants. Status: `in_progress`. Producers work in their local DAW.
4. **Upload window** — Before timer expires, producers upload their beat. Go server validates format (WAV/MP3/FLAC), transcodes to 128k AAC via FFmpeg, stores in temp directory, and notifies lobby via WebSocket.
5. **Voting phase** — Timer ends, status moves to `voting`. All submissions streamed to all participants via WebSocket/chunked HTTP. Each producer votes for their favourite (cannot vote for self). Votes stored in PostgreSQL with timestamps.
6. **Results** — All votes in (or voting timeout). Winner determined by vote count. ELO calculated for all participants. Results broadcast via WebSocket. Status: `completed`.
7. **Cleanup** — Once all participants disconnect (or after a grace period), a goroutine purges all temp audio files for that battle. `temp_audio_key` fields cleared in DB. Battle metadata and results persist permanently.

---

## ELO Ranking System

- Standard ELO with K-factor of 32 (adjustable)
- Every participant's rating adjusts based on battle outcome
- Genre-specific leaderboards stored as Redis sorted sets (`ZADD genre:{genre} {elo} {user_id}`)
- Global leaderboard aggregates across genres
- Full ELO history stored in PostgreSQL for auditability and transparency
- Algorithm published openly — producers hate black boxes

---

## Sample Pack Strategy

Sample packs are the platform's key differentiator. The founder has 10 years of production experience and live music industry connections to source high-quality, genre-diverse content.

- Professionally designed, high-quality samples
- Genre diversity: drill, afrobeats, Jersey club, footwork, ambient, soul, jazz, trap, EDM, and more
- Stored permanently in Cloudflare R2, served via CDN
- Assigned to battles by genre or curator selection

---

## Genre Support

Broad from launch — not limited to trap/EDM like beat-battle.net:
- Trap, EDM, Drill, Afrobeats, Jersey club, Footwork, Ambient, Soul, Jazz, Lo-fi, DnB, House, Techno, UK Garage, Grime, and expandable via database (not hardcoded enums)

---

## Build Phases

### Phase 1 — Audio Pipeline (Weeks 1-3)
- Go project scaffolding and structure
- PostgreSQL schema and migrations
- Audio upload endpoint with format validation
- FFmpeg transcoding pipeline (background goroutine using Go channels)
- Temp file storage and cleanup logic
- R2 integration for permanent sample pack storage
- Basic streaming endpoint

### Phase 2 — Core Battle Loop (Weeks 4-6)
- Battle/lobby CRUD endpoints
- WebSocket lobby management (join, leave, state sync)
- Battle timer with server-authoritative countdown
- Submission flow (upload during battle, notify lobby)
- Voting system (one vote per user per battle, no self-votes)
- Results calculation and broadcast

### Phase 3 — Ranking and Progression (Weeks 7-8)
- ELO calculation engine
- Per-genre and global leaderboards via Redis sorted sets
- ELO history tracking
- Profile stats (battles played, won, win rate, ranking)

### Phase 4 — Integration and Polish (Weeks 9-10)
- Frontend developer builds UI against the API (ideally starts in parallel from week 3)
- Rate limiting and input validation hardening
- Error handling and structured logging
- Basic monitoring and health checks
- Load testing concurrent lobby sessions

---

## Key Technical Decisions

1. **No ORM** — Raw SQL via pgx. Full control over queries, no abstraction headaches.
2. **Ephemeral audio only** — No permanent storage of battle submissions. Producers have their beats locally. Keeps costs predictable and bounded.
3. **Server-authoritative timers** — Never trust the client for battle timing. Server manages countdown, broadcasts state.
4. **Separate services as internal packages** — Not microservices, but cleanly separated domains within a single Go binary. Can extract later if needed.
5. **Genre as database values, not code enums** — New genres added without code changes.
6. **WebSockets for all real-time state** — Lobby presence, vote counts, timer sync, results announcement.

---

## Scaling Considerations

The primary bottleneck is concurrent active lobbies, not storage. Each lobby holds audio in temp storage and streams to N participants. Scaling means:
- Horizontal scaling of Go instances behind a load balancer
- Sticky sessions or shared session state (Redis) for WebSocket connections
- CDN for sample pack delivery (already handled by Cloudflare)
- PostgreSQL connection pooling via pgxpool

---

## Non-Goals for MVP

- Social features (following, messaging, comments)
- Sample marketplace or user-uploaded sample packs
- Premium/paid tier
- Chat within lobbies (add later)
- Mobile app (responsive web first)
- Audio recording in-browser (producers use their own DAWs)
