package fight

import (
	"fmt"
	"sort"
	"spoodblort/database"
	"spoodblort/utils"
	"time"
)

type Generator struct {
	repo *database.Repository
}

func NewGenerator(repo *database.Repository) *Generator {
	return &Generator{repo: repo}
}

func (g *Generator) SelectDailyFighters(fighters []database.Fighter, date time.Time) []database.Fighter {
	seed := utils.DailyFighterSeed(date)
	rng := utils.NewSeededRNG(seed)

	available := make([]database.Fighter, len(fighters))
	copy(available, fighters)

	for i := len(available) - 1; i > 0; i-- {
		j := rng.Intn(i + 1)
		available[i], available[j] = available[j], available[i]
	}

	if len(available) > 48 {
		available = available[:48]
	}

	return available
}

func (g *Generator) GenerateFightSchedule(tournament *database.Tournament, fighters []database.Fighter, date time.Time) ([]database.Fight, error) {
	if len(fighters) < 2 {
		return nil, fmt.Errorf("need at least 2 fighters to create fights")
	}

	if len(fighters)%2 != 0 {
		fighters = fighters[:len(fighters)-1]
	}

	// Sort fighters by total combat stat points (strength + speed + endurance + technique)
	sort.Slice(fighters, func(i, j int) bool {
		ti := fighters[i].Strength + fighters[i].Speed + fighters[i].Endurance + fighters[i].Technique
		tj := fighters[j].Strength + fighters[j].Speed + fighters[j].Endurance + fighters[j].Technique
		if ti == tj {
			// Stable tie-breaker by ID to keep determinism across runs
			return fighters[i].ID < fighters[j].ID
		}
		return ti < tj
	})

	var fights []database.Fight

	startTime := time.Date(date.Year(), date.Month(), date.Day(), 12, 0, 0, 0, date.Location())

	for i := 0; i < len(fighters); i += 2 {
		if i+1 >= len(fighters) {
			break
		}

		fightTime := startTime.Add(time.Duration(i/2) * 30 * time.Minute)

		fight := database.Fight{
			TournamentID:  tournament.ID,
			Fighter1ID:    fighters[i].ID,
			Fighter2ID:    fighters[i+1].ID,
			Fighter1Name:  fighters[i].Name,
			Fighter2Name:  fighters[i+1].Name,
			ScheduledTime: fightTime,
			Status:        "scheduled",
		}

		fights = append(fights, fight)
	}

	return fights, nil
}

func (g *Generator) CreateFights(fights []database.Fight) error {
	for _, fight := range fights {
		err := g.repo.InsertFight(fight)
		if err != nil {
			return fmt.Errorf("failed to insert fight: %w", err)
		}
	}
	return nil
}
