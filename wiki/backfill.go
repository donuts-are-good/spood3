package wiki

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"spoodblort/database"
)

// BackfillWorker processes one fight OR one fighter per minute from queue files
// and writes rich wiki pages using the wiki client. Progress is tracked by
// removing processed IDs from the queue file, one line per item.
type BackfillWorker struct {
	repo              *database.Repository
	fightsQueuePath   string
	fightersQueuePath string
	ticker            *time.Ticker
}

// NewBackfillWorker creates a worker. Paths default to project root files if empty.
func NewBackfillWorker(repo *database.Repository, fightsQueuePath, fightersQueuePath string) *BackfillWorker {
	if strings.TrimSpace(fightsQueuePath) == "" {
		fightsQueuePath = "./wiki_backfill_fights.queue"
	}
	if strings.TrimSpace(fightersQueuePath) == "" {
		fightersQueuePath = "./wiki_backfill_fighters.queue"
	}
	return &BackfillWorker{
		repo:              repo,
		fightsQueuePath:   fightsQueuePath,
		fightersQueuePath: fightersQueuePath,
		ticker:            time.NewTicker(1 * time.Minute),
	}
}

// Start runs the worker in a goroutine. It logs progress; suitable for viewing
// in the remote debugger that streams application logs.
func (w *BackfillWorker) Start() {
	go func() {
		log.Printf("[Backfill] Worker started; fights=%s fighters=%s", abs(w.fightsQueuePath), abs(w.fightersQueuePath))
		defer w.ticker.Stop()
		for range w.ticker.C {
			if err := w.step(); err != nil {
				log.Printf("[Backfill] step error: %v", err)
			}
		}
	}()
}

func (w *BackfillWorker) step() error {
	// Try a fight first; if none, try a fighter.
	if id, ok := w.popLine(w.fightsQueuePath); ok {
		return w.processFight(strings.TrimSpace(id))
	}
	if id, ok := w.popLine(w.fightersQueuePath); ok {
		return w.processFighter(strings.TrimSpace(id))
	}
	log.Printf("[Backfill] queues empty; nothing to process this minute")
	return nil
}

func (w *BackfillWorker) processFight(idStr string) error {
	id, err := parseInt(idStr)
	if err != nil {
		return fmt.Errorf("fight id not an integer: %q", idStr)
	}
	f, err := w.repo.GetFight(id)
	if err != nil {
		return fmt.Errorf("load fight %d: %w", id, err)
	}
	f1, err1 := w.repo.GetFighter(f.Fighter1ID)
	f2, err2 := w.repo.GetFighter(f.Fighter2ID)
	if err1 != nil || err2 != nil {
		return errors.New("failed to load fighters for fight")
	}
	t, _ := w.repo.GetTournament(f.TournamentID)

	client, err := New()
	if err != nil {
		return err
	}
	if t != nil {
		if err := client.UpsertFightPage(*f, *f1, *f2, t.Name); err != nil {
			return err
		}
	} else {
		if err := client.UpsertFightPage(*f, *f1, *f2, ""); err != nil {
			return err
		}
	}
	log.Printf("[Backfill] fight %d written to wiki", id)
	return nil
}

func (w *BackfillWorker) processFighter(idStr string) error {
	id, err := parseInt(idStr)
	if err != nil {
		return fmt.Errorf("fighter id not an integer: %q", idStr)
	}
	ft, err := w.repo.GetFighter(id)
	if err != nil {
		return fmt.Errorf("load fighter %d: %w", id, err)
	}
	client, err := New()
	if err != nil {
		return err
	}
	if err := client.UpsertFighterPage(*ft); err != nil {
		return err
	}
	log.Printf("[Backfill] fighter %d written to wiki", id)
	return nil
}

// popLine reads and removes the first non-empty line from a file. Returns (line,true)
// if a line was popped; otherwise ("",false). It is tolerant of missing files.
func (w *BackfillWorker) popLine(path string) (string, bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()

	var first string
	var rest []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			rest = append(rest, scanner.Text())
			continue
		}
		if first == "" {
			first = line
			// skip adding to rest
		} else {
			rest = append(rest, scanner.Text())
		}
	}
	// If nothing to do, return
	if first == "" {
		return "", false
	}
	// Write remaining lines atomically
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(strings.Join(rest, "\n")), 0o644); err != nil {
		log.Printf("[Backfill] failed to write temp file %s: %v", tmp, err)
		return "", false
	}
	if err := os.Rename(tmp, path); err != nil {
		log.Printf("[Backfill] failed to replace %s: %v", path, err)
		return "", false
	}
	return first, true
}

func parseInt(s string) (int, error) {
	var v int
	_, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &v)
	if err != nil {
		return 0, err
	}
	return v, nil
}

func abs(p string) string {
	a, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return a
}
