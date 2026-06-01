package battle_participants

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/DevJoshBrown/BeatBattler/internal/audio"
	"github.com/DevJoshBrown/BeatBattler/internal/auth"
	"github.com/DevJoshBrown/BeatBattler/internal/db"
	"github.com/DevJoshBrown/BeatBattler/internal/hub"
	"github.com/DevJoshBrown/BeatBattler/internal/scheduler"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Handler struct {
	queries   *db.Queries
	hubs      *hub.Manager
	scheduler *scheduler.Scheduler
}

func NewHandler(queries *db.Queries, hubs *hub.Manager, sched *scheduler.Scheduler) *Handler {
	return &Handler{queries: queries, hubs: hubs, scheduler: sched}
}

func (h Handler) CreateParticipant(w http.ResponseWriter, r *http.Request) {
	var params db.CreateParticipantParams

	user, err := auth.GetUserFromRequest(r, h.queries)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	battle_id := chi.URLParam(r, "id")
	var btl_uid pgtype.UUID

	if err := btl_uid.Scan(battle_id); err != nil {
		http.Error(w, "invalid battle id", http.StatusBadRequest)
		return
	}

	params.UserID = user.ID
	params.BattleID = btl_uid

	p, err := h.queries.CreateParticipant(r.Context(), params)
	if err != nil {
		log.Printf("error: %v", err)
		http.Error(w, "failed to join battle", http.StatusInternalServerError)
		return
	}

	msg, _ := json.Marshal(map[string]string{"type": "participant_joined"})
	h.hubs.Broadcast(btl_uid, msg)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)

}

