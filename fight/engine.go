package fight

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"spoodblort/database"
	"spoodblort/discord"
	"spoodblort/utils"
	"spoodblort/wiki"
	"sync"
	"time"
)

// Fight engine constants - adjust these to tune gameplay
const (
	TICK_DURATION_SECONDS = 5
	DEATH_CHANCE          = 100000 // 1 in 100000 chance per damage tick
	CRIT_CHANCE           = 2      // 1 in 2 chance per tick for the losing fighter to attempt a crit
	STARTING_HEALTH       = 100000 // Increased from 100k for longer fights
	MIN_DAMAGE            = 10
	MAX_DAMAGE            = 500 // Reduced from 5000 to balance simultaneous combat
)

// Broadcaster interface for live fight updates
type Broadcaster interface {
	BroadcastAction(fightID int, action LiveAction)
	BroadcastViewerCount(fightID int)
	BroadcastRoundClapSummary(fightID, round int)
	// ConsumeClapHealth returns the health deltas from aggregated claps for the two fighters
	// and resets the internal counters for this fight and these fighters.
	ConsumeClapHealth(fightID int, fighter1ID int, fighter2ID int) (int, int)
}

type FightState struct {
	Fighter1Health int
	Fighter2Health int
	TickNumber     int
	LastDamage1    int
	LastDamage2    int
	CurrentRound   int
	IsComplete     bool
	WinnerID       int
	DeathOccurred  bool
	// Simulation orientation bookkeeping: which DB fighter IDs correspond to the
	// Fighter1Health/Fighter2Health lanes used during simulation.
	SimFighter1ID int
	SimFighter2ID int
}

type Engine struct {
	repo             *database.Repository
	broadcaster      Broadcaster
	discordNotifier  *discord.Notifier
	roleManager      *discord.RoleManager
	liveSimulations  map[int]bool // Track which fights have live simulations running
	simulationsMutex sync.RWMutex
	// Fight logging
	fightLogs     map[int]*os.File // Track open log files for each fight
	fightLogMutex sync.Mutex
}

func NewEngine(repo *database.Repository) *Engine {
	// Check if Discord features should be disabled
	noDiscord := os.Getenv("SPOODBLORT_NO_DISCORD") != ""

	if noDiscord {
		log.Printf("🚫 Discord features disabled via SPOODBLORT_NO_DISCORD environment variable")
	}

	engine := &Engine{
		repo:            repo,
		discordNotifier: discord.NewNotifier(repo),
		liveSimulations: make(map[int]bool),
		fightLogs:       make(map[int]*os.File),
	}

	// Only initialize Role Manager if not disabled
	if !noDiscord {
		engine.roleManager = discord.NewRoleManager(repo)
	}

	return engine
}

// SetBroadcaster allows setting a live broadcaster for the engine
func (e *Engine) SetBroadcaster(broadcaster Broadcaster) {
	e.broadcaster = broadcaster
}

// GetRoleManager returns the Discord role manager
func (e *Engine) GetRoleManager() *discord.RoleManager {
	return e.roleManager
}

// AnnounceReanimationAttempt posts to Discord general when a user attempts to reanimate
func (e *Engine) AnnounceReanimationAttempt(user *database.User, fighter database.Fighter) {
	if e.discordNotifier == nil || user == nil {
		return
	}
	_ = e.discordNotifier.AnnounceReanimationAttempt(user, fighter)
}

// AnnounceNecromancerSuccess posts success and assigns Necromancer role
func (e *Engine) AnnounceNecromancerSuccess(user *database.User, fighter database.Fighter) {
	if user == nil {
		return
	}
	if e.discordNotifier != nil {
		_ = e.discordNotifier.AnnounceNecromancer(user, fighter)
	}
	if e.roleManager != nil {
		_ = e.roleManager.AssignNecromancerRole(user)
	}
}

// initFightLog creates a log file for the fight and writes the header
func (e *Engine) initFightLog(fight database.Fight, fighter1, fighter2 database.Fighter) error {
	e.fightLogMutex.Lock()
	defer e.fightLogMutex.Unlock()

	// Create logs directory if it doesn't exist
	logsDir := "fight_logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Format: YYYY-MM-DD-Fight12-sampson-vs-timber.txt
	date := fight.ScheduledTime.Format("2006-01-02")
	sanitizedName1 := sanitizeName(fighter1.Name)
	sanitizedName2 := sanitizeName(fighter2.Name)
	filename := fmt.Sprintf("%s-Fight%d-%s-vs-%s.txt", date, fight.ID, sanitizedName1, sanitizedName2)
	filepath := filepath.Join(logsDir, filename)

	// Create/open the file
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}

	e.fightLogs[fight.ID] = file

	// Get tournament info
	tournament, err := e.repo.GetTournament(fight.TournamentID)
	if err != nil {
		log.Printf("Failed to get tournament for fight log: %v", err)
		tournament = &database.Tournament{Name: "Unknown Tournament", Sponsor: "Unknown Sponsor"}
	}

	// Write header
	_, err = file.WriteString(fmt.Sprintf("%s vs %s\n", fighter1.Name, fighter2.Name))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("%s, Sponsored by %s\n\n", tournament.Name, tournament.Sponsor))
	if err != nil {
		return err
	}

	log.Printf("Created fight log: %s", filepath)
	return nil
}

// logFightAction writes an action to the fight log
func (e *Engine) logFightAction(fightID int, text string) {
	e.fightLogMutex.Lock()
	defer e.fightLogMutex.Unlock()

	if file, exists := e.fightLogs[fightID]; exists {
		file.WriteString(text + "\n")
		file.Sync() // Ensure it's written immediately
	}
}

// closeFightLog closes and removes the log file from tracking
func (e *Engine) closeFightLog(fightID int) {
	e.fightLogMutex.Lock()
	defer e.fightLogMutex.Unlock()

	if file, exists := e.fightLogs[fightID]; exists {
		file.Close()
		delete(e.fightLogs, fightID)
	}
}

// sanitizeName removes characters that aren't safe for filenames
func sanitizeName(name string) string {
	// Replace spaces and unsafe characters with hyphens
	result := ""
	for _, char := range name {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') {
			result += string(char)
		} else if char == ' ' || char == '_' {
			result += "-"
		}
		// Skip other characters
	}
	return result
}

