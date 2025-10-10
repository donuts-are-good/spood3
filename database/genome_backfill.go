package database

import (
	"log"
)

// BackfillFighterGenomes populates genome for any fighters with 'unknown' or NULL.
// Logs progress per fighter and remaining count.
func (r *Repository) BackfillFighterGenomes() error {
	// Count total needing backfill upfront
	var total int
	if err := r.db.Get(&total, `SELECT COUNT(1) FROM fighters WHERE genome IS NULL OR genome = 'unknown'`); err != nil {
		return err
	}
	if total == 0 {
		log.Println("Genome backfill: no fighters need updates")
		return nil
	}

	// Fetch in one batch; small dataset expected. For very large sets, use pagination.
	var fighters []Fighter
	if err := r.db.Select(&fighters, `SELECT * FROM fighters WHERE genome IS NULL OR genome = 'unknown' ORDER BY id ASC`); err != nil {
		return err
	}

	remaining := total
	for _, f := range fighters {
		g := f.DeriveGenome()
		if _, err := r.db.Exec(`UPDATE fighters SET genome = ? WHERE id = ?`, g, f.ID); err != nil {
			return err
		}
		remaining--
		log.Printf("genome backfill: fighter_id=%d name=\"%s\" genome=%s remaining=%d", f.ID, f.Name, g[:16], remaining)
	}

	log.Printf("genome backfill complete: updated=%d", total)
	return nil
}
