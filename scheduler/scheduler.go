package scheduler

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"spoodblort/database"
	"spoodblort/fight"
	"spoodblort/utils"
	"time"
)

type Scheduler struct {
	repo      *database.Repository
	generator *fight.Generator
	recovery  *Recovery
	engine    *fight.Engine
}

// Saturday feature flag and timing (code-level kill switch)
var SaturdayRoundRobinEnabled = true

const SaturdayStartHour = 10
const SaturdayStartMinute = 30

func NewScheduler(repo *database.Repository) *Scheduler {
	return &Scheduler{
		repo:      repo,
		generator: fight.NewGenerator(repo),
		recovery:  NewRecovery(repo),
		engine:    fight.NewEngine(repo),
	}
}

// SetBroadcaster allows setting a live broadcaster for the fight engine
func (s *Scheduler) SetBroadcaster(broadcaster fight.Broadcaster) {
	s.engine.SetBroadcaster(broadcaster)

	// If the broadcaster has a SetEngine method, connect the engine for logging
	if engineSetter, ok := broadcaster.(interface {
		SetEngine(interface{ LogAction(int, string) })
	}); ok {
		engineSetter.SetEngine(s.engine)
	}
}

// GetEngine returns the fight engine for external use
func (s *Scheduler) GetEngine() *fight.Engine {
	return s.engine
}

func (s *Scheduler) GetCurrentTournament(now time.Time) (*database.Tournament, error) {
	weekNum := utils.GetCurrentWeek(now)

	tournament, err := s.repo.GetTournamentByWeek(weekNum)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tournament for week %d not found", weekNum)
	}
	if err != nil {
		return nil, fmt.Errorf("error checking tournament: %w", err)
	}

	return tournament, nil
}

func (s *Scheduler) EnsureTodaysSchedule(now time.Time) error {
	log.Printf("Ensuring schedule for %s", now.Format("2006-01-02 15:04:05"))

	// Skip schedule generation on Sundays - the Department is closed
	if now.Weekday() == time.Sunday {
		log.Printf("Skipping schedule generation - Department closed on Sundays")

		// Clean up any incorrectly created Discord events on Sundays
		if s.engine.DiscordEvents != nil {
			go func() {
				err := s.engine.DiscordEvents.ClearAllFightEvents()
				if err != nil {
					log.Printf("Sunday cleanup: Failed to clear Discord events: %v", err)
				}
			}()
		}

		return nil
	}

	tournament, err := s.GetCurrentTournament(now)
	if err != nil {
		return fmt.Errorf("failed to get current tournament: %w", err)
	}

	log.Printf("Current tournament: %s sponsored by %s (Week %d)",
		tournament.Name, tournament.Sponsor, tournament.WeekNumber)

	today, tomorrow := utils.GetDayBounds(now)

	existingFights, err := s.repo.GetTodaysFights(tournament.ID, today, tomorrow)
	if err != nil {
		return fmt.Errorf("failed to check existing fights: %w", err)
	}

	if len(existingFights) > 0 {
		log.Printf("Found %d existing fights for today", len(existingFights))

		err = s.recovery.ActivateCurrentFights(tournament.ID, now)
		if err != nil {
			return fmt.Errorf("failed to activate current fights: %w", err)
		}

		err = s.recovery.VoidPastFights(tournament.ID, now)
		if err != nil {
			return fmt.Errorf("failed to void past fights: %w", err)
		}

		// Process any active fights
		err = s.engine.ProcessActiveFights(now)
		if err != nil {
			return fmt.Errorf("failed to process active fights: %w", err)
		}

		// Discord events are synced once daily by a dedicated scheduler

		return nil
	}

	// Saturday branch
	if now.Weekday() == time.Saturday && SaturdayRoundRobinEnabled {
		log.Printf("No fights found — generating Saturday round-robin schedule...")
		if err := s.ensureSaturdayRoundRobin(tournament, now); err != nil {
			return err
		}

		// After generation, run normal activation and processing
		if err := s.recovery.ActivateCurrentFights(tournament.ID, now); err != nil {
			return fmt.Errorf("failed to activate current fights: %w", err)
		}
		if err := s.recovery.VoidPastFights(tournament.ID, now); err != nil {
			return fmt.Errorf("failed to void past fights: %w", err)
		}
		if err := s.engine.ProcessActiveFights(now); err != nil {
			return fmt.Errorf("failed to process active fights: %w", err)
		}
		return nil
	}

	log.Printf("No fights found for today, generating new schedule...")

	allFighters, err := s.repo.GetAliveFighters()
	if err != nil {
		return fmt.Errorf("failed to get alive fighters: %w", err)
	}

	log.Printf("Found %d alive fighters", len(allFighters))

	todaysFighters := s.generator.SelectDailyFighters(allFighters, today)
	log.Printf("Selected %d fighters for today", len(todaysFighters))

	fights, err := s.generator.GenerateFightSchedule(tournament, todaysFighters, today)
	if err != nil {
		return fmt.Errorf("failed to generate fight schedule: %w", err)
	}

	log.Printf("Generated %d fights", len(fights))

	err = s.generator.CreateFights(fights)
	if err != nil {
		return fmt.Errorf("failed to insert fights: %w", err)
	}

	log.Printf("Successfully created today's fight schedule")

	err = s.recovery.ActivateCurrentFights(tournament.ID, now)
	if err != nil {
		return fmt.Errorf("failed to activate current fights: %w", err)
	}

	err = s.recovery.VoidPastFights(tournament.ID, now)
	if err != nil {
		return fmt.Errorf("failed to void past fights: %w", err)
	}

	// Process any active fights
	err = s.engine.ProcessActiveFights(now)
	if err != nil {
		return fmt.Errorf("failed to process active fights: %w", err)
	}

	// Discord events are synced once daily by a dedicated scheduler

	return nil
}