// SimulateFightFromStart runs a complete fight simulation from the beginning
func (e *Engine) SimulateFightFromStart(fight database.Fight, fighter1, fighter2 database.Fighter) (*FightState, error) {
	log.Printf("Starting fight simulation: %s vs %s", fighter1.Name, fighter2.Name)

	// Orientation swap disabled

	// Apply stat effects to fighters for this fight's date
	modifiedFighter1 := e.applyStatEffectsToFighter(fighter1, fight.ScheduledTime)
	modifiedFighter2 := e.applyStatEffectsToFighter(fighter2, fight.ScheduledTime)

	// Calculate starting health (keeping original health calculation for now)
	fighter1Health := e.calculateFighterHealthForDate(fighter1.ID, fight.ScheduledTime)
	fighter2Health := e.calculateFighterHealthForDate(fighter2.ID, fight.ScheduledTime)

	state := &FightState{
		Fighter1Health: fighter1Health,
		Fighter2Health: fighter2Health,
		TickNumber:     0,
		CurrentRound:   1,
	}

	maxTicks := (30 * 60) / TICK_DURATION_SECONDS // 30 minutes worth of ticks

	for tick := 1; tick <= maxTicks && !state.IsComplete; tick++ {
		e.simulateTick(fight.ID, tick, modifiedFighter1, modifiedFighter2, state)

		if tick%6 == 0 { // Every minute, check for round progression
			state.CurrentRound++
		}
	}

	// If fight went the distance, determine winner by remaining health
	if !state.IsComplete {
		if state.Fighter1Health > state.Fighter2Health {
			state.WinnerID = fighter1.ID
		} else if state.Fighter2Health > state.Fighter1Health {
			state.WinnerID = fighter2.ID
		}
		// If exactly equal health, it's a draw (WinnerID stays 0)
		state.IsComplete = true
		log.Printf("Fight went to judges decision")
	}

	return state, nil
}

// CatchUpSimulation simulates a fight from start to current time for recovery
func (e *Engine) CatchUpSimulation(fight database.Fight, fighter1, fighter2 database.Fighter, now time.Time) (*FightState, error) {
	elapsed := now.Sub(fight.ScheduledTime)
	elapsedSeconds := int(elapsed.Seconds())
	targetTick := elapsedSeconds / TICK_DURATION_SECONDS

	log.Printf("Catching up fight simulation: %d ticks elapsed", targetTick)

	// Orientation swap disabled

	// Apply stat effects to fighters for this fight's date
	modifiedFighter1 := e.applyStatEffectsToFighter(fighter1, fight.ScheduledTime)
	modifiedFighter2 := e.applyStatEffectsToFighter(fighter2, fight.ScheduledTime)

	// Calculate starting health with applied effects from the fight's scheduled date
	fighter1Health := e.calculateFighterHealthForDate(fighter1.ID, fight.ScheduledTime)
	fighter2Health := e.calculateFighterHealthForDate(fighter2.ID, fight.ScheduledTime)

	state := &FightState{
		Fighter1Health: fighter1Health,
		Fighter2Health: fighter2Health,
		TickNumber:     0,
		CurrentRound:   1,
	}

	// Simulate all elapsed ticks at once
	for tick := 1; tick <= targetTick && !state.IsComplete; tick++ {
		e.simulateTick(fight.ID, tick, modifiedFighter1, modifiedFighter2, state)

		if tick%6 == 0 {
			state.CurrentRound++
		}
	}

	return state, nil
}

// StartLiveFightSimulation begins real-time simulation for an active fight
func (e *Engine) StartLiveFightSimulation(fight database.Fight, fighter1, fighter2 database.Fighter) error {
	// Check if simulation is already running for this fight
	e.simulationsMutex.Lock()
	if e.liveSimulations[fight.ID] {
		e.simulationsMutex.Unlock()
		log.Printf("Live simulation already running for fight %d", fight.ID)
		return nil
	}
	e.liveSimulations[fight.ID] = true
	e.simulationsMutex.Unlock()

	log.Printf("Starting live simulation for fight %d: %s vs %s", fight.ID, fighter1.Name, fighter2.Name)

	// Initialize fight log
	// Orientation swap disabled

	err := e.initFightLog(fight, fighter1, fighter2)
	if err != nil {
		log.Printf("Failed to initialize fight log: %v", err)
		// Continue without logging
	}

	// Calculate how many ticks have already passed since fight started
	elapsed := time.Since(fight.ScheduledTime)
	elapsedTicks := int(elapsed.Seconds()) / TICK_DURATION_SECONDS

	// Apply stat effects to fighters for today (live fights use today's effects)
	centralTime, _ := time.LoadLocation("America/Chicago")
	now := time.Now().In(centralTime)
	modifiedFighter1 := e.applyStatEffectsToFighter(fighter1, now)
	modifiedFighter2 := e.applyStatEffectsToFighter(fighter2, now)

	// Calculate starting health with applied effects from today
	fighter1Health := e.calculateFighterHealthForDate(fighter1.ID, now)
	fighter2Health := e.calculateFighterHealthForDate(fighter2.ID, now)

	// Create initial state
	state := &FightState{
		Fighter1Health: fighter1Health,
		Fighter2Health: fighter2Health,
		TickNumber:     0,
		CurrentRound:   1,
	}

	// Catch up to current time without broadcasting (for consistency)
	for tick := 1; tick <= elapsedTicks && !state.IsComplete; tick++ {
		e.simulateTickQuiet(fight.ID, tick, modifiedFighter1, modifiedFighter2, state)
		if tick%6 == 0 {
			state.CurrentRound++
		}
	}

	// If fight is already complete, finish it
	if state.IsComplete {
		e.simulationsMutex.Lock()
		delete(e.liveSimulations, fight.ID)
		e.simulationsMutex.Unlock()
		return e.CompleteFight(fight, state)
	}

	// Start real-time broadcasting from current state
	go e.broadcastLiveFight(fight, modifiedFighter1, modifiedFighter2, state)

	return nil
}

