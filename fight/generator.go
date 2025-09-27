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

	// Deterministic RNG per day/tournament to randomly flip fighter order
	flipRNG := utils.NewSeededRNG(utils.DailyFighterSeed(date) ^ int64(tournament.ID))

	startTime := time.Date(date.Year(), date.Month(), date.Day(), 12, 0, 0, 0, date.Location())

	for i := 0; i < len(fighters); i += 2 {
		if i+1 >= len(fighters) {
			break
		}

		fightTime := startTime.Add(time.Duration(i/2) * 30 * time.Minute)

		// Base pairing by neighbor, then coin flip orientation
		f1 := fighters[i]
		f2 := fighters[i+1]
		if flipRNG.Intn(2) == 1 {
			f1, f2 = f2, f1
		}

		fight := database.Fight{
			TournamentID:  tournament.ID,
			Fighter1ID:    f1.ID,
			Fighter2ID:    f2.ID,
			Fighter1Name:  f1.Name,
			Fighter2Name:  f2.Name,
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

// GenerateRoundRobinGroups creates 24 group fights (A–D) starting at the provided date (10:30 assumed by caller)
// Groups of 4 use match order: (1–2,3–4),(1–3,2–4),(1–4,2–3). 30-minute spacing.
func (g *Generator) GenerateRoundRobinGroups(tournament *database.Tournament, entrants []database.Fighter, date time.Time) ([]database.Fight, error) {
	if len(entrants) < 8 {
		return nil, fmt.Errorf("need at least 8 entrants for round-robin groups (got %d)", len(entrants))
	}

	take := func(n int) []database.Fighter {
		out := make([]database.Fighter, n)
		copy(out, entrants[:n])
		return out
	}

	var fighters []database.Fighter
	switch {
	case len(entrants) >= 16:
		fighters = take(16)
	case len(entrants) >= 12:
		fighters = take(12)
	default:
		fighters = take(8)
	}

	groupConfig := map[int]int{16: 4, 12: 3, 8: 2}
	groupCount := groupConfig[len(fighters)]
	if groupCount == 0 {
		return nil, fmt.Errorf("unsupported entrant count %d", len(fighters))
	}

	groups := make([][]database.Fighter, groupCount)
	for i, f := range fighters {
		groups[i%groupCount] = append(groups[i%groupCount], f)
	}

	matchTemplates := map[int][][]int{
		2: [][]int{
			[]int{0, 1},
		},
		3: [][]int{
			[]int{0, 1}, []int{1, 2}, []int{0, 2},
		},
		4: [][]int{
			[]int{0, 1}, []int{2, 3}, []int{0, 2}, []int{1, 3}, []int{0, 3}, []int{1, 2},
		},
	}

	slotsPerGroup := 6
	switch len(fighters) {
	case 12:
		slotsPerGroup = 8
	case 8:
		slotsPerGroup = 12
	}

	const startHour = 10
	const startMinute = 30
	start := time.Date(date.Year(), date.Month(), date.Day(), startHour, startMinute, 0, 0, date.Location())
	var fights []database.Fight

	// Deterministic RNG per day/tournament to randomly flip fighter order
	flipRNG := utils.NewSeededRNG(utils.DailyFighterSeed(date) ^ int64(tournament.ID))

	for gi, grp := range groups {
		indices := matchTemplates[len(grp)]
		if len(indices) == 0 {
			return nil, fmt.Errorf("unsupported group size %d", len(grp))
		}
		for slot := 0; slot < slotsPerGroup; slot++ {
			pair := indices[slot%len(indices)]
			f1 := grp[pair[0]]
			f2 := grp[pair[1]]
			if flipRNG.Intn(2) == 1 {
				f1, f2 = f2, f1
			}
			scheduled := start.Add(time.Duration(gi*slotsPerGroup+slot) * 30 * time.Minute)
			fights = append(fights, database.Fight{
				TournamentID:  tournament.ID,
				Fighter1ID:    f1.ID,
				Fighter2ID:    f2.ID,
				Fighter1Name:  f1.Name,
				Fighter2Name:  f2.Name,
				ScheduledTime: scheduled,
				Status:        "scheduled",
			})
		}
	}

	return fights, nil
}
