package scheduler

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/DevJoshBrown/BeatBattler/internal/db"
	"github.com/DevJoshBrown/BeatBattler/internal/hub"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Scheduler struct {
	queries *db.Queries
	pool    *pgxpool.Pool
	hubs    *hub.Manager
}

func NewScheduler(queries *db.Queries, pool *pgxpool.Pool, hubs *hub.Manager) *Scheduler {
	return &Scheduler{queries: queries, pool: pool, hubs: hubs}
}

func (s *Scheduler) Run(ctx context.Context, battleID pgtype.UUID, duration time.Duration) {
	log.Printf("scheduler: starting for battle %v, duration %v", battleID, duration)

	go func() {

		select {
		case <-time.After(duration):
			_, err := s.queries.UpdateBattleStatus(ctx, db.UpdateBattleStatusParams{
				ID: battleID, Status: "upload",
			})
			if err != nil {
				log.Printf("scheduler: failed to update battle status to upload %v: %v", battleID, err)
			}
			log.Printf("scheduler: battle %v -> upload", battleID)
			s.broadcastStage(battleID, "upload")
		case <-ctx.Done():
			return
		}

		// LISTENING
		select {
		case <-time.After(2 * time.Minute):
			_, err := s.queries.UpdateBattleStatus(ctx, db.UpdateBattleStatusParams{
				ID: battleID, Status: "listening",
			})

			if err != nil {
				log.Printf("scheduler: failed to update battle status to 'listening' for battle %v: %v", battleID, err)
				return
			}
			log.Printf("scheduler: battle %v -> listening", battleID)
			s.broadcastStage(battleID, "listening")

			participants, err := s.queries.ListParticipants(ctx, battleID)
			if err != nil {
				log.Printf("scheduler: failed to list participants for battle %v: %v", battleID, err)
				return
			}
			rand.Shuffle(len(participants), func(i, j int) {
				participants[i], participants[j] = participants[j], participants[i]
			})

			listeningOrder := make([]pgtype.UUID, len(participants))
			for i, p := range participants {
				listeningOrder[i] = p.ID
			}

			_, err = s.queries.UpdateListeningOrder(ctx, db.UpdateListeningOrderParams{
				ID:             battleID,
				ListeningOrder: listeningOrder,
			})
			if err != nil {
				log.Printf("scheduler: failed to update listening order for battle %v: %v", battleID, err)
				return
			}

			for i := range listeningOrder {
				_, err = s.queries.UpdateListeningIndex(ctx, db.UpdateListeningIndexParams{
					ID:                    battleID,
					CurrentListeningIndex: int32(i),
				})
				if err != nil {
					log.Printf("scheduler: failed to update listening index %v: %v", battleID, err)
					return
				}
				select {
				case <-time.After(45 * time.Second):
				case <-ctx.Done():
					return
				}
			}

			// VOTING
			_, err = s.queries.UpdateBattleStatus(ctx, db.UpdateBattleStatusParams{
				ID:     battleID,
				Status: "voting",
			})
			if err != nil {
				log.Printf("scheduler: failed to updated battle status to voting %v: %v", battleID, err)
				return
			}
			log.Printf("scheduler: battle %v -> voting", battleID)
			s.broadcastStage(battleID, "voting")

			deadline := time.After(60 * time.Second)
			for {
				select {
				case <-deadline:
					goto done
				case <-ctx.Done():
					return
				case <-time.After(2 * time.Second):
					participants, err := s.queries.ListParticipants(ctx, battleID)
					if err != nil {
						log.Printf("scheduler: fauled to poll vote confirmations: %v", err)
						return
					}
					allConfirmed := true
					for _, p := range participants {
						if !p.VotesConfirmed {
							allConfirmed = false
							break
						}
					}
					if allConfirmed {
						goto done
					}
				}
			}
		done:

			// RESULTS
			_, err = s.queries.UpdateBattleStatus(ctx, db.UpdateBattleStatusParams{
				ID:     battleID,
				Status: "results",
			})
			if err != nil {
				log.Printf("scheduler: failed to update battle status to results %v: %v", battleID, err)
				return
			}
			log.Printf("scheduler: battle %v -> results", battleID)
			s.broadcastStage(battleID, "results")

			battle, err := s.queries.GetBattle(ctx, battleID)
			if err != nil {
				log.Printf("scheduler: failed to fetch battle for ELO: %v", err)
				return
			}

			if !battle.CreatorID.Valid {
				participants, err := s.queries.ListParticipants(ctx, battleID)
				if err != nil {
					log.Printf("scheduler: failed to ListParticipants for scoring in battle %v: %v", battleID, err)
					return
				}
				votes, err := s.queries.GetVotesForBattle(ctx, battleID)
				if err != nil {
					log.Printf("scheduler: failed to get votes for battle %v: %v", battleID, err)
				}

				// tally average score per participant
				scoreSums := make(map[pgtype.UUID]int32)
				voteCounts := make(map[pgtype.UUID]int)
				for _, v := range votes {
					scoreSums[v.VotedForParticipantID] += v.Score
					voteCounts[v.VotedForParticipantID]++
				}

				type ranked struct {
					participantID pgtype.UUID
					userID        pgtype.UUID
					avg           float64
				}

				rankedList := make([]ranked, 0, len(participants))
				for _, p := range participants {
					avg := 0.0
					if c := voteCounts[p.ID]; c > 0 {
						avg = float64(scoreSums[p.ID]) / float64(c)
					}
					rankedList = append(rankedList, ranked{participantID: p.ID, userID: p.UserID, avg: avg})
				}

				// sort descending by average score
				for i := 1; i < len(rankedList); i++ {
					for j := i; j > 0 && rankedList[j].avg > rankedList[j-1].avg; j-- {
						rankedList[j], rankedList[j-1] = rankedList[j-1], rankedList[j]
					}
				}

				// fetch each user's current ELO and compute average ELO of all participants
				N := float64(len(rankedList))
				eloRatings := make([]float64, len(rankedList))
				eloSum := 0.0
				for i, r := range rankedList {
					user, err := s.queries.GetUserByID(ctx, r.userID)
					if err != nil {
						log.Printf("scheduler: failed to get user %v for ELO: %v", r.userID, err)
						return
					}
					eloRatings[i] = float64(user.EloRating)
					eloSum += float64(user.EloRating)
				}

				// apply ELO updates
				for i, r := range rankedList {
					myElo := eloRatings[i]
					ravg := (eloSum - myElo) / (N - 1)
					actual := (N - float64(i+1)) / (N - 1)
					expected := 1.0 / (1.0 + math.Pow(10, (ravg-myElo)/400))
					delta := 32.0 * (actual - expected)
					newElo := int32(math.Round(myElo + delta))

					_, err := s.queries.UpdateUserElo(ctx, db.UpdateUserEloParams{
						ID:        r.userID,
						EloRating: newElo,
					})
					if err != nil {
						log.Printf("scheduler: failed to update ELO for user %v: %v", r.userID, err)
					}
				}
			}

			// increment battles_played for all participants; battles_won for joint 1st
			allParticipants, err := s.queries.ListParticipants(ctx, battleID)
			if err != nil {
				log.Printf("scheduler: failed to list participants for stat update: %v", err)
				return
			}
			for _, p := range allParticipants {
				if _, err := s.queries.IncrementBattlesPlayed(ctx, p.UserID); err != nil {
					log.Printf("scheduler: failed to increment battles_played for user %v: %v", p.UserID, err)
				}
			}

			votes, err := s.queries.GetVotesForBattle(ctx, battleID)
			if err != nil {
				log.Printf("scheduler: failed to get votes for battles_won update: %v", err)
				return
			}
			scoreSums := make(map[pgtype.UUID]int32)
			voteCounts := make(map[pgtype.UUID]int)
			for _, v := range votes {
				scoreSums[v.VotedForParticipantID] += v.Score
				voteCounts[v.VotedForParticipantID]++
			}
			type scored struct {
				userID pgtype.UUID
				avg    float64
			}
			scoredList := make([]scored, 0, len(allParticipants))
			for _, p := range allParticipants {
				avg := 0.0
				if c := voteCounts[p.ID]; c > 0 {
					avg = float64(scoreSums[p.ID]) / float64(c)
				}
				scoredList = append(scoredList, scored{userID: p.UserID, avg: avg})
			}
			for i := 1; i < len(scoredList); i++ {
				for j := i; j > 0 && scoredList[j].avg > scoredList[j-1].avg; j-- {
					scoredList[j], scoredList[j-1] = scoredList[j-1], scoredList[j]
				}
			}
			topScore := scoredList[0].avg
			for _, entry := range scoredList {
				if entry.avg < topScore {
					break
				}
				if _, err := s.queries.IncrementBattlesWon(ctx, entry.userID); err != nil {
					log.Printf("scheduler: failed to increment battles_won for user %v: %v", entry.userID, err)
				}
			}

		case <-ctx.Done():
			return
		}

	}()

}

func (s *Scheduler) broadcastStage(battleID pgtype.UUID, status string) {
	msg := fmt.Sprintf(`{type":"stage_change","status":"%s"}`, status)
	s.hubs.Broadcast(battleID, []byte(msg))
}