// broadcastLiveFight runs the live fight simulation in a goroutine
func (e *Engine) broadcastLiveFight(fight database.Fight, fighter1, fighter2 database.Fighter, state *FightState) {
	defer func() {
		e.simulationsMutex.Lock()
		delete(e.liveSimulations, fight.ID)
		e.simulationsMutex.Unlock()
		log.Printf("Live simulation ended for fight %d", fight.ID)
	}()

	ticker := time.NewTicker(TICK_DURATION_SECONDS * time.Second)
	defer ticker.Stop()

	// Broadcast initial viewer count
	if e.broadcaster != nil {
		e.broadcaster.BroadcastViewerCount(fight.ID)
	}

	maxTicks := (30 * 60) / TICK_DURATION_SECONDS // 30 minutes worth of ticks

	for !state.IsComplete && state.TickNumber < maxTicks {
		select {
		case <-ticker.C:
			state.TickNumber++

			// Simulate this tick with broadcasting
			e.simulateTick(fight.ID, state.TickNumber, fighter1, fighter2, state)

			// Check for round progression
			if state.TickNumber%6 == 0 {
				state.CurrentRound++
				// Broadcast round change
				if e.broadcaster != nil {
					roundAction := GenerateRoundAction(state.CurrentRound, state.Fighter1Health, state.Fighter2Health)
					e.broadcaster.BroadcastAction(fight.ID, roundAction)

					// Log the round action
					e.logFightAction(fight.ID, roundAction.Action)
					if roundAction.Commentary != "" {
						e.logFightAction(fight.ID, fmt.Sprintf("%s: \"%s\"", roundAction.Announcer, roundAction.Commentary))
					}

					// Broadcast clap summary for the previous round if it was a clapping round
					e.broadcaster.BroadcastRoundClapSummary(fight.ID, state.CurrentRound)
				}
			}

			// If fight is complete, finish it
			if state.IsComplete {
				log.Printf("Live fight %d completed, finishing...", fight.ID)
				e.CompleteFight(fight, state)
				return
			}
		}
	}

	// If fight went the distance, complete it
	if !state.IsComplete {
		if state.Fighter1Health > state.Fighter2Health {
			state.WinnerID = fighter1.ID
		} else if state.Fighter2Health > state.Fighter1Health {
			state.WinnerID = fighter2.ID
		}
		state.IsComplete = true
		log.Printf("Live fight %d went to judges decision", fight.ID)
		e.CompleteFight(fight, state)
	}
}