// ensureSaturdayRoundRobin creates the 24 group fights starting at 10:30 (no playoffs here)
func (s *Scheduler) ensureSaturdayRoundRobin(t *database.Tournament, now time.Time) error {
	// Determine Mon–Fri winners
	centralTime, _ := time.LoadLocation("America/Chicago")
	nowC := now.In(centralTime)
	mon, sat := utils.GetMonToFriBounds(nowC)

	wins, err := s.repo.GetCompletedFightsInRange(t.ID, mon, sat)
	if err != nil {
		return fmt.Errorf("failed to load Mon–Fri completed fights: %w", err)
	}

	// Count wins per fighter
	winCount := map[int]int{}
	fighterSeen := map[int]bool{}
	for _, f := range wins {
		if f.WinnerID.Valid {
			wid := int(f.WinnerID.Int64)
			winCount[wid]++
			fighterSeen[wid] = true
		}
	}

	// Load fighter details
	var entrants []database.Fighter
	for fid := range fighterSeen {
		ft, err := s.repo.GetFighter(fid)
		if err == nil {
			entrants = append(entrants, *ft)
		}
	}

	// Sort entrants by weekly wins desc, stable by ID
	sort.Slice(entrants, func(i, j int) bool {
		wi := winCount[entrants[i].ID]
		wj := winCount[entrants[j].ID]
		if wi == wj {
			return entrants[i].ID < entrants[j].ID
		}
		return wi > wj
	})

	// Generate 24 group fights
	fights, err := s.generator.GenerateRoundRobinGroups(t, entrants, time.Date(nowC.Year(), nowC.Month(), nowC.Day(), SaturdayStartHour, SaturdayStartMinute, 0, 0, nowC.Location()))
	if err != nil {
		return err
	}
	if err := s.generator.CreateFights(fights); err != nil {
		return err
	}
	log.Printf("Saturday: generated %d group fights", len(fights))
	return nil
}