func (h Handler) SubmitParticipant(w http.ResponseWriter, r *http.Request) {

	user, err := auth.GetUserFromRequest(r, h.queries)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Get Battle
	battle_id := chi.URLParam(r, "id")

	var battle_uid pgtype.UUID
	if err := battle_uid.Scan(battle_id); err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	b, err := h.queries.GetBattle(r.Context(), battle_uid)
	if err != nil {
		http.Error(w, "battle not found", http.StatusNotFound)
		return
	}

	params := db.GetParticipantParams{
		BattleID: battle_uid,
		UserID:   user.ID,
	}

	// check user is a participant in the battle
	valid_participant, err := h.queries.GetParticipant(r.Context(), params)
	if err != nil {
		http.Error(w, "user is not a participant in this battle", http.StatusForbidden)
		return
	}

	// check the battle is still in progress
	if b.Status != "upload" {
		http.Error(w, "battle is not accepting uploads at this time", http.StatusBadRequest)
		return
	} else {

		//Set 32MB limit
		err = r.ParseMultipartForm(32 << 20)
		if err != nil {
			http.Error(w, "failed to parse multipart form", http.StatusInternalServerError)
			return
		}

		// get the audio from the form
		file, _, err := r.FormFile("audio")
		if err != nil {
			http.Error(w, "failed to read uploaded file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// save file to temp memory
		tmp, err := os.CreateTemp("", "beat-upload-*.tmp")
		if err != nil {
			http.Error(w, "failed to create temp file", http.StatusInternalServerError)
			return
		}
		defer os.Remove(tmp.Name())

		io.Copy(tmp, file)
		tmp.Close()

		outputDir := fmt.Sprintf("uploads/%s/%s", battle_id, valid_participant.ID)
		outputPath, duration, err := audio.Transcode(tmp.Name(), outputDir)
		if err != nil {
			http.Error(w, "failed to transcode audio", http.StatusInternalServerError)
			return
		}

		_, err = h.queries.UpdateParticipantBeatURL(r.Context(), db.UpdateParticipantBeatURLParams{
			ID:          valid_participant.ID,
			BeatUrl:     pgtype.Text{String: outputPath, Valid: true},
			SubmittedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})
		if err != nil {
			http.Error(w, "failed to update beat URL", http.StatusInternalServerError)
			return
		}

		_, err = h.queries.UpdateParticipantDuration(r.Context(), db.UpdateParticipantDurationParams{
			ID:              valid_participant.ID,
			DurationSeconds: pgtype.Int4{Int32: int32(duration), Valid: true},
		})
		if err != nil {
			http.Error(w, "failed to update beat duration", http.StatusInternalServerError)
			return
		}

		participants, err := h.queries.ListParticipants(r.Context(), battle_uid)
		if err == nil {
			allSubmitted := true
			for _, p := range participants {
				if !p.SubmittedAt.Valid {
					allSubmitted = false
					break
				}
			}
			if allSubmitted {
				h.scheduler.SkipUpload(battle_uid)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(valid_participant)

	}
}

func (h Handler) ListParticipants(w http.ResponseWriter, r *http.Request) {
	battle_id := chi.URLParam(r, "id")
	var btl_uid pgtype.UUID
	if err := btl_uid.Scan(battle_id); err != nil {
		http.Error(w, "failed to scan battle_id", http.StatusBadRequest)
		return
	}

	participants, err := h.queries.ListParticipants(r.Context(), btl_uid)
	if err != nil {
		http.Error(w, "failed to list participants", http.StatusInternalServerError)
		return
	}

	if len(participants) == 0 {
		participants = []db.ListParticipantsRow{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(participants)
}

func (h Handler) LeaveParticipant(w http.ResponseWriter, r *http.Request) {
	user, err := auth.GetUserFromRequest(r, h.queries)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	battle_id := chi.URLParam(r, "id")
	var btl_uid pgtype.UUID
	if err := btl_uid.Scan(battle_id); err != nil {
		http.Error(w, "invalid battle id", http.StatusBadRequest)
		return
	}

	b, err := h.queries.GetBattle(r.Context(), btl_uid)
	if err != nil {
		http.Error(w, "battle not found", http.StatusNotFound)
		return
	}

	if b.Status != "waiting" {
		http.Error(w, "cannot leave a battle that has already started", http.StatusBadRequest)
		return
	}

	err = h.queries.RemoveParticipant(r.Context(), db.RemoveParticipantParams{
		BattleID: btl_uid,
		UserID:   user.ID,
	})
	if err != nil {
		http.Error(w, "failed to leave battle", http.StatusInternalServerError)
		return
	}

	msg, _ := json.Marshal(map[string]string{"type": "participant_joined"})
	h.hubs.Broadcast(btl_uid, msg)

	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) FinishEarly(w http.ResponseWriter, r *http.Request) {
	user, err := auth.GetUserFromRequest(r, h.queries)
	if err != nil {
		http.Error(w, "unathorized", http.StatusUnauthorized)
		return
	}

	battle_id := chi.URLParam(r, "id")
	var btl_uid pgtype.UUID
	if err := btl_uid.Scan(battle_id); err != nil {
		http.Error(w, "invalid battle id", http.StatusBadRequest)
		return
	}

	b, err := h.queries.GetBattle(r.Context(), btl_uid)
	if err != nil {
		http.Error(w, "battle not found", http.StatusNotFound)
		return
	}

	if b.Status != "in_progress" {
		http.Error(w, "battle is not in progress", http.StatusBadRequest)
		return
	}

	err = h.queries.MarkFinishedEarly(r.Context(), db.MarkFinishedEarlyParams{
		BattleID: btl_uid,
		UserID:   user.ID,
	})
	if err != nil {
		http.Error(w, "failed to mark finished early", http.StatusInternalServerError)
		return
	}

	allDone, err := h.queries.AllFinishedEarly(r.Context(), btl_uid)
	if err == nil && allDone {
		if h.concludeIfWalkover(r.Context(), btl_uid, b) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.scheduler.SkipInProgress(btl_uid)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) concludeIfWalkover(ctx context.Context, battleID pgtype.UUID, b db.Battle) bool {
	count, err := h.queries.CountActiveParticipants(ctx, battleID)
	if err != nil || count != 1 {
		return false
	}

	eloGain := 0
	if !b.CreatorID.Valid {
		eloGain = 15
		participants, err := h.queries.ListParticipants(ctx, battleID)
		if err == nil {
			for _, p := range participants {
				if p.ParticipantStatus == "active" || p.ParticipantStatus == "finished" {
					h.queries.UpdateUserElo(ctx, db.UpdateUserEloParams{
						ID:        p.UserID,
						EloRating: p.EloRating + int32(eloGain),
					})
					break
				}
			}
		}
	}

	h.queries.UpdateBattleStatus(ctx, db.UpdateBattleStatusParams{
		ID:     battleID,
		Status: "cancelled",
	})
	msg, _ := json.Marshal(map[string]interface{}{"type": "walkover", "elo_gain": eloGain})
	h.hubs.Broadcast(battleID, msg)

	battleIDStr := fmt.Sprintf("%s", battleID)
	time.AfterFunc(2*time.Second, func() {
		ctx := context.Background()
		h.queries.DeleteBattleParticipants(ctx, battleID)
		h.queries.DeleteBattle(ctx, battleID)
		log.Printf("cleanup: deleted walkover battle %s", battleIDStr)
	})
	return true
}

func (h Handler) Forfeit(w http.ResponseWriter, r *http.Request) {
	user, err := auth.GetUserFromRequest(r, h.queries)
	if err != nil {
		http.Error(w, "unathorized", http.StatusUnauthorized)
		return
	}

	battle_id := chi.URLParam(r, "id")
	var btl_uid pgtype.UUID
	if err := btl_uid.Scan(battle_id); err != nil {
		http.Error(w, "invalid battle id", http.StatusBadRequest)
		return
	}

	b, err := h.queries.GetBattle(r.Context(), btl_uid)
	if err != nil {
		http.Error(w, "battle not found", http.StatusNotFound)
		return
	}

	if b.Status == "waiting" || b.Status == "results" {
		http.Error(w, "battle is not in progress", http.StatusBadRequest)
		return
	}

	err = h.queries.SetParticipantDisqualified(r.Context(), db.SetParticipantDisqualifiedParams{
		BattleID: btl_uid,
		UserID:   user.ID,
	})
	if err != nil {
		http.Error(w, "failed to DQ participant", http.StatusInternalServerError)
		return
	}

	// A forfeit is permanent, so the lone survivor wins immediately — no need to
	// wait for them to finish (unlike absent, where we give time to reconnect).
	if h.concludeIfWalkover(r.Context(), btl_uid, b) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	allDone, err := h.queries.AllFinishedEarly(r.Context(), btl_uid)
	if err == nil && allDone {
		h.scheduler.SkipInProgress(btl_uid)
	}

	msg, _ := json.Marshal(map[string]string{"type": "participant_update"})
	h.hubs.Broadcast(btl_uid, msg)

	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) Rejoin(w http.ResponseWriter, r *http.Request) {
	user, err := auth.GetUserFromRequest(r, h.queries)
	if err != nil {
		http.Error(w, "unathorized", http.StatusUnauthorized)
		return
	}

	battle_id := chi.URLParam(r, "id")
	var btl_uid pgtype.UUID
	if err := btl_uid.Scan(battle_id); err != nil {
		http.Error(w, "invalid battle id", http.StatusBadRequest)
		return
	}

	b, err := h.queries.GetBattle(r.Context(), btl_uid)
	if err != nil {
		http.Error(w, "battle not found", http.StatusNotFound)
		return
	}

	if b.Status != "in_progress" {
		http.Error(w, "battle is not in progress", http.StatusBadRequest)
		return
	}

	// Backend updates the DB (`SetParticipantDisqualified`, `SetParticipantActive`, etc.)
	err = h.queries.SetParticipantActive(r.Context(), db.SetParticipantActiveParams{
		BattleID: btl_uid,
		UserID:   user.ID,
	})
	if err != nil {
		http.Error(w, "failed to set participant to active", http.StatusInternalServerError)
		return
	}

	// Backend broadcasts `{"type": "participant_update"}` to all clients in that battle
	// Each client's `useBattleSocket` receives it → invalidates the participants query
	msg, _ := json.Marshal(map[string]string{"type": "participant_update"})
	h.hubs.Broadcast(btl_uid, msg)

	// React Query refetches participants → frontend re-renders showing the new status
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) Absent(w http.ResponseWriter, r *http.Request) {
	user, err := auth.GetUserFromRequest(r, h.queries)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	battle_id := chi.URLParam(r, "id")
	var btl_uid pgtype.UUID
	if err := btl_uid.Scan(battle_id); err != nil {
		http.Error(w, "invalid battle id", http.StatusBadRequest)
		return
	}

	b, err := h.queries.GetBattle(r.Context(), btl_uid)
	if err != nil {
		http.Error(w, "battle not found", http.StatusNotFound)
		return
	}

	if b.Status != "in_progress" {
		http.Error(w, "battle is not in progress", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	h.queries.SetParticipantAbsent(ctx, db.SetParticipantAbsentParams{
		BattleID: btl_uid,
		UserID:   user.ID,
	})

	// Tell everyone this player is now absent so their card greys out.
	msg, _ := json.Marshal(map[string]string{"type": "participant_update"})
	h.hubs.Broadcast(btl_uid, msg)

	// Give them 60s to rejoin. If they're still absent when the timer fires they
	// forfeit, which then resolves the battle the same way an explicit forfeit does.
	time.AfterFunc(60*time.Second, func() {
		ctx := context.Background()
		p, err := h.queries.GetParticipant(ctx, db.GetParticipantParams{
			BattleID: btl_uid,
			UserID:   user.ID,
		})
		if err != nil || p.ParticipantStatus != "absent" {
			return // they rejoined
		}

		h.queries.SetParticipantDisqualified(ctx, db.SetParticipantDisqualifiedParams{
			BattleID: btl_uid,
			UserID:   user.ID,
		})

		current, err := h.queries.GetBattle(ctx, btl_uid)
		if err != nil {
			return
		}

		if h.concludeIfWalkover(ctx, btl_uid, current) {
			return
		}

		// Everyone abandoned — no one left to crown. Cancel the battle outright
		// rather than limping to an empty results screen.
		if count, err := h.queries.CountActiveParticipants(ctx, btl_uid); err == nil && count == 0 {
			h.queries.UpdateBattleStatus(ctx, db.UpdateBattleStatusParams{
				ID:     btl_uid,
				Status: "cancelled",
			})
			msg, _ := json.Marshal(map[string]string{"type": "stage_change", "status": "cancelled"})
			h.hubs.Broadcast(btl_uid, msg)
			return
		}

		allDone, err := h.queries.AllFinishedEarly(ctx, btl_uid)
		if err == nil && allDone {
			h.scheduler.SkipInProgress(btl_uid)
		}

		msg, _ := json.Marshal(map[string]string{"type": "participant_update"})
		h.hubs.Broadcast(btl_uid, msg)
	})

	w.WriteHeader(http.StatusNoContent)
}