// simulateTick runs one 10-second combat tick
func (e *Engine) simulateTick(fightID, tickNumber int, fighter1, fighter2 database.Fighter, state *FightState) {
	seed := utils.FightTickSeed(fightID, tickNumber)
	rng := utils.NewSeededRNG(seed)

	// Determine advantage using a random stat and coinflip aggregation
	fighter1Advantage := e.determineStatBasedAdvantage(fighter1, fighter2, rng)

	// Undead frenzy (25% chance). If frenzied, zero one stat and apply 2x/3x/4x multiplier to outgoing damage.
	frenzy1 := false
	frenzy2 := false
	frenzy1Mult := 1
	frenzy2Mult := 1
	frenzy1Zero := ""
	frenzy2Zero := ""
	if fighter1.IsUndead && rng.Intn(4) == 0 {
		frenzy1 = true
		switch rng.Intn(4) {
		case 0:
			fighter1.Strength, frenzy1Zero = 0, "strength"
		case 1:
			fighter1.Speed, frenzy1Zero = 0, "speed"
		case 2:
			fighter1.Endurance, frenzy1Zero = 0, "endurance"
		default:
			fighter1.Technique, frenzy1Zero = 0, "technique"
		}
		switch rng.Intn(3) {
		case 0:
			frenzy1Mult = 2
		case 1:
			frenzy1Mult = 3
		default:
			frenzy1Mult = 4
		}
		fighter1Advantage = e.determineStatBasedAdvantage(fighter1, fighter2, rng)
	}
	if fighter2.IsUndead && rng.Intn(4) == 0 {
		frenzy2 = true
		switch rng.Intn(4) {
		case 0:
			fighter2.Strength, frenzy2Zero = 0, "strength"
		case 1:
			fighter2.Speed, frenzy2Zero = 0, "speed"
		case 2:
			fighter2.Endurance, frenzy2Zero = 0, "endurance"
		default:
			fighter2.Technique, frenzy2Zero = 0, "technique"
		}
		switch rng.Intn(3) {
		case 0:
			frenzy2Mult = 2
		case 1:
			frenzy2Mult = 3
		default:
			frenzy2Mult = 4
		}
		fighter1Advantage = e.determineStatBasedAdvantage(fighter1, fighter2, rng)
	}

	// Calculate base damage for both fighters (simultaneous combat)
	baseDamage1 := e.calculateDamage(fighter1, rng)
	baseDamage2 := e.calculateDamage(fighter2, rng)
	if frenzy2 {
		baseDamage1 *= frenzy2Mult
	}
	if frenzy1 {
		baseDamage2 *= frenzy1Mult
	}

	var damage1, damage2 int
	if fighter1Advantage {
		// Fighter1 wins this exchange - gets full damage, Fighter2 gets reduced damage
		damage1 = 0
		damage2 = baseDamage1 + (baseDamage1 / 2) // Winner gets 150% damage
		// Fighter1 still takes some damage back but reduced
		damage1 = baseDamage2 / 3 // Loser deals 33% damage back
	} else {
		// Fighter2 wins this exchange - gets full damage, Fighter1 gets reduced damage
		damage2 = 0
		damage1 = baseDamage2 + (baseDamage2 / 2) // Winner gets 150% damage
		// Fighter2 still takes some damage back but reduced
		damage2 = baseDamage1 / 3 // Loser deals 33% damage back
	}

	// Apply damage
	state.Fighter1Health -= damage1
	state.Fighter2Health -= damage2
	state.LastDamage1 = damage1
	state.LastDamage2 = damage2
	state.TickNumber = tickNumber

	// Apply aggregated clap healing after damage is applied for this tick
	if e.broadcaster != nil {
		delta1, delta2 := e.broadcaster.ConsumeClapHealth(fightID, fighter1.ID, fighter2.ID)
		if delta1 > 0 {
			healed := state.Fighter1Health + delta1
			if healed > STARTING_HEALTH {
				healed = STARTING_HEALTH
			}
			state.Fighter1Health = healed
		}
		if delta2 > 0 {
			healed := state.Fighter2Health + delta2
			if healed > STARTING_HEALTH {
				healed = STARTING_HEALTH
			}
			state.Fighter2Health = healed
		}
	}

	// Generate and broadcast live action if broadcaster is available
	if e.broadcaster != nil {
		action := GenerateLiveAction(fightID, tickNumber, fighter1, fighter2, damage1, damage2, state.Fighter1Health, state.Fighter2Health, state.CurrentRound)
		if frenzy1 {
			action.Frenzy1, action.Frenzy1Mult, action.Frenzy1Zero = true, frenzy1Mult, frenzy1Zero
		}
		if frenzy2 {
			action.Frenzy2, action.Frenzy2Mult, action.Frenzy2Zero = true, frenzy2Mult, frenzy2Zero
		}
		e.broadcaster.BroadcastAction(fightID, action)

		// Log the action to file
		e.logFightAction(fightID, action.Action)
		if action.Commentary != "" {
			e.logFightAction(fightID, fmt.Sprintf("%s: \"%s\"", action.Announcer, action.Commentary))
		}
	}

	// Comeback Critical: only if someone is losing by >= 3000 health
	var critAttacker *database.Fighter
	healthDiff := state.Fighter2Health - state.Fighter1Health // positive means fighter1 is behind
	if healthDiff >= 3000 {
		critAttacker = &fighter1
	} else if healthDiff <= -3000 {
		critAttacker = &fighter2
	}

	if critAttacker != nil && rng.Intn(CRIT_CHANCE) == 0 {
		critDmg := e.calculateCritDamage(rng)
		if critDmg > 0 {
			if critAttacker == &fighter1 {
				// Fighter1 crits Fighter2
				state.Fighter2Health -= critDmg
				state.LastDamage2 += critDmg
				// Lifesteal: attacker recovers half the crit damage (capped)
				if !fighter2.IsUndead {
					heal := critDmg / 2
					if heal > 0 {
						healed := state.Fighter1Health + heal
						if healed > STARTING_HEALTH {
							healed = STARTING_HEALTH
						}
						state.Fighter1Health = healed
					}
				}
				if e.broadcaster != nil {
					critAction := LiveAction{
						Type:       "critical",
						Action:     fmt.Sprintf("COMEBACK CRIT! %s detonates %s for %s bonus damage!", fighter1.Name, fighter2.Name, formatNumber(critDmg)),
						Damage:     critDmg,
						Attacker:   fighter1.Name,
						Victim:     fighter2.Name,
						Commentary: "",
						Announcer:  "\"Screaming\" Sally Bloodworth",
						Health1:    state.Fighter1Health,
						Health2:    state.Fighter2Health,
						Round:      state.CurrentRound,
						TickNumber: tickNumber,
					}
					e.broadcaster.BroadcastAction(fightID, critAction)
					e.logFightAction(fightID, critAction.Action)
				}
				// Extra death chance due to crit damage on Fighter2
				if e.checkDeath(rng) {
					state.DeathOccurred = true
					state.IsComplete = true
					state.WinnerID = fighter1.ID
					if e.broadcaster != nil {
						deathAction := GenerateDeathAction(fightID, fighter1, fighter2, state.Fighter1Health, state.Fighter2Health, state.CurrentRound)
						e.broadcaster.BroadcastAction(fightID, deathAction)
						e.logFightAction(fightID, deathAction.Action)
						if deathAction.Commentary != "" {
							e.logFightAction(fightID, fmt.Sprintf("%s: \"%s\"", deathAction.Announcer, deathAction.Commentary))
						}
					}
					return
				}
			} else {
				// Fighter2 crits Fighter1
				state.Fighter1Health -= critDmg
				state.LastDamage1 += critDmg
				// Lifesteal: attacker recovers half the crit damage (capped)
				if !fighter1.IsUndead {
					heal := critDmg / 2
					if heal > 0 {
						healed := state.Fighter2Health + heal
						if healed > STARTING_HEALTH {
							healed = STARTING_HEALTH
						}
						state.Fighter2Health = healed
					}
				}
				if e.broadcaster != nil {
					critAction := LiveAction{
						Type:       "critical",
						Action:     fmt.Sprintf("COMEBACK CRIT! %s detonates %s for %s bonus damage!", fighter2.Name, fighter1.Name, formatNumber(critDmg)),
						Damage:     critDmg,
						Attacker:   fighter2.Name,
						Victim:     fighter1.Name,
						Commentary: "",
						Announcer:  "\"Screaming\" Sally Bloodworth",
						Health1:    state.Fighter1Health,
						Health2:    state.Fighter2Health,
						Round:      state.CurrentRound,
						TickNumber: tickNumber,
					}
					e.broadcaster.BroadcastAction(fightID, critAction)
					e.logFightAction(fightID, critAction.Action)
				}
				// Extra death chance due to crit damage on Fighter1
				if e.checkDeath(rng) {
					state.DeathOccurred = true
					state.IsComplete = true
					state.WinnerID = fighter2.ID
					if e.broadcaster != nil {
						deathAction := GenerateDeathAction(fightID, fighter2, fighter1, state.Fighter1Health, state.Fighter2Health, state.CurrentRound)
						e.broadcaster.BroadcastAction(fightID, deathAction)
						e.logFightAction(fightID, deathAction.Action)
						if deathAction.Commentary != "" {
							e.logFightAction(fightID, fmt.Sprintf("%s: \"%s\"", deathAction.Announcer, deathAction.Commentary))
						}
					}
					return
				}
			}
		}
	}

	// Check for death (only if damage was dealt)
	if damage1 > 0 && e.checkDeath(rng) {
		state.DeathOccurred = true
		state.IsComplete = true
		state.WinnerID = fighter2.ID
		log.Printf("%s died from damage!", fighter1.Name)

		// Broadcast death action
		if e.broadcaster != nil {
			deathAction := GenerateDeathAction(fightID, fighter2, fighter1, state.Fighter1Health, state.Fighter2Health, state.CurrentRound)
			e.broadcaster.BroadcastAction(fightID, deathAction)

			// Log the death action
			e.logFightAction(fightID, deathAction.Action)
			if deathAction.Commentary != "" {
				e.logFightAction(fightID, fmt.Sprintf("%s: \"%s\"", deathAction.Announcer, deathAction.Commentary))
			}
		}
		return
	}

	if damage2 > 0 && e.checkDeath(rng) {
		state.DeathOccurred = true
		state.IsComplete = true
		state.WinnerID = fighter1.ID
		log.Printf("%s died from damage!", fighter2.Name)

		// Broadcast death action
		if e.broadcaster != nil {
			deathAction := GenerateDeathAction(fightID, fighter1, fighter2, state.Fighter1Health, state.Fighter2Health, state.CurrentRound)
			e.broadcaster.BroadcastAction(fightID, deathAction)

			// Log the death action
			e.logFightAction(fightID, deathAction.Action)
			if deathAction.Commentary != "" {
				e.logFightAction(fightID, fmt.Sprintf("%s: \"%s\"", deathAction.Announcer, deathAction.Commentary))
			}
		}
		return
	}

	// Check for KO
	if state.Fighter1Health <= 0 {
		state.IsComplete = true
		state.WinnerID = fighter2.ID
		log.Printf("%s won by KO!", fighter2.Name)

		// Broadcast KO action
		if e.broadcaster != nil {
			koAction := GenerateDeathAction(fightID, fighter2, fighter1, state.Fighter1Health, state.Fighter2Health, state.CurrentRound)
			e.broadcaster.BroadcastAction(fightID, koAction)

			// Log the KO action
			e.logFightAction(fightID, koAction.Action)
			if koAction.Commentary != "" {
				e.logFightAction(fightID, fmt.Sprintf("%s: \"%s\"", koAction.Announcer, koAction.Commentary))
			}
		}
		return
	}

	if state.Fighter2Health <= 0 {
		state.IsComplete = true
		state.WinnerID = fighter1.ID
		log.Printf("%s won by KO!", fighter1.Name)

		// Broadcast KO action
		if e.broadcaster != nil {
			koAction := GenerateDeathAction(fightID, fighter1, fighter2, state.Fighter1Health, state.Fighter2Health, state.CurrentRound)
			e.broadcaster.BroadcastAction(fightID, koAction)

			// Log the KO action
			e.logFightAction(fightID, koAction.Action)
			if koAction.Commentary != "" {
				e.logFightAction(fightID, fmt.Sprintf("%s: \"%s\"", koAction.Announcer, koAction.Commentary))
			}
		}
		return
	}
}