// MaybeCreateSaturdayPlayoffs inserts semifinals/final when inputs are known. Idempotent.
func (s *Scheduler) MaybeCreateSaturdayPlayoffs(now time.Time) error {
	if now.Weekday() != time.Saturday || !SaturdayRoundRobinEnabled {
		return nil
	}

	t, err := s.GetCurrentTournament(now)
	if err != nil {
		return nil
	}
	today, tomorrow := utils.GetDayBounds(now)
	fights, err := s.repo.GetTodaysFights(t.ID, today, tomorrow)
	if err != nil {
		return nil
	}

	// Helper: slice fights in a window
	byWindow := func(startHour, startMin, endHour, endMin int) []database.Fight {
		start := time.Date(now.Year(), now.Month(), now.Day(), startHour, startMin, 0, 0, now.Location())
		end := time.Date(now.Year(), now.Month(), now.Day(), endHour, endMin, 0, 0, now.Location())
		var out []database.Fight
		for _, f := range fights {
			if !f.ScheduledTime.Before(start) && f.ScheduledTime.Before(end) {
				out = append(out, f)
			}
		}
		return out
	}

	// Group windows (6 fights x 30m)
	a := byWindow(10, 30, 13, 30)
	b := byWindow(13, 30, 16, 30)
	c := byWindow(16, 30, 19, 30)
	d := byWindow(19, 30, 22, 30)

	// Winner of a set
	winnerOf := func(fs []database.Fight) (int, bool) {
		if len(fs) < 6 {
			return 0, false
		}
		wins := map[int]int{}
		diff := map[int]int{}
		firstWin := map[int]time.Time{}
		completed := 0
		for _, f := range fs {
			if f.Status != "completed" || !f.WinnerID.Valid {
				continue
			}
			completed++
			winnerID := int(f.WinnerID.Int64)
			wins[winnerID]++
			if _, ok := firstWin[winnerID]; !ok {
				when := f.CompletedAt.Time
				if !f.CompletedAt.Valid {
					when = f.ScheduledTime
				}
				firstWin[winnerID] = when
			}
			s1, s2 := 0, 0
			if f.FinalScore1.Valid {
				s1 = int(f.FinalScore1.Int64)
			}
			if f.FinalScore2.Valid {
				s2 = int(f.FinalScore2.Int64)
			}
			diff[f.Fighter1ID] += s1 - s2
			diff[f.Fighter2ID] += s2 - s1
		}
		if completed < 6 {
			return 0, false
		}

		bestID := 0
		bestWins := -1
		bestDiff := math.MinInt32
		var bestTime time.Time
		for id := range wins {
			w := wins[id]
			d := diff[id]
			t := firstWin[id]
			if w > bestWins ||
				(w == bestWins && (d > bestDiff ||
					(d == bestDiff && (bestTime.IsZero() || t.Before(bestTime))))) {
				bestWins = w
				bestDiff = d
				bestTime = t
				bestID = id
			}
		}
		if bestID == 0 {
			return 0, false
		}
		return bestID, true
	}

	sf1Time := time.Date(now.Year(), now.Month(), now.Day(), 22, 30, 0, 0, now.Location())
	sf2Time := time.Date(now.Year(), now.Month(), now.Day(), 23, 0, 0, 0, now.Location())
	fTime := time.Date(now.Year(), now.Month(), now.Day(), 23, 30, 0, 0, now.Location())

	// SF1 A vs B
	if ok, _ := s.repo.FightExistsAt(t.ID, sf1Time); !ok {
		aw, okA := winnerOf(a)
		bw, okB := winnerOf(b)
		if okA && okB {
			af, _ := s.repo.GetFighter(aw)
			bf, _ := s.repo.GetFighter(bw)
			_ = s.generator.CreateFights([]database.Fight{{
				TournamentID:  t.ID,
				Fighter1ID:    af.ID,
				Fighter2ID:    bf.ID,
				Fighter1Name:  af.Name,
				Fighter2Name:  bf.Name,
				ScheduledTime: sf1Time,
				Status:        "scheduled",
			}})
		}
	}

	// SF2 C vs D
	if ok, _ := s.repo.FightExistsAt(t.ID, sf2Time); !ok {
		cw, okC := winnerOf(c)
		dw, okD := winnerOf(d)
		if okC && okD {
			cf, _ := s.repo.GetFighter(cw)
			df, _ := s.repo.GetFighter(dw)
			_ = s.generator.CreateFights([]database.Fight{{
				TournamentID:  t.ID,
				Fighter1ID:    cf.ID,
				Fighter2ID:    df.ID,
				Fighter1Name:  cf.Name,
				Fighter2Name:  df.Name,
				ScheduledTime: sf2Time,
				Status:        "scheduled",
			}})
		}
	}

	// Final — if both semis completed and no final exists, schedule winners
	if ok, _ := s.repo.FightExistsAt(t.ID, fTime); !ok {
		// Find the two semi fights
		var sf1, sf2 *database.Fight
		for i := range fights {
			if fights[i].ScheduledTime.Equal(sf1Time) {
				sf1 = &fights[i]
			}
			if fights[i].ScheduledTime.Equal(sf2Time) {
				sf2 = &fights[i]
			}
		}
		if sf1 != nil && sf2 != nil && sf1.Status == "completed" && sf2.Status == "completed" && sf1.WinnerID.Valid && sf2.WinnerID.Valid {
			w1 := int(sf1.WinnerID.Int64)
			w2 := int(sf2.WinnerID.Int64)
			f1, _ := s.repo.GetFighter(w1)
			f2, _ := s.repo.GetFighter(w2)
			_ = s.generator.CreateFights([]database.Fight{{
				TournamentID:  t.ID,
				Fighter1ID:    f1.ID,
				Fighter2ID:    f2.ID,
				Fighter1Name:  f1.Name,
				Fighter2Name:  f2.Name,
				ScheduledTime: fTime,
				Status:        "scheduled",
			}})
		}
	}
	return nil
}

