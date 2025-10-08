package main

import (
	"io"
	"log"
	"os"
	"regexp"
	"spoodblort/database"
	"spoodblort/scheduler"
	"spoodblort/web"
	"spoodblort/wiki"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

// IPMaskingWriter wraps an io.Writer to mask IP addresses in log output
type IPMaskingWriter struct {
	writer  io.Writer
	ipRegex *regexp.Regexp
}

func NewIPMaskingWriter(w io.Writer) *IPMaskingWriter {
	// Regex to match IPv4 addresses
	ipRegex := regexp.MustCompile(`\b(\d{1,3}\.\d{1,3})\.(\d{1,3}\.\d{1,3})\b`)
	return &IPMaskingWriter{
		writer:  w,
		ipRegex: ipRegex,
	}
}

func (w *IPMaskingWriter) Write(p []byte) (n int, err error) {
	// Replace IP addresses with masked versions (show only last 2 octets)
	masked := w.ipRegex.ReplaceAllFunc(p, func(match []byte) []byte {
		ip := string(match)
		return []byte(w.ipRegex.ReplaceAllString(ip, "*.*.${2}"))
	})

	return w.writer.Write(masked)
}

func main() {
	// Set up IP masking for log output
	maskedWriter := NewIPMaskingWriter(os.Stdout)
	log.SetOutput(maskedWriter)

	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	// Set up timezone
	centralTime, err := time.LoadLocation("America/Chicago")
	if err != nil {
		log.Fatal("Failed to load Central Time zone:", err)
	}

	now := time.Now().In(centralTime)
	log.Printf("🕒 Starting Spoodblort at: %s", now.Format("Monday, January 2, 2006 at 3:04:05 PM MST"))

	// Connect to database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "./spoodblort.db"
	}

	db, err := sqlx.Connect("sqlite3", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Initialize components
	repo := database.NewRepository(db)
	sched := scheduler.NewScheduler(repo)

	// Ensure today's schedule exists (skip on Sundays - Department closed)
	if now.Weekday() != time.Sunday {
		err = sched.EnsureTodaysSchedule(now)
		if err != nil {
			log.Printf("Warning: Failed to ensure today's schedule: %v", err)
		}
	} else {
		log.Printf("Skipping schedule generation - Department closed on Sundays")
	}

	// Get session secret
	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		log.Fatal("SESSION_SECRET environment variable is required")
	}

	// Start web server
	server := web.NewServer(repo, sched, sessionSecret)

	// Start background scheduler to handle fight activation
	go func() {
		ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
		defer ticker.Stop()

		for range ticker.C {
			centralTime, _ := time.LoadLocation("America/Chicago")
			now := time.Now().In(centralTime)

			// Skip all processing on Sundays - Department is closed
			if now.Weekday() == time.Sunday {
				continue
			}

			// Weekly high-roller tithe on Mondays (idempotent)
			_ = repo.TaxHighRollersIfNeeded(now)
			// Weekly sacrifice decay (idempotent)
			_ = repo.DecaySacrificesIfNeeded(now)

			// Ensure today's schedule exists (will create if missing)
			err := sched.EnsureTodaysSchedule(now)
			if err != nil {
				log.Printf("Background scheduler: Error ensuring today's schedule: %v", err)
				continue
			}

			// Get current tournament for fight processing
			tournament, err := sched.GetCurrentTournament(now)
			if err != nil {
				log.Printf("Background scheduler: Error getting tournament: %v", err)
				continue
			}

			// Activate any fights that should be starting
			err = repo.ActivateCurrentFights(tournament.ID, now)
			if err != nil {
				log.Printf("Background scheduler: Error activating fights: %v", err)
			}

			// Use the scheduler's existing engine to process active fights
			engine := sched.GetEngine()
			engine.SetBroadcaster(server.GetBroadcaster())
			err = engine.ProcessActiveFights(now)
			if err != nil {
				log.Printf("Background scheduler: Error processing active fights: %v", err)
			}

			// Saturday playoff creation (idempotent)
			_ = sched.MaybeCreateSaturdayPlayoffs(now)
		}
	}()

	// Start wiki backfill worker (1 item/min). Queues are plain text files with one ID per line.
	// Files: ./wiki_backfill_fights.queue and ./wiki_backfill_fighters.queue
	// Safe to leave empty. Progress will be logged for the debugger.
	wiki.NewBackfillWorker(repo, "", "").Start()

	// Daily Discord event sync at 4:00 AM Central
	go func() {
		for {
			centralTime, _ := time.LoadLocation("America/Chicago")
			now := time.Now().In(centralTime)
			// compute next 4:00 AM
			next := time.Date(now.Year(), now.Month(), now.Day(), 4, 0, 0, 0, centralTime)
			if !now.Before(next) {
				next = next.Add(24 * time.Hour)
			}
			dur := time.Until(next)
			timer := time.NewTimer(dur)
			<-timer.C
			// Skip Sundays
			now = time.Now().In(centralTime)
			if now.Weekday() == time.Sunday {
				continue
			}
			// Daily user credits top-up to minimum (idempotent)
			if err := repo.TopUpUsersToMinimum(1000000); err != nil {
				log.Printf("Daily top-up: error: %v", err)
			}
			// Get today's fights and sync Discord events once
			fights, err := sched.GetTodaysSchedule(now)
			if err != nil {
				log.Printf("Daily Discord sync: failed to get today's schedule: %v", err)
				continue
			}
			if len(fights) == 0 {
				// Ensure schedule then re-fetch
				if err := sched.EnsureTodaysSchedule(now); err != nil {
					log.Printf("Daily Discord sync: ensure schedule error: %v", err)
					continue
				}
				fights, _ = sched.GetTodaysSchedule(now)
			}
		}
	}()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🥊 Ready for violence!")

	// Start server (this blocks)
	if err := server.Start(port); err != nil {
		log.Fatal("Failed to start web server:", err)
	}
}