// simulateTickQuiet runs a tick without broadcasting (for catch-up)
func (e *Engine) simulateTickQuiet(fightID, tickNumber int, fighter1, fighter2 database.Fighter, state *FightState) {
	seed := utils.FightTickSeed(fightID, tickNumber)
	rng := utils.NewSeededRNG(seed)

	// Determine advantage using a random stat and coinflip aggregation
	fighter1Advantage := e.determineStatBasedAdvantage(fighter1, fighter2, rng)

	// Calculate base damage for both fighters (simultaneous combat)
	baseDamage1 := e.calculateDamage(fighter1, rng)
	baseDamage2 := e.calculateDamage(fighter2, rng)

	var damage1, damage2 int
	if fighter1Advantage {
		// Fighter1 wins this exchange - gets full damage, Fighter2 gets reduced damage
		damage1 = 0
		damage2 = baseDamage1 + (baseDamage1 / 2) // Winner gets 150% damage
		// Fighter1 still takes some damage back but reduced
		damage1 = baseDamage2 / 3 // Loser deals 33% damage back
	} else {
		// Fighter2 wins this exchange - gets full damage, Fighter1 gets reduced damage
		damage2 = 0
		damage1 = baseDamage2 + (baseDamage2 / 2) // Winner gets 150% damage
		// Fighter2 still takes some damage back but reduced
		damage2 = baseDamage1 / 3 // Loser deals 33% damage back
	}

	// Apply damage
	state.Fighter1Health -= damage1
	state.Fighter2Health -= damage2
	state.LastDamage1 = damage1
	state.LastDamage2 = damage2
	state.TickNumber = tickNumber

	// Check for death (only if damage was dealt)
	if damage1 > 0 && e.checkDeath(rng) {
		state.DeathOccurred = true
		state.IsComplete = true
		state.WinnerID = fighter2.ID
		return
	}

	if damage2 > 0 && e.checkDeath(rng) {
		state.DeathOccurred = true
		state.IsComplete = true
		state.WinnerID = fighter1.ID
		return
	}

	// Comeback Critical (quiet): only if someone is losing by >= 3000 health
	var critAttacker *database.Fighter
	healthDiff := state.Fighter2Health - state.Fighter1Health // positive means fighter1 is behind
	if healthDiff >= 3000 {
		critAttacker = &fighter1
	} else if healthDiff <= -3000 {
		critAttacker = &fighter2
	}

	if critAttacker != nil && rng.Intn(CRIT_CHANCE) == 0 {
		critDmg := e.calculateCritDamage(rng)
		if critDmg > 0 {
			if critAttacker == &fighter1 {
				state.Fighter2Health -= critDmg
				state.LastDamage2 += critDmg
				// Lifesteal: attacker recovers half the crit damage (capped)
				heal := critDmg / 2
				if heal > 0 {
					healed := state.Fighter1Health + heal
					if healed > STARTING_HEALTH {
						healed = STARTING_HEALTH
					}
					state.Fighter1Health = healed
				}
				if e.checkDeath(rng) {
					state.DeathOccurred = true
					state.IsComplete = true
					state.WinnerID = fighter1.ID
					return
				}
			} else {
				state.Fighter1Health -= critDmg
				state.LastDamage1 += critDmg
				// Lifesteal: attacker recovers half the crit damage (capped)
				heal := critDmg / 2
				if heal > 0 {
					healed := state.Fighter2Health + heal
					if healed > STARTING_HEALTH {
						healed = STARTING_HEALTH
					}
					state.Fighter2Health = healed
				}
				if e.checkDeath(rng) {
					state.DeathOccurred = true
					state.IsComplete = true
					state.WinnerID = fighter2.ID
					return
				}
			}
		}
	}

	// Check for KO
	if state.Fighter1Health <= 0 {
		state.IsComplete = true
		state.WinnerID = fighter2.ID
		return
	}

	if state.Fighter2Health <= 0 {
		state.IsComplete = true
		state.WinnerID = fighter1.ID
		return
	}
}

