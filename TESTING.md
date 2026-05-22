# BeatBattler — curl Testing Guide

Base URL: `http://localhost:8080`

---

## 1. Discover existing UUIDs

```bash
# Get a user
curl -s http://localhost:8080/users/<USER_ID> | jq .

# List all battles
curl -s http://localhost:8080/battles | jq .

# List participants for a specific battle
curl -s http://localhost:8080/battles/<BATTLE_ID>/participants | jq .
```

---

## 2. Create test users

```bash
# Create user 1 (will be the battle creator)
curl -s -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"username":"creator","display_name":"Creator"}' | jq .

# Save the returned id as CREATOR_ID

# Create user 2
curl -s -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"username":"player2","display_name":"Player 2"}' | jq .

# Save the returned id as PLAYER2_ID

# Create user 3 (voter / extra participant)
curl -s -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"username":"player3","display_name":"Player 3"}' | jq .

# Save the returned id as PLAYER3_ID
```

---

## 3. Create and inspect a battle

```bash
# Create a battle (creator auto-joins on creation)
# duration_minutes must be one of: 10, 15, 20, 30, 60, 120
curl -s -X POST http://localhost:8080/battles \
  -H "Content-Type: application/json" \
  -H "X-User-ID: <CREATOR_ID>" \
  -d '{"title":"Test Battle","duration_minutes":1}' | jq .

# Save the returned id as BATTLE_ID

# Get the battle
curl -s http://localhost:8080/battles/<BATTLE_ID> | jq .

# Confirm creator is already a participant
curl -s http://localhost:8080/battles/<BATTLE_ID>/participants | jq .
```

> **Note:** `duration_minutes: 1` is not in the allowed set — use `10` for the quickest real test, or temporarily remove the CHECK constraint in the DB for dev.

---

## 4. Players join the battle

```bash
# Player 2 joins
curl -s -X POST http://localhost:8080/battles/<BATTLE_ID>/join \
  -H "X-User-ID: <PLAYER2_ID>" | jq .

# Player 3 joins
curl -s -X POST http://localhost:8080/battles/<BATTLE_ID>/join \
  -H "X-User-ID: <PLAYER3_ID>" | jq .

# Verify all three participants
curl -s http://localhost:8080/battles/<BATTLE_ID>/participants | jq .
```

---

## 5. Start the battle

```bash
# Only the creator can start — status flips to in_progress, timer begins
curl -s -X POST http://localhost:8080/battles/<BATTLE_ID>/start \
  -H "X-User-ID: <CREATOR_ID>" | jq .
```

The scheduler goroutine is now running. It will advance through:
`in_progress → upload → listening → voting → results`

Watch the server logs for stage transition messages.

---

## 6. Submit beat URLs (during upload stage)

Each participant must submit before the upload window closes (2 min).

```bash
# Creator submits
curl -s -X POST http://localhost:8080/battles/<BATTLE_ID>/submit \
  -H "X-User-ID: <CREATOR_ID>" \
  -H "Content-Type: application/json" \
  -d '{"beat_url":"https://example.com/beats/creator.mp3"}' | jq .

# Player 2 submits
curl -s -X POST http://localhost:8080/battles/<BATTLE_ID>/submit \
  -H "X-User-ID: <PLAYER2_ID>" \
  -H "Content-Type: application/json" \
  -d '{"beat_url":"https://example.com/beats/player2.mp3"}' | jq .

# Player 3 submits
curl -s -X POST http://localhost:8080/battles/<BATTLE_ID>/submit \
  -H "X-User-ID: <PLAYER3_ID>" \
  -H "Content-Type: application/json" \
  -d '{"beat_url":"https://example.com/beats/player3.mp3"}' | jq .

# Check participants — beat_url and is_submitted should now be populated
curl -s http://localhost:8080/battles/<BATTLE_ID>/participants | jq .
```

---

## 7. Cast votes (during voting stage)

Each participant votes on the other two tracks (no self-voting). Scores are 1–5. Votes can be updated (upsert) until the voting window closes.

