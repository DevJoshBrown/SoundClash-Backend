package matchmaker

import (
	"context"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/DevJoshBrown/BeatBattler/internal/db"
	"github.com/DevJoshBrown/BeatBattler/internal/scheduler"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	pollInterval  = 5 * time.Second
	targetSize    = 8
	minSize       = 4
	minWaitToShip = 3 * time.Minute
	baseEloRange  = 20.0
)

type Matchmaker struct {
	queries   *db.Queries
	scheduler *scheduler.Scheduler
}

type candidate struct {
	ticket   db.BattleQueue
	elo      float64
	eloRange float64
}

func New(queries *db.Queries, s *scheduler.Scheduler) *Matchmaker {
	return &Matchmaker{queries: queries, scheduler: s}
}

func (m *Matchmaker) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(pollInterval):
			m.tick(ctx)

		}
	}
}

func (m *Matchmaker) tick(ctx context.Context) {
	tickets, err := m.queries.ListQueueTickets(ctx)
	if err != nil {
		log.Printf("matchmaker: failed to list queue: %v", err)
		return
	}
	if len(tickets) < minSize {
		return
	}
	// Elo Fetching

	candidates := make([]candidate, 0, len(tickets))
	for _, t := range tickets {
		user, err := m.queries.GetUserByID(ctx, t.UserID)
		if err != nil {
			log.Printf("matchmaker: failed to fetch user %v: %v", t.UserID, err)
			continue
		}
		waited := time.Since(t.JoinedAt.Time).Seconds()
		eloRange := baseEloRange * math.Pow(2, waited/30)
		candidates = append(candidates, candidate{ticket: t, elo: float64(user.EloRating), eloRange: eloRange})
	}

	used := make([]bool, len(candidates))

	for i, seed := range candidates {
		if used[i] {
			continue
		}

		group := []candidate{seed}
		groupGenres := seed.ticket.Genres

		for j, c := range candidates {
			if i == j || used[j] {
				continue
			}
			if !eloCompatible(seed, c) {
				continue
			}
			intersection := genreIntersection(groupGenres, c.ticket.Genres)
			if len(intersection) == 0 {
				continue
			}
			group = append(group, c)
			groupGenres = intersection
			if len(group) == targetSize {
				break
			}
		}

		oldestWait := time.Since(group[0].ticket.JoinedAt.Time)
		readyToShip := len(group) >= targetSize || (len(group) >= minSize && oldestWait > minWaitToShip)
		if !readyToShip {
			continue
		}

		genre := groupGenres[rand.Intn(len(groupGenres))]
		if err := m.ship(ctx, group, genre); err != nil {
			log.Printf("matchmaker: failed to ship battle: %v", err)
			continue
		}

		for _, c := range group {
			for k, ca := range candidates {
				if ca.ticket.ID == c.ticket.ID {
					used[k] = true
				}
			}
		}
	}
}

func (m *Matchmaker) ship(ctx context.Context, group []candidate, genre string) error {
	battle, err := m.queries.CreateBattle(ctx, db.CreateBattleParams{
		Mode:            "ranked",
		Genre:           pgtype.Text{String: genre, Valid: true},
		DurationMinutes: 20,
		MaxParticipants: int32(len(group)),
	})
	if err != nil {
		return err
	}
	log.Printf("matchmaker: created ranked battle %v with %d players, genre = %s", battle.ID, len(group), genre)

	for _, c := range group {
		_, err := m.queries.CreateParticipant(ctx, db.CreateParticipantParams{
			BattleID: battle.ID,
			UserID:   c.ticket.UserID,
		})
		if err != nil {
			log.Printf("matchmaker: failed to add participant %v: %v", c.ticket.UserID, err)
		}
	}

	ticketIDs := make([]pgtype.UUID, len(group))
	for i, c := range group {
		ticketIDs[i] = c.ticket.ID
	}
	if err := m.queries.DeleteQueueTicketsByIDs(ctx, ticketIDs); err != nil {
		log.Printf("matchmaker: failed to clear queue tickets: %v", err)
	}

	m.scheduler.Run(ctx, battle.ID, 20*time.Minute)
	return nil
}

func eloCompatible(a, b candidate) bool {
	diff := math.Abs(a.elo - b.elo)
	return diff <= math.Min(a.eloRange, b.eloRange)
}

func genreIntersection(a, b []string) []string {
	set := make(map[string]bool, len(a))
	for _, g := range a {
		set[g] = true
	}
	var out []string
	for _, g := range b {
		if set[g] {
			out = append(out, g)
		}
	}
	return out
}