// calculateDamage calculates damage dealt by winning fighter
func (e *Engine) calculateDamage(winner database.Fighter, rng *rand.Rand) int {
	// Base damage with some randomness
	baseDamage := MIN_DAMAGE + rng.Intn(MAX_DAMAGE-MIN_DAMAGE)

	// Modify by strength (higher strength = more damage)
	strengthMultiplier := 1.0 + (float64(winner.Strength)/100.0)*0.5

	finalDamage := int(float64(baseDamage) * strengthMultiplier)

	return int(math.Max(float64(MIN_DAMAGE), float64(finalDamage)))
}

// calculateCritDamage rolls 5d20 and maps number of success dice to bonus damage.
// We treat a die as a success if it rolls >= 10 (50% success) to get a decent spread.
// 1 success → 1000, 2 → 2000, 3 → 4000, 4 → 8000, 5 → 10000. 0 → 0.
func (e *Engine) calculateCritDamage(rng *rand.Rand) int {
	successes := 0
	for i := 0; i < 5; i++ {
		roll := rng.Intn(20) + 1 // 1..20
		if roll == 20 {          // natural 20 counts as a success
			successes++
		}
	}
	switch successes {
	case 1:
		return 5000
	case 2:
		return 10000
	case 3:
		return 15000
	case 4:
		return 20000
	case 5:
		return 100000
	default:
		return 0
	}
}

// checkDeath determines if death occurs this tick
func (e *Engine) checkDeath(rng *rand.Rand) bool {
	return rng.Intn(DEATH_CHANCE) == 0
}

// determineStatBasedAdvantage picks a random combat stat and gives advantage to the fighter
// with more "heads" from coin flips equal to that stat value. Ties break randomly.
func (e *Engine) determineStatBasedAdvantage(f1, f2 database.Fighter, rng *rand.Rand) bool {
	// Choose a stat: 0=strength, 1=speed, 2=endurance, 3=technique
	statIdx := rng.Intn(4)

	var v1, v2 int
	switch statIdx {
	case 0:
		v1, v2 = f1.Strength, f2.Strength
	case 1:
		v1, v2 = f1.Speed, f2.Speed
	case 2:
		v1, v2 = f1.Endurance, f2.Endurance
	default:
		v1, v2 = f1.Technique, f2.Technique
	}

	// Bound to non-negative just in case
	if v1 < 0 {
		v1 = 0
	}
	if v2 < 0 {
		v2 = 0
	}

	heads1 := 0
	heads2 := 0

	for i := 0; i < v1; i++ {
		if rng.Intn(2) == 0 {
			heads1++
		}
	}
	for i := 0; i < v2; i++ {
		if rng.Intn(2) == 0 {
			heads2++
		}
	}

	if heads1 == heads2 {
		// random tie-breaker
		return rng.Intn(2) == 0
	}
	return heads1 > heads2
}

// calculateFighterHealthForDate calculates starting health (base health only, no effect modifications)
func (e *Engine) calculateFighterHealthForDate(fighterID int, effectDate time.Time) int {
	baseHealth := STARTING_HEALTH

	// Blessings and curses now only affect stats, not health
	// Health remains at the base value for all fighters
	log.Printf("Fighter %d starting health for date %s: %d (base health, no effect modifications)",
		fighterID, effectDate.Format("2006-01-02"), baseHealth)

	return baseHealth
}

// applyStatEffectsToFighter applies stat-based effects to a fighter's stats
func (e *Engine) applyStatEffectsToFighter(fighter database.Fighter, effectDate time.Time) database.Fighter {
	modifiedFighter := fighter

	// Get day bounds for the effect date
	startDate := time.Date(effectDate.Year(), effectDate.Month(), effectDate.Day(), 0, 0, 0, 0, effectDate.Location())
	endDate := startDate.Add(24 * time.Hour)

	// Get applied effects for this fighter on the specific date
	effects, err := e.repo.GetAppliedEffectsForDate("fighter", fighter.ID, startDate, endDate)
	if err != nil {
		log.Printf("Error getting applied effects for fighter %d on date %s: %v", fighter.ID, effectDate.Format("2006-01-02"), err)
		return modifiedFighter
	}

	// Apply stat modifications
	for _, effect := range effects {
		switch effect.EffectType {
		case "strength_blessing":
			modifiedFighter.Strength += effect.EffectValue
		case "strength_curse":
			modifiedFighter.Strength -= effect.EffectValue
		case "speed_blessing":
			modifiedFighter.Speed += effect.EffectValue
		case "speed_curse":
			modifiedFighter.Speed -= effect.EffectValue
		case "endurance_blessing":
			modifiedFighter.Endurance += effect.EffectValue
		case "endurance_curse":
			modifiedFighter.Endurance -= effect.EffectValue
		case "technique_blessing":
			modifiedFighter.Technique += effect.EffectValue
		case "technique_curse":
			modifiedFighter.Technique -= effect.EffectValue
		}
	}

	// Ensure stats don't go below minimum values
	if modifiedFighter.Strength < 1 {
		modifiedFighter.Strength = 1
	}
	if modifiedFighter.Speed < 1 {
		modifiedFighter.Speed = 1
	}
	if modifiedFighter.Endurance < 1 {
		modifiedFighter.Endurance = 1
	}
	if modifiedFighter.Technique < 1 {
		modifiedFighter.Technique = 1
	}

	log.Printf("Fighter %s stats modified - Str: %d->%d, Spd: %d->%d, End: %d->%d, Tech: %d->%d",
		fighter.Name, fighter.Strength, modifiedFighter.Strength, fighter.Speed, modifiedFighter.Speed,
		fighter.Endurance, modifiedFighter.Endurance, fighter.Technique, modifiedFighter.Technique)

	return modifiedFighter
}

