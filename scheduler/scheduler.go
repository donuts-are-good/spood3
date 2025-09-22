package scheduler

import (
	"database/sql"
	"fmt"
	"log"
	"os"
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
