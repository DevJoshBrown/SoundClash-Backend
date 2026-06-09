# BeatBattler / Soundclash — Project Handoff

## What this is
A real-time muliplayer beat battle platform. Multiple producers join, get a sample pack, record their beat, then score the other producers tracks out of 5, which decides on a winner. Branded "Soundclash".

## Repos
- `BeatBattler/` — Go backend
- `beatbattler-web/` — React frontend

## Backend stack
- Go + Chi router
- PostgreSQL via `pgx/pgxpool`
- `sqlc` for DB queries (raw SQL in `sql/queries/`, generated code in `internal/db/`)
- `goose` for migrations (`migrations/`)
- Gorilla WebSockets for real-time battle state
- Clerk for auth (JWT middleware in `internal/middleware/`)
- FFmpeg for audio transcoding (`internal/audio/`)
- Cloudflare R2 for sample pack storage (`pkg/storage/r2/`)
- Deployed on Railway via Dockerfile

## Frontend stack
- React 19 + Vite + TypeScript
- TanStack React Query 5 (data fetching/cache)
- Clerk React (auth)
- Tailwind CSS 4
- Framer Motion (animations)
- Deployed on Vercel

## Battle lifecycle (status field on `battles` table)
`waiting` → `forming` (30s sample pack reveal, not fully built yet) → `in_progress` → `upload` → `listening` → `voting` → `finished` / `cancelled`

## Participant status (`battle_participants.participant_status`)
`active` | `absent` | `forfeited` | `finished`

Absent = disconnected/navigated away from an in-progress battle. 60s grace period before auto-DQ. User can rejoin. Intentional forfeit is permanent.

## Key patterns
- `concludeIfWalkover` helper in `internal/battle_participants/handler.go` — call after any status change that could leave one active participant; handles walkover ELO only for rated games (`!b.CreatorID.Valid`)
- `AbsentBattleContext` in frontend (`src/context/absentBattle.tsx`) — client-driven React Context for showing the "you left a battle" card; don't replace with server polling
- `leavingDeliberatelyRef` in `Battle.tsx` — guards the unmount cleanup so deliberate forfeit/walkover navigation doesn't re-trigger the absent flow
- `ApiError` class in `src/lib/api.ts` with `.status` field — use `instanceof ApiError` checks to handle 404s without swallowing other errors

## Current state (as of 2026-06-09)
- Core battle flow: working end-to-end in production
- Participant status system (absent/forfeit/rejoin/walkover): working
- Sample packs: DB + R2 storage built, upload + list endpoints working. **Not yet wired into battle start.**
- `forming` stage: not yet built

## Next planned work
1. Wire `GetRandomPackByGenre` into `StartBattle` — set battle status to `forming`, attach sample pack ID
2. Build `forming` stage (30s countdown): backend timer transitions `forming` → `in_progress`; frontend shows pack info + download link during the countdown
3. Keep download button accessible during `in_progress`

## Teaching style — IMPORTANT
Josh is learning. **Do not edit his codebase directly.** Guide him with:
- Pseudocode / concept explanations
- Pointing to the right files and line numbers
- Explaining the mental model before the implementation
- Short walkthroughs, not code dumps

He will write the code himself. Only paste complete code if he explicitly asks.

## Visual direction
Red Bull 3Style aesthetic: event poster meets video game HUD. Dense layout, heavy condensed type, sharp borders, near-black background + one accent colour. Desktop-first.

## Deployment
- Backend: Railway, auto-deploys from `BeatBattler/` on push to main. Env vars set in Railway dashboard.
- Frontend: Vercel, auto-deploys from `beatbattler-web/` on push to main.
- `vercel.json` rewrites all routes to `index.html` for React Router.

## Known deferred work
- WS auth (currently unauthenticated — browsers can't set Auth headers on WS upgrades)
- Rate limiting
- Clerk production keys (using test keys until wider launch)
- Leaderboard, chat, singleplayer mode
- Performance Optimisation
- Visual Design once all page elements are implemented
- Animations and visual/audio cues through the battle sequence (to give a video game feel)