// CompleteFight finishes a fight and persists results to database
func (e *Engine) CompleteFight(fight database.Fight, state *FightState) error {
	log.Printf("Completing fight %d: final scores %d-%d", fight.ID, state.Fighter1Health, state.Fighter2Health)

	// Log the final result
	if state.WinnerID != 0 {
		// Get the winner's name
		var winnerName string
		if state.WinnerID == fight.Fighter1ID {
			winnerName = fight.Fighter1Name
		} else {
			winnerName = fight.Fighter2Name
		}

		if state.DeathOccurred {
			e.logFightAction(fight.ID, fmt.Sprintf("%s wins by DEATH!", winnerName))
		} else if state.Fighter1Health <= 0 || state.Fighter2Health <= 0 {
			e.logFightAction(fight.ID, fmt.Sprintf("%s wins by KO!", winnerName))
		} else {
			e.logFightAction(fight.ID, fmt.Sprintf("%s wins by decision!", winnerName))
		}
	} else {
		e.logFightAction(fight.ID, "Fight ends in a draw!")
	}

	// Close the fight log
	defer e.closeFightLog(fight.ID)

	// Determine winner ID for betting purposes. Orientation swap doesn't change
	// winner ID because WinnerID is compared against fight.Fighter1ID/2ID later.
	var winnerIDPtr *int
	if state.WinnerID != 0 {
		winnerIDPtr = &state.WinnerID
	}

	// Get user IDs who have bets on this fight (for role updates after credit changes)
	affectedUserIDs, err := e.repo.GetUserIDsWithBetsOnFight(fight.ID)
	if err != nil {
		log.Printf("Failed to get user IDs for role updates: %v", err)
		affectedUserIDs = nil // Continue without role updates
	}

	// Process all bets for this fight BEFORE updating the fight status
	err = e.repo.ProcessBetsForFight(fight.ID, winnerIDPtr)
	if err != nil {
		log.Printf("Failed to process bets for fight %d: %v", fight.ID, err)
		// Continue with fight completion even if bet processing fails
	} else if len(affectedUserIDs) > 0 {
		// Update Discord roles for users whose credits changed
		go e.UpdateUserRolesAfterCreditsChange(affectedUserIDs)
	}

	// Process MVP rewards if there's a winner
	if state.WinnerID != 0 {
		err = e.repo.ProcessMVPRewards(fight.ID, state.WinnerID)
		if err != nil {
			log.Printf("Failed to process MVP rewards for fight %d: %v", fight.ID, err)
			// Continue with fight completion even if MVP processing fails
		}

		err = e.applyLegacyInfusionIfSaturdayChampion(fight, state, time.Now())
		if err != nil {
			log.Printf("Failed to apply legacy infusion for fight %d: %v", fight.ID, err)
		}
	}

	// Update fight in database using lane healths directly
	err = e.repo.UpdateFightResult(fight.ID, nullableInt64(state.WinnerID), state.Fighter1Health, state.Fighter2Health)
	if err != nil {
		return fmt.Errorf("failed to update fight: %w", err)
	}

	// Handle death if it occurred
	if state.DeathOccurred {
		var deadFighterID int
		if state.WinnerID == fight.Fighter1ID {
			deadFighterID = fight.Fighter2ID
		} else {
			deadFighterID = fight.Fighter1ID
		}

		err = e.repo.KillFighter(deadFighterID)
		if err != nil {
			return fmt.Errorf("failed to kill fighter: %w", err)
		}

		log.Printf("Fighter %d has died!", deadFighterID)
	}

	// Update fighter records
	switch state.WinnerID {
	case fight.Fighter1ID:
		err = e.repo.UpdateFighterRecords(fight.Fighter1ID, fight.Fighter2ID, "fighter1_wins")
	case fight.Fighter2ID:
		err = e.repo.UpdateFighterRecords(fight.Fighter1ID, fight.Fighter2ID, "fighter2_wins")
	default:
		err = e.repo.UpdateFighterRecords(fight.Fighter1ID, fight.Fighter2ID, "draw")
	}

	if err != nil {
		return fmt.Errorf("failed to update fighter records: %w", err)
	}

	// Get fighter information for Discord notification
	fighter1, err := e.repo.GetFighter(fight.Fighter1ID)
	if err != nil {
		log.Printf("Failed to get fighter1 for Discord notification: %v", err)
		fighter1 = nil
	}

	fighter2, err := e.repo.GetFighter(fight.Fighter2ID)
	if err != nil {
		log.Printf("Failed to get fighter2 for Discord notification: %v", err)
		fighter2 = nil
	}

	// Send Discord notification (don't fail the fight if this fails)
	if fighter1 != nil && fighter2 != nil {
		// Convert fight.FightState to discord.FightState to avoid import cycles
		discordState := &discord.FightState{
			Fighter1Health: state.Fighter1Health,
			Fighter2Health: state.Fighter2Health,
			TickNumber:     state.TickNumber,
			LastDamage1:    state.LastDamage1,
			LastDamage2:    state.LastDamage2,
			CurrentRound:   state.CurrentRound,
			IsComplete:     state.IsComplete,
			WinnerID:       state.WinnerID,
			DeathOccurred:  state.DeathOccurred,
		}

		err = e.discordNotifier.NotifyFightResult(fight, discordState, *fighter1, *fighter2)
		if err != nil {
			log.Printf("Failed to send Discord notification for fight %d: %v", fight.ID, err)
			// Continue - don't fail fight completion if Discord notification fails
		}

		// Removed Action channel settlement summary

		// Discord events removed
	}

	// Wiki sync (non-blocking best-effort): upsert rich fight + fighter pages
	go func() {
		client, err := wiki.New()
		if err != nil {
			log.Printf("wiki: %v", err)
			return
		}
		// Upsert fight page with avatars, stats, and final health
		if fighter1 != nil && fighter2 != nil {
			tournamentName := ""
			if t, err := e.repo.GetTournament(fight.TournamentID); err == nil && t != nil {
				tournamentName = t.Name
			}
			richFight := fight
			// Reuse DB final scores already saved above
			_ = client.UpsertFightPage(richFight, *fighter1, *fighter2, tournamentName)
		}

		// Also upsert individual fighter pages to ensure max info stays current
		if fighter1 != nil {
			_ = client.UpsertFighterPage(*fighter1)
		}
		if fighter2 != nil {
			_ = client.UpsertFighterPage(*fighter2)
		}
	}()

	log.Printf("Fight completed successfully, bets and MVP rewards processed")
	return nil
}