// syncDiscordEvents syncs fight schedule with Discord Events
func (s *Scheduler) syncDiscordEvents(fights []database.Fight) error {
	if s.engine.DiscordEvents == nil {
		return nil // Discord events not configured
	}

	serverBaseURL := getServerBaseURL()
	return s.engine.DiscordEvents.SyncFightEvents(fights, serverBaseURL)
}

// SyncDiscordEventsForToday runs a one-shot Discord events sync for today's fights.
// Exported for use by main's daily scheduler.
func (s *Scheduler) SyncDiscordEventsForToday(now time.Time) error {
	// Skip Sundays
	if now.Weekday() == time.Sunday {
		return nil
	}

	fights, err := s.GetTodaysSchedule(now)
	if err != nil {
		return err
	}
	return s.syncDiscordEvents(fights)
}

// getServerBaseURL determines the server's base URL for Discord events
func getServerBaseURL() string {
	if url := os.Getenv("SERVER_BASE_URL"); url != "" {
		return url
	}

	// Fallback to localhost in development
	if os.Getenv("ENVIRONMENT") != "production" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		return fmt.Sprintf("http://localhost:%s", port)
	}

	return "https://your-domain.com"
}

func (s *Scheduler) GetTodaysSchedule(now time.Time) ([]database.Fight, error) {
	tournament, err := s.GetCurrentTournament(now)
	if err != nil {
		return nil, err
	}

	today, tomorrow := utils.GetDayBounds(now)
	return s.repo.GetTodaysFights(tournament.ID, today, tomorrow)
}

// GetNextFight finds the next upcoming fight, accounting for weekend closures
func (s *Scheduler) GetNextFight(now time.Time) (*database.Fight, error) {
	tournament, err := s.GetCurrentTournament(now)
	if err != nil {
		return nil, err
	}

	// First, check today's fights for the next scheduled one
	today, tomorrow := utils.GetDayBounds(now)
	todaysFights, err := s.repo.GetTodaysFights(tournament.ID, today, tomorrow)
	if err != nil {
		return nil, err
	}

	// Look for the next scheduled fight today
	for _, fight := range todaysFights {
		if fight.Status == "scheduled" && fight.ScheduledTime.After(now) {
			return &fight, nil
		}
	}

	// If no more fights today, check if we need to skip Sunday
	dayOfWeek := now.Weekday()

	// If it's Saturday, we need to skip Sunday and look for Monday's fights
	var nextDay time.Time
	if dayOfWeek == time.Saturday {
		// Skip Sunday, go to Monday
		nextDay = now.AddDate(0, 0, 2) // Saturday + 2 days = Monday
	} else {
		// For all other days (including Friday), just go to tomorrow
		nextDay = now.AddDate(0, 0, 1)
	}

	// Get the start and end of the next fighting day
	nextDayStart := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 0, 0, 0, 0, nextDay.Location())
	nextDayEnd := nextDayStart.Add(24 * time.Hour)

	// Check if fights exist for that day, if not try to generate them
	nextDayFights, err := s.repo.GetTodaysFights(tournament.ID, nextDayStart, nextDayEnd)
	if err != nil {
		return nil, err
	}

	// If no fights exist for the next day, we can't predict yet
	if len(nextDayFights) == 0 {
		return nil, nil
	}

	// Return the first scheduled fight of the next day
	for _, fight := range nextDayFights {
		if fight.Status == "scheduled" {
			return &fight, nil
		}
	}

	return nil, nil
}
