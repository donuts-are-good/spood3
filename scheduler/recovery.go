package scheduler

import (
	"fmt"
	"log"
	"spoodblort/database"
	"spoodblort/utils"
	"time"
)

type Recovery struct {
	repo *database.Repository
}

func NewRecovery(repo *database.Repository) *Recovery {
	return &Recovery{repo: repo}
}

func (r *Recovery) VoidPastFights(tournamentID int, now time.Time) error {
	voidReasons := []string{
		"Lost to the temporal void due to server maintenance",
		"Fighters got lost in the existential dread dimension",
		"Fight cancelled due to molecular density interference",
		"Bout voided by the Department of Recreational Violence",
		"Match dissolved into pure chaos energy",
		"Fighters busy counting their fingers and toes",
		"Combat suspended due to horoscope incompatibility",
		"Fight absorbed by a nearby participation trophy",
	}

	seed := utils.VoidReasonSeed(now)
	rng := utils.NewSeededRNG(seed)

	fights, err := r.repo.GetExpiredScheduledFights(tournamentID, now)
	if err != nil {
		return fmt.Errorf("failed to get past fights: %w", err)
	}

	log.Printf("Voiding %d past fights that never happened", len(fights))

	for _, fight := range fights {
		reason := voidReasons[rng.Intn(len(voidReasons))]

		err := r.repo.VoidFight(fight.ID, reason)
		if err != nil {
			return fmt.Errorf("failed to void fight %d: %w", fight.ID, err)
		}

		err = r.repo.UpdateFighterRecords(fight.Fighter1ID, fight.Fighter2ID, "draw")
		if err != nil {
			return fmt.Errorf("failed to update fighter records for voided fight %d: %w", fight.ID, err)
		}
	}

	return nil
}

func (r *Recovery) ActivateCurrentFights(tournamentID int, now time.Time) error {
	return r.repo.ActivateCurrentFights(tournamentID, now)
}