```bash
# Creator votes for Player 2's participant
curl -s -X POST http://localhost:8080/battles/<BATTLE_ID>/vote \
  -H "X-User-ID: <CREATOR_ID>" \
  -H "Content-Type: application/json" \
  -d '{"voted_for_participant_id":"<PLAYER2_PARTICIPANT_ID>","score":4}' | jq .

# Creator votes for Player 3's participant
curl -s -X POST http://localhost:8080/battles/<BATTLE_ID>/vote \
  -H "X-User-ID: <CREATOR_ID>" \
  -H "Content-Type: application/json" \
  -d '{"voted_for_participant_id":"<PLAYER3_PARTICIPANT_ID>","score":3}' | jq .

# Player 2 votes
curl -s -X POST http://localhost:8080/battles/<BATTLE_ID>/vote \
  -H "X-User-ID: <PLAYER2_ID>" \
  -H "Content-Type: application/json" \
  -d '{"voted_for_participant_id":"<CREATOR_PARTICIPANT_ID>","score":5}' | jq .

curl -s -X POST http://localhost:8080/battles/<BATTLE_ID>/vote \
  -H "X-User-ID: <PLAYER2_ID>" \
  -H "Content-Type: application/json" \
  -d '{"voted_for_participant_id":"<PLAYER3_PARTICIPANT_ID>","score":2}' | jq .

# Player 3 votes
curl -s -X POST http://localhost:8080/battles/<BATTLE_ID>/vote \
  -H "X-User-ID: <PLAYER3_ID>" \
  -H "Content-Type: application/json" \
  -d '{"voted_for_participant_id":"<CREATOR_PARTICIPANT_ID>","score":4}' | jq .

curl -s -X POST http://localhost:8080/battles/<BATTLE_ID>/vote \
  -H "X-User-ID: <PLAYER3_ID>" \
  -H "Content-Type: application/json" \
  -d '{"voted_for_participant_id":"<PLAYER2_PARTICIPANT_ID>","score":3}' | jq .
```

---

## 8. Confirm votes (optional early end)

If all participants confirm, the voting stage ends immediately instead of waiting for the 60s timer.

```bash
curl -s -X POST http://localhost:8080/battles/<BATTLE_ID>/confirm-votes \
  -H "X-User-ID: <CREATOR_ID>" | jq .

curl -s -X POST http://localhost:8080/battles/<BATTLE_ID>/confirm-votes \
  -H "X-User-ID: <PLAYER2_ID>" | jq .

curl -s -X POST http://localhost:8080/battles/<BATTLE_ID>/confirm-votes \
  -H "X-User-ID: <PLAYER3_ID>" | jq .
```

---

## 9. Get results (once status = results)

```bash
curl -s http://localhost:8080/battles/<BATTLE_ID>/results | jq .
```

Response includes each participant ranked by average vote score, with their position (1 = winner).

---

## Tips

- Run `curl -s http://localhost:8080/battles/<BATTLE_ID> | jq .status` to poll the current stage.
- The server logs show each stage transition as it happens.
- Participant IDs (not user IDs) are what you pass to `/vote`. Get them from `/participants`.
- A disqualified participant (no submission by end of upload) cannot vote and is placed last.


//
TEST USERS
{
  "id": "21d71f20-48b2-4f4b-9dc9-74c8ffb3ec5b",
  "username": "testuser",
  "display_name": "Test User",
  "elo_rating": 1000,
  "battles_played": 0,
  "battles_won": 0,
  "created_at": "2026-05-15T13:59:26.308334+01:00",
  "updated_at": "2026-05-15T13:59:26.308334+01:00"
}
{
  "id": "52fb7abf-8155-4906-8e23-5a97b7f6b4ee",
  "username": "testuser2",
  "display_name": "",
  "elo_rating": 1000,
  "battles_played": 0,
  "battles_won": 0,
  "created_at": "2026-05-15T18:02:49.983488+01:00",
  "updated_at": "2026-05-15T18:02:49.983488+01:00"
}
{
  "id": "625a1696-b48e-4b09-9d1a-02b67ee9e157",
  "username": "testuser3",
  "display_name": "Test User3",
  "elo_rating": 1000,
  "battles_played": 0,
  "battles_won": 0,
  "created_at": "2026-05-22T09:36:31.552429+01:00",
  "updated_at": "2026-05-22T09:36:31.552429+01:00"
}