func (e *Engine) applyLegacyInfusionIfSaturdayChampion(fight database.Fight, state *FightState, now time.Time) error {
	if state.WinnerID == 0 {
		return nil
	}

	centralTime, err := time.LoadLocation("America/Chicago")
	if err != nil {
		return err
	}

	nowC := now.In(centralTime)
	scheduledC := fight.ScheduledTime.In(centralTime)

	if scheduledC.Weekday() != time.Saturday {
		return nil
	}

	if scheduledC.Hour() < 23 || (scheduledC.Hour() == 23 && scheduledC.Minute() < 30) {
		return nil
	}

	isInWindow := false
	if nowC.Weekday() == time.Saturday {
		if nowC.Hour() > 23 || (nowC.Hour() == 23 && nowC.Minute() >= 30) {
			isInWindow = true
		}
	} else if nowC.Weekday() == time.Sunday {
		if nowC.Hour() < 1 {
			isInWindow = true
		}
	}

	if !isInWindow {
		return nil
	}

	stats := []string{"strength", "speed", "endurance", "technique"}
	seed := now.UnixNano() ^ int64(fight.ID)
	rng := utils.NewSeededRNG(seed)
	chosenStat := stats[rng.Intn(len(stats))]

	if err := e.repo.IncrementFighterCombatStat(state.WinnerID, chosenStat, 1); err != nil {
		return err
	}

	rec := database.ChampionLegacyRecord{
		FightID:      fight.ID,
		FighterID:    state.WinnerID,
		TournamentID: fight.TournamentID,
		StatAwarded:  chosenStat,
		StatDelta:    1,
		AwardedAt:    nowC,
	}

	if totalWagered, totalPayout, err := e.repo.SumChampionFightBets(fight.ID); err == nil {
		rec.TotalWagered = totalWagered
		rec.TotalPayout = totalPayout
	} else {
		log.Printf("Legacy infusion: failed to aggregate bets for fight %d: %v", fight.ID, err)
	}

	if blessings, curses, err := e.repo.CountEffectsForFightDay(fight.ID); err == nil {
		rec.BlessingsCount = blessings
		rec.CursesCount = curses
	} else {
		log.Printf("Legacy infusion: failed to aggregate effects for fight %d: %v", fight.ID, err)
	}

	if tournament, err := e.repo.GetTournament(fight.TournamentID); err == nil {
		rec.TournamentWeek = tournament.WeekNumber
		rec.TournamentName = tournament.Name
	}

	if err := e.repo.CreateChampionLegacyRecord(rec); err != nil {
		return err
	}

	if fighter, err := e.repo.GetFighter(state.WinnerID); err == nil {
		log.Printf("Legacy infusion: %s gains +1 %s after Saturday championship", fighter.Name, chosenStat)
	} else {
		log.Printf("Legacy infusion applied to fighter %d (+1 %s)", state.WinnerID, chosenStat)
	}

	return nil
}

// ProcessActiveFights handles all currently active fights
func (e *Engine) ProcessActiveFights(now time.Time) error {
	// Get all active fights
	activeFights, err := e.repo.GetActiveFights()
	if err != nil {
		return fmt.Errorf("failed to get active fights: %w", err)
	}

	log.Printf("Processing %d active fights", len(activeFights))

	for _, fight := range activeFights {
		err = e.processSingleActiveFight(fight, now)
		if err != nil {
			log.Printf("Error processing fight %d: %v", fight.ID, err)
			continue
		}
	}

	return nil
}

// processSingleActiveFight handles one active fight
func (e *Engine) processSingleActiveFight(fight database.Fight, now time.Time) error {
	// Get fighters
	fighter1, err := e.repo.GetFighter(fight.Fighter1ID)
	if err != nil {
		return fmt.Errorf("failed to get fighter1: %w", err)
	}

	fighter2, err := e.repo.GetFighter(fight.Fighter2ID)
	if err != nil {
		return fmt.Errorf("failed to get fighter2: %w", err)
	}

	// Check if fight should be over (30 minutes elapsed)
	fightEndTime := fight.ScheduledTime.Add(30 * time.Minute)
	if now.After(fightEndTime) {
		// Simulate complete fight to get final result
		state, err := e.SimulateFightFromStart(fight, *fighter1, *fighter2)
		if err != nil {
			return fmt.Errorf("failed to simulate complete fight: %w", err)
		}

		return e.CompleteFight(fight, state)
	}

	// Start live simulation for this active fight
	log.Printf("Starting live simulation for active fight %d", fight.ID)
	return e.StartLiveFightSimulation(fight, *fighter1, *fighter2)
}

// nullableInt64 converts int to sql.NullInt64 (0 becomes NULL)
func nullableInt64(value int) interface{} {
	if value == 0 {
		return nil
	}
	return value
}

// UpdateUserRolesAfterCreditsChange updates Discord roles for users whose credits changed
func (e *Engine) UpdateUserRolesAfterCreditsChange(userIDs []int) {
	if e.roleManager == nil {
		return
	}

	for _, userID := range userIDs {
		user, err := e.repo.GetUser(userID)
		if err != nil {
			log.Printf("Failed to get user %d for role update: %v", userID, err)
			continue
		}

		err = e.roleManager.UpdateUserRole(user)
		if err != nil {
			log.Printf("Failed to update Discord role for user %s: %v", user.Username, err)
		}
	}
}

// LogAction is a public method to log fight actions (used by websocket for clap summaries)
func (e *Engine) LogAction(fightID int, text string) {
	e.logFightAction(fightID, text)
}
