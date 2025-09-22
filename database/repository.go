package database

import (
	"fmt"
	"log"
	"time"

	"database/sql"

	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetTournamentByWeek(weekNumber int) (*Tournament, error) {
	var tournament Tournament
	err := r.db.Get(&tournament, "SELECT * FROM tournaments WHERE week_number = ?", weekNumber)
	return &tournament, err
}

func (r *Repository) GetTournament(tournamentID int) (*Tournament, error) {
	var tournament Tournament
	err := r.db.Get(&tournament, "SELECT * FROM tournaments WHERE id = ?", tournamentID)
	return &tournament, err
}

func (r *Repository) GetAliveFighters() ([]Fighter, error) {
	var fighters []Fighter
	err := r.db.Select(&fighters, "SELECT * FROM fighters WHERE is_dead = FALSE ORDER BY id")
	return fighters, err
}

func (r *Repository) GetTodaysFights(tournamentID int, today, tomorrow time.Time) ([]Fight, error) {
	var fights []Fight
	err := r.db.Select(&fights,
		"SELECT * FROM fights WHERE tournament_id = ? AND scheduled_time >= ? AND scheduled_time < ? ORDER BY scheduled_time",
		tournamentID, today, tomorrow)
	return fights, err
}

func (r *Repository) InsertFight(fight Fight) error {
	_, err := r.db.NamedExec(`
		INSERT INTO fights (tournament_id, fighter1_id, fighter2_id, fighter1_name, fighter2_name, scheduled_time, status, created_at)
		VALUES (:tournament_id, :fighter1_id, :fighter2_id, :fighter1_name, :fighter2_name, :scheduled_time, :status, datetime('now'))
	`, fight)
	return err
}

func (r *Repository) ActivateCurrentFights(tournamentID int, now time.Time) error {
	_, err := r.db.Exec(`
		UPDATE fights 
		SET status = 'active' 
		WHERE tournament_id = ? 
		AND scheduled_time <= ? 
		AND datetime(scheduled_time, '+30 minutes') > ?
		AND status = 'scheduled'`,
		tournamentID, now, now)
	return err
}

func (r *Repository) GetExpiredScheduledFights(tournamentID int, now time.Time) ([]Fight, error) {
	var fights []Fight
	err := r.db.Select(&fights,
		"SELECT * FROM fights WHERE tournament_id = ? AND datetime(scheduled_time, '+30 minutes') < ? AND status = 'scheduled'",
		tournamentID, now)
	return fights, err
}

func (r *Repository) VoidFight(fightID int, reason string) error {
	_, err := r.db.Exec(`
		UPDATE fights 
		SET status = 'voided', voided_reason = ?, completed_at = datetime('now')
		WHERE id = ?`, reason, fightID)
	return err
}

func (r *Repository) UpdateFighterRecords(fighter1ID, fighter2ID int, result string) error {
	switch result {
	case "draw":
		_, err := r.db.Exec("UPDATE fighters SET draws = draws + 1 WHERE id = ? OR id = ?",
			fighter1ID, fighter2ID)
		return err
	case "fighter1_wins":
		tx, err := r.db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		_, err = tx.Exec("UPDATE fighters SET wins = wins + 1 WHERE id = ?", fighter1ID)
		if err != nil {
			return err
		}
		_, err = tx.Exec("UPDATE fighters SET losses = losses + 1 WHERE id = ?", fighter2ID)
		if err != nil {
			return err
		}
		return tx.Commit()
	case "fighter2_wins":
		tx, err := r.db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		_, err = tx.Exec("UPDATE fighters SET wins = wins + 1 WHERE id = ?", fighter2ID)
		if err != nil {
			return err
		}
		_, err = tx.Exec("UPDATE fighters SET losses = losses + 1 WHERE id = ?", fighter1ID)
		if err != nil {
			return err
		}
		return tx.Commit()
	default:
		return fmt.Errorf("invalid result: %s", result)
	}
}

func (r *Repository) KillFighter(fighterID int) error {
	_, err := r.db.Exec("UPDATE fighters SET is_dead = TRUE WHERE id = ?", fighterID)
	return err
}

func (r *Repository) GetActiveFights() ([]Fight, error) {
	var fights []Fight
	err := r.db.Select(&fights, "SELECT * FROM fights WHERE status = 'active'")
	return fights, err
}

func (r *Repository) GetFighter(fighterID int) (*Fighter, error) {
	var fighter Fighter
	err := r.db.Get(&fighter, "SELECT * FROM fighters WHERE id = ?", fighterID)
	return &fighter, err
}

func (r *Repository) GetFight(fightID int) (*Fight, error) {
	var fight Fight
	err := r.db.Get(&fight, "SELECT * FROM fights WHERE id = ?", fightID)
	return &fight, err
}

func (r *Repository) UpdateFightResult(fightID int, winnerID interface{}, score1, score2 int) error {
	_, err := r.db.Exec(`
		UPDATE fights 
		SET status = 'completed', 
			winner_id = ?, 
			final_score1 = ?, 
			final_score2 = ?, 
			completed_at = datetime('now')
		WHERE id = ?`,
		winnerID, score1, score2, fightID)
	return err
}

// Session management methods
func (r *Repository) CreateSession(token string, userID int, expiresAt time.Time) error {
	_, err := r.db.Exec(`
		INSERT INTO sessions (token, user_id, expires_at) 
		VALUES (?, ?, ?)`,
		token, userID, expiresAt)
	return err
}

func (r *Repository) GetUserBySessionToken(token string) (*User, error) {
	var user User
	err := r.db.Get(&user, `
		SELECT u.* FROM users u 
		JOIN sessions s ON u.id = s.user_id 
		WHERE s.token = ? AND s.expires_at > datetime('now')`,
		token)
	return &user, err
}

func (r *Repository) DeleteSession(token string) error {
	_, err := r.db.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

func (r *Repository) CleanExpiredSessions() error {
	_, err := r.db.Exec("DELETE FROM sessions WHERE expires_at <= datetime('now')")
	return err
}

// User management methods
func (r *Repository) CreateUser(discordID, username, avatarURL string) (*User, error) {
	result, err := r.db.Exec(`
		INSERT INTO users (discord_id, username, avatar_url, custom_username, credits) 
		VALUES (?, ?, ?, ?, 1000)`,
		discordID, username, avatarURL, username)
	if err != nil {
		return nil, err
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	var user User
	err = r.db.Get(&user, "SELECT * FROM users WHERE id = ?", userID)
	return &user, err
}

func (r *Repository) GetUserByDiscordID(discordID string) (*User, error) {
	var user User
	err := r.db.Get(&user, "SELECT * FROM users WHERE discord_id = ?", discordID)
	return &user, err
}

func (r *Repository) GetUser(userID int) (*User, error) {
	var user User
	err := r.db.Get(&user, "SELECT * FROM users WHERE id = ?", userID)
	return &user, err
}

func (r *Repository) GetUserByUsername(username string) (*User, error) {
	var user User
	err := r.db.Get(&user, "SELECT * FROM users WHERE username = ? OR custom_username = ?", username, username)
	return &user, err
}

func (r *Repository) UpdateUserCustomUsername(userID int, customUsername string) error {
	_, err := r.db.Exec("UPDATE users SET custom_username = ?, updated_at = datetime('now') WHERE id = ?",
		customUsername, userID)
	return err
}

func (r *Repository) GetAllUsersByCredits() ([]User, error) {
	var users []User
	err := r.db.Select(&users, "SELECT * FROM users ORDER BY credits DESC, created_at ASC")
	return users, err
}

func (r *Repository) GetAllFightersByRecord() ([]Fighter, error) {
	var fighters []Fighter
	err := r.db.Select(&fighters, "SELECT * FROM fighters ORDER BY wins DESC, losses ASC, name ASC")
	return fighters, err
}

// Betting methods
func (r *Repository) CreateBet(userID, fightID, fighterID, amount int) error {
	_, err := r.db.Exec(`
		INSERT INTO bets (user_id, fight_id, fighter_id, amount, status, created_at) 
		VALUES (?, ?, ?, ?, 'pending', datetime('now'))`,
		userID, fightID, fighterID, amount)
	return err
}

func (r *Repository) GetUserBetOnFight(userID, fightID int) (*Bet, error) {
	var bet Bet
	err := r.db.Get(&bet, "SELECT * FROM bets WHERE user_id = ? AND fight_id = ?", userID, fightID)
	return &bet, err
}

func (r *Repository) GetAllBetsOnFight(fightID int) ([]BetWithUser, error) {
	var bets []BetWithUser
	err := r.db.Select(&bets, `
		SELECT b.*, u.username, u.custom_username, f.name as fighter_name
		FROM bets b 
		JOIN users u ON b.user_id = u.id 
		JOIN fighters f ON b.fighter_id = f.id
		WHERE b.fight_id = ? 
		ORDER BY b.created_at ASC`,
		fightID)
	return bets, err
}

func (r *Repository) UpdateUserCredits(userID, credits int) error {
	_, err := r.db.Exec("UPDATE users SET credits = ?, updated_at = datetime('now') WHERE id = ?",
		credits, userID)
	return err
}

func (r *Repository) ProcessBetsForFight(fightID int, winnerID *int) error {
	// Get all bets for this fight
	var bets []Bet
	err := r.db.Select(&bets, "SELECT * FROM bets WHERE fight_id = ? AND status = 'pending'", fightID)
	if err != nil {
		return err
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, bet := range bets {
		var newStatus string
		var payout int

		if winnerID == nil {
			// Draw - return original bet
			newStatus = "voided"
			payout = bet.Amount
		} else if bet.FighterID == *winnerID {
			// Won - double the bet (1:1 odds)
			newStatus = "won"
			payout = bet.Amount * 2
		} else {
			// Lost - no payout
			newStatus = "lost"
			payout = 0
		}

		// Update bet status
		_, err = tx.Exec(`
			UPDATE bets 
			SET status = ?, payout = ?, resolved_at = datetime('now') 
			WHERE id = ?`,
			newStatus, payout, bet.ID)
		if err != nil {
			return err
		}

		// Update user credits if there's a payout
		if payout > 0 {
			_, err = tx.Exec(`
				UPDATE users 
				SET credits = credits + ?, updated_at = datetime('now') 
				WHERE id = ?`,
				payout, bet.UserID)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// ConsolidateUserInventory removes duplicate inventory items and consolidates quantities
func (r *Repository) ConsolidateUserInventory() error {
	// Get all users with duplicate inventory items
	rows, err := r.db.Query(`
		SELECT user_id, shop_item_id, SUM(quantity) as total_quantity, MIN(id) as oldest_id
		FROM user_inventory 
		GROUP BY user_id, shop_item_id 
		HAVING COUNT(*) > 1`)
	if err != nil {
		return err
	}
	defer rows.Close()

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	consolidationCount := 0
	for rows.Next() {
		var userID, shopItemID, totalQuantity, oldestID int
		err := rows.Scan(&userID, &shopItemID, &totalQuantity, &oldestID)
		if err != nil {
			return err
		}

		// Update the oldest entry with the total quantity
		_, err = tx.Exec(`
			UPDATE user_inventory 
			SET quantity = ? 
			WHERE id = ?`,
			totalQuantity, oldestID)
		if err != nil {
			return err
		}

		// Delete duplicate entries
		_, err = tx.Exec(`
			DELETE FROM user_inventory 
			WHERE user_id = ? AND shop_item_id = ? AND id != ?`,
			userID, shopItemID, oldestID)
		if err != nil {
			return err
		}

		consolidationCount++
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	if consolidationCount > 0 {
		log.Printf("Consolidated %d duplicate inventory entries", consolidationCount)
	}

	return nil
}

func (r *Repository) UpdateUser(user *User) error {
	_, err := r.db.Exec(`
		UPDATE users 
		SET username = ?, avatar_url = ?, updated_at = datetime('now') 
		WHERE id = ?`,
		user.Username, user.AvatarURL, user.ID)
	return err
}

func (r *Repository) GetUserBets(userID int) ([]BetWithFight, error) {
	var bets []BetWithFight
	err := r.db.Select(&bets, `
		SELECT b.id, b.user_id, b.fight_id, b.fighter_id, b.amount, b.status, b.payout, 
		       b.created_at, b.resolved_at,
		       f.fighter1_name, f.fighter2_name, f.scheduled_time, f.status as fight_status,
		       fighter.name as fighter_name
		FROM bets b 
		JOIN fights f ON b.fight_id = f.id 
		JOIN fighters fighter ON b.fighter_id = fighter.id
		WHERE b.user_id = ? 
		ORDER BY b.created_at DESC`,
		userID)
	return bets, err
}

// GetUserBettingStats returns comprehensive betting statistics for a user
func (r *Repository) GetUserBettingStats(userID int) (*BettingStats, error) {
	var stats BettingStats

	// Get basic bet counts and totals
	err := r.db.Get(&stats, `
		SELECT 
			COUNT(*) as total_bets,
			COUNT(CASE WHEN status = 'won' THEN 1 END) as bets_won,
			COUNT(CASE WHEN status = 'lost' THEN 1 END) as bets_lost,
			COUNT(CASE WHEN status = 'voided' THEN 1 END) as bets_voided,
			COUNT(CASE WHEN status = 'pending' THEN 1 END) as active_bets,
			COALESCE(SUM(CASE WHEN status = 'won' THEN payout - amount ELSE 0 END), 0) as total_winnings,
			COALESCE(SUM(CASE WHEN status = 'lost' THEN amount ELSE 0 END), 0) as total_losses,
			COALESCE(AVG(amount), 0) as avg_bet_size,
			COALESCE(MAX(CASE WHEN status = 'won' THEN payout - amount END), 0) as biggest_win,
			COALESCE(MAX(CASE WHEN status = 'lost' THEN amount END), 0) as biggest_loss
		FROM bets 
		WHERE user_id = ?`, userID)

	if err != nil {
		return nil, err
	}

	// Calculate derived stats
	if stats.BetsWon+stats.BetsLost > 0 {
		stats.WinRate = float64(stats.BetsWon) / float64(stats.BetsWon+stats.BetsLost) * 100
	}

	if stats.BetsLost > 0 {
		stats.WinLossRatio = float64(stats.BetsWon) / float64(stats.BetsLost)
	} else if stats.BetsWon > 0 {
		stats.WinLossRatio = float64(stats.BetsWon) // Perfect record
	}

	stats.NetProfit = stats.TotalWinnings - stats.TotalLosses

	return &stats, nil
}

// Shop methods
func (r *Repository) GetAllShopItems() ([]ShopItem, error) {
	var items []ShopItem
	err := r.db.Select(&items, "SELECT * FROM shop_items ORDER BY price ASC, name ASC")
	return items, err
}

func (r *Repository) GetShopItem(itemID int) (*ShopItem, error) {
	var item ShopItem
	err := r.db.Get(&item, "SELECT * FROM shop_items WHERE id = ?", itemID)
	return &item, err
}

func (r *Repository) PurchaseItem(userID, itemID, quantity int, totalCost int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Deduct credits from user
	_, err = tx.Exec(`
		UPDATE users 
		SET credits = credits - ?, updated_at = datetime('now') 
		WHERE id = ? AND credits >= ?`,
		totalCost, userID, totalCost)
	if err != nil {
		return err
	}

	// Check if user already has this item (using SUM to handle multiple rows)
	var existingQuantity int
	err = tx.QueryRow(`
		SELECT COALESCE(SUM(quantity), 0) FROM user_inventory 
		WHERE user_id = ? AND shop_item_id = ?`,
		userID, itemID).Scan(&existingQuantity)

	if err != nil {
		return err
	}

	if existingQuantity > 0 {
		// User has this item - consolidate all existing rows into one
		// First, get the oldest inventory entry ID for this item
		var oldestID int
		err = tx.QueryRow(`
			SELECT id FROM user_inventory 
			WHERE user_id = ? AND shop_item_id = ? 
			ORDER BY created_at ASC 
			LIMIT 1`,
			userID, itemID).Scan(&oldestID)
		if err != nil {
			return err
		}

		// Update the oldest entry with the total quantity
		_, err = tx.Exec(`
			UPDATE user_inventory 
			SET quantity = ? 
			WHERE id = ?`,
			existingQuantity+quantity, oldestID)
		if err != nil {
			return err
		}

		// Delete any other duplicate entries for this item
		_, err = tx.Exec(`
			DELETE FROM user_inventory 
			WHERE user_id = ? AND shop_item_id = ? AND id != ?`,
			userID, itemID, oldestID)
		if err != nil {
			return err
		}
	} else {
		// Insert new inventory item
		_, err = tx.Exec(`
			INSERT INTO user_inventory (user_id, shop_item_id, quantity, created_at) 
			VALUES (?, ?, ?, datetime('now'))`,
			userID, itemID, quantity)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *Repository) GetUserInventory(userID int) ([]UserInventoryItem, error) {
	var items []UserInventoryItem
	err := r.db.Select(&items, `
		SELECT ui.id, ui.user_id, ui.shop_item_id, ui.quantity, ui.created_at,
		       si.name, si.description, si.emoji, si.item_type, si.effect_value
		FROM user_inventory ui 
		JOIN shop_items si ON ui.shop_item_id = si.id
		WHERE ui.user_id = ? AND ui.quantity > 0
		ORDER BY si.item_type, si.name`,
		userID)
	return items, err
}

func (r *Repository) UseInventoryItem(userID, itemID int, quantity int) error {
	_, err := r.db.Exec(`
		UPDATE user_inventory 
		SET quantity = quantity - ? 
		WHERE user_id = ? AND shop_item_id = ? AND quantity >= ?`,
		quantity, userID, itemID, quantity)
	return err
}

func (r *Repository) ApplyEffect(userID int, targetType string, targetID int, effectType string, effectValue int) error {
	// Store current time as UTC for consistent storage
	now := time.Now().UTC()
	timestampStr := now.Format("2006-01-02 15:04:05")

	// For fighter effects, randomly select which stat to modify
	var finalEffectType string
	if effectType == "fighter_blessing" || effectType == "fighter_curse" {
		stats := []string{"strength", "speed", "endurance", "technique"}
		randomStat := stats[now.UnixNano()%4] // Simple random selection using timestamp

		if effectType == "fighter_blessing" {
			finalEffectType = randomStat + "_blessing"
		} else {
			finalEffectType = randomStat + "_curse"
		}
	} else {
		finalEffectType = effectType
	}

	_, err := r.db.Exec(`
		INSERT INTO applied_effects (user_id, target_type, target_id, effect_type, effect_value, created_at) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		userID, targetType, targetID, finalEffectType, effectValue, timestampStr)
	return err
}

func (r *Repository) GetAppliedEffects(targetType string, targetID int) ([]AppliedEffect, error) {
	var effects []AppliedEffect
	err := r.db.Select(&effects, `
		SELECT * FROM applied_effects 
		WHERE target_type = ? AND target_id = ? 
		ORDER BY created_at DESC`,
		targetType, targetID)
	return effects, err
}

// GetAppliedEffectsForDate gets applied effects for a target within a specific date range
func (r *Repository) GetAppliedEffectsForDate(targetType string, targetID int, startDate, endDate time.Time) ([]AppliedEffect, error) {
	var effects []AppliedEffect

	// Convert to Central Time and format as strings to match SQLite's storage format
	centralTime, _ := time.LoadLocation("America/Chicago")
	startCentral := startDate.In(centralTime)
	endCentral := endDate.In(centralTime)

	startStr := startCentral.Format("2006-01-02 15:04:05")
	endStr := endCentral.Format("2006-01-02 15:04:05")

	err := r.db.Select(&effects, `
		SELECT * FROM applied_effects 
		WHERE target_type = ? AND target_id = ? 
		AND created_at >= ? AND created_at < ?
		ORDER BY created_at DESC`,
		targetType, targetID, startStr, endStr)
	return effects, err
}

// GetAppliedEffectsByUserForFight gets effects applied by users for a specific fight with user info
func (r *Repository) GetAppliedEffectsByUserForFight(fightID int) ([]AppliedEffectWithUser, error) {
	var effects []AppliedEffectWithUser
	err := r.db.Select(&effects, `
		SELECT ae.*, u.username, u.custom_username, f.name as target_name
		FROM applied_effects ae
		JOIN users u ON ae.user_id = u.id
		JOIN fighters f ON ae.target_id = f.id
		WHERE ae.target_type = 'fighter' 
		AND (ae.target_id = (SELECT fighter1_id FROM fights WHERE id = ?) 
		     OR ae.target_id = (SELECT fighter2_id FROM fights WHERE id = ?))
		ORDER BY ae.created_at DESC`,
		fightID, fightID)
	return effects, err
}

// User Settings methods
func (r *Repository) GetUserSetting(userID int, settingType string) (*UserSetting, error) {
	var setting UserSetting
	err := r.db.Get(&setting, `
		SELECT * FROM user_settings 
		WHERE user_id = ? AND setting_type = ?`,
		userID, settingType)
	return &setting, err
}

func (r *Repository) SetUserSetting(userID int, settingType, settingValue string, canChangeAt *time.Time) error {
	var canChangeAtValue interface{}
	if canChangeAt != nil {
		canChangeAtValue = *canChangeAt
	}

	_, err := r.db.Exec(`
		INSERT INTO user_settings (user_id, setting_type, setting_value, can_change_at, updated_at) 
		VALUES (?, ?, ?, ?, datetime('now'))
		ON CONFLICT(user_id, setting_type) DO UPDATE SET
			setting_value = excluded.setting_value,
			can_change_at = excluded.can_change_at,
			updated_at = datetime('now')`,
		userID, settingType, settingValue, canChangeAtValue)
	return err
}

func (r *Repository) CanChangeUserSetting(userID int, settingType string) (bool, error) {
	var setting UserSetting
	err := r.db.Get(&setting, `
		SELECT * FROM user_settings 
		WHERE user_id = ? AND setting_type = ?`,
		userID, settingType)
	if err != nil {
		// If no setting exists, they can set it
		return true, nil
	}

	// If no restriction, they can change
	if !setting.CanChangeAt.Valid {
		return true, nil
	}

	// Check if enough time has passed
	return time.Now().After(setting.CanChangeAt.Time), nil
}

func (r *Repository) PayToChangeUserSetting(userID int, settingType, newValue string, cost int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Deduct credits
	_, err = tx.Exec(`
		UPDATE users 
		SET credits = credits - ?, updated_at = datetime('now') 
		WHERE id = ? AND credits >= ?`,
		cost, userID, cost)
	if err != nil {
		return err
	}

	// Update setting with no restriction
	_, err = tx.Exec(`
		INSERT INTO user_settings (user_id, setting_type, setting_value, can_change_at, updated_at) 
		VALUES (?, ?, ?, NULL, datetime('now'))
		ON CONFLICT(user_id, setting_type) DO UPDATE SET
			setting_value = excluded.setting_value,
			can_change_at = NULL,
			updated_at = datetime('now')`,
		userID, settingType, newValue)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// MVP reward processing
func (r *Repository) ProcessMVPRewards(fightID int, winnerID int) error {
	// Get all users who have this fighter as their MVP AND own the MVP Player item
	mvpUsers, err := r.db.Query(`
		SELECT DISTINCT us.user_id 
		FROM user_settings us
		WHERE us.setting_type = 'mvp_player' 
		AND us.setting_value = ?
		AND EXISTS (
			SELECT 1 FROM user_inventory ui 
			JOIN shop_items si ON ui.shop_item_id = si.id 
			WHERE ui.user_id = us.user_id 
			AND si.item_type = 'mvp_player' 
			AND ui.quantity > 0
		)`,
		fmt.Sprintf("%d", winnerID))
	if err != nil {
		log.Printf("Error querying MVP users for fighter %d: %v", winnerID, err)
		return err
	}
	defer mvpUsers.Close()

	rewardCount := 0
	// Award credits to each MVP holder
	for mvpUsers.Next() {
		var userID int
		err := mvpUsers.Scan(&userID)
		if err != nil {
			log.Printf("Error scanning MVP user: %v", err)
			continue
		}

		// Award 10,000 credits
		_, err = r.db.Exec(`
			UPDATE users 
			SET credits = credits + 10000, updated_at = datetime('now') 
			WHERE id = ?`,
			userID)
		if err != nil {
			log.Printf("Failed to award MVP credits to user %d: %v", userID, err)
		} else {
			log.Printf("Awarded 10,000 MVP credits to user %d for fighter %d winning", userID, winnerID)
			rewardCount++
		}
	}

	log.Printf("MVP rewards processed: %d users rewarded for fighter %d winning", rewardCount, winnerID)
	return nil
}

// GetFighterByName gets a fighter by their name (for duplicate checking)
func (r *Repository) GetFighterByName(name string) (*Fighter, error) {
	fighter := &Fighter{}
	err := r.db.Get(fighter, "SELECT * FROM fighters WHERE name = ?", name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found, which is what we want for uniqueness checking
		}
		return nil, err
	}
	return fighter, nil
}

// CreateCustomFighter creates a new custom fighter and returns the fighter ID
func (r *Repository) CreateCustomFighter(fighter Fighter) (int, error) {
	result, err := r.db.Exec(`
		INSERT INTO fighters (
			name, team, strength, speed, endurance, technique, 
			blood_type, horoscope, molecular_density, existential_dread, 
			fingers, toes, ancestors, fighter_class, wins, losses, draws, 
			is_dead, created_by_user_id, is_custom, creation_date, 
			custom_description, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		fighter.Name, fighter.Team, fighter.Strength, fighter.Speed,
		fighter.Endurance, fighter.Technique, fighter.BloodType,
		fighter.Horoscope, fighter.MolecularDensity, fighter.ExistentialDread,
		fighter.Fingers, fighter.Toes, fighter.Ancestors, fighter.FighterClass,
		fighter.Wins, fighter.Losses, fighter.Draws, fighter.IsDead,
		fighter.CreatedByUserID, fighter.IsCustom, fighter.CreationDate,
		fighter.CustomDescription, time.Now(),
	)
	if err != nil {
		return 0, err
	}

	fighterID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(fighterID), nil
}

// GetUserIDsWithBetsOnFight returns user IDs of users who have bets on the given fight
func (r *Repository) GetUserIDsWithBetsOnFight(fightID int) ([]int, error) {
	var userIDs []int
	err := r.db.Select(&userIDs, "SELECT DISTINCT user_id FROM bets WHERE fight_id = ? AND status = 'pending'", fightID)
	return userIDs, err
}

// GetHighRollerUserIDs returns user IDs that own the High Roller Card
func (r *Repository) GetHighRollerUserIDs() ([]int, error) {
	var userIDs []int
	err := r.db.Select(&userIDs, `
        SELECT DISTINCT ui.user_id
        FROM user_inventory ui
        JOIN shop_items si ON ui.shop_item_id = si.id
        WHERE si.item_type = 'high_roller' AND ui.quantity > 0`)
	return userIDs, err
}

// TaxHighRollersIfNeeded applies a weekly 7.5% tithe on Mondays to users with High Roller Card.
// Idempotent per week per user using user_settings (setting_type='high_roller_tax_week').
func (r *Repository) TaxHighRollersIfNeeded(now time.Time) error {
	// Only run on Mondays to limit load (but still idempotent)
	if now.Weekday() != time.Monday {
		return nil
	}

	// Use ISO week for stability
	year, week := now.ISOWeek()
	weekKey := fmt.Sprintf("%04d-%02d", year, week)

	userIDs, err := r.GetHighRollerUserIDs()
	if err != nil {
		return err
	}

	for _, uid := range userIDs {
		// Check if already taxed this week
		taxed, err := r.alreadyTaxedThisWeek(uid, weekKey)
		if err != nil {
			log.Printf("Tax check error for user %d: %v", uid, err)
			continue
		}
		if taxed {
			continue
		}

		// Get current credits
		user, err := r.GetUser(uid)
		if err != nil {
			log.Printf("Failed to load user %d for tax: %v", uid, err)
			continue
		}

		if user.Credits <= 0 {
			_ = r.SetUserSetting(uid, "high_roller_tax_week", weekKey, nil)
			continue
		}

		// Deduct 7.5%
		tax := (user.Credits * 75) / 1000
		newCredits := user.Credits - tax
		if newCredits < 0 {
			newCredits = 0
		}

		err = r.UpdateUserCredits(uid, newCredits)
		if err != nil {
			log.Printf("Failed to apply high-roller tax to user %d: %v", uid, err)
			continue
		}

		// Mark as taxed for this week
		_ = r.SetUserSetting(uid, "high_roller_tax_week", weekKey, nil)
		log.Printf("Applied weekly high-roller tithe to user %d: -%d credits", uid, tax)
	}

	return nil
}

func (r *Repository) alreadyTaxedThisWeek(userID int, weekKey string) (bool, error) {
	var setting UserSetting
	err := r.db.Get(&setting, `SELECT * FROM user_settings WHERE user_id = ? AND setting_type = 'high_roller_tax_week'`, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return setting.SettingValue == weekKey, nil
}

// DecaySacrificesIfNeeded reduces each user's total 'sacrifice' item quantity weekly by 10% (floor),
// with a minimum decay of 1 if they have at least 1. Idempotent per week using user_settings
// with setting_type='sacrifice_decay_week'. Runs only on Mondays to limit load.
func (r *Repository) DecaySacrificesIfNeeded(now time.Time) error {
	// Only run on Mondays
	if now.Weekday() != time.Monday {
		return nil
	}

	// ISO week for stability
	year, week := now.ISOWeek()
	weekKey := fmt.Sprintf("%04d-%02d", year, week)

	// Find a canonical shop_item_id for 'sacrifice'
	var sacrificeItemID int
	err := r.db.Get(&sacrificeItemID, `SELECT id FROM shop_items WHERE item_type = 'sacrifice' ORDER BY id LIMIT 1`)
	if err != nil {
		return err
	}

	// Get all users who currently hold sacrifices (sum > 0)
	type row struct {
		UserID int
		Total  int
	}
	var rows []row
	err = r.db.Select(&rows, `
        SELECT ui.user_id AS user_id, COALESCE(SUM(ui.quantity),0) AS total
        FROM user_inventory ui
        JOIN shop_items si ON ui.shop_item_id = si.id
        WHERE si.item_type = 'sacrifice' AND ui.quantity > 0
        GROUP BY ui.user_id`)
	if err != nil {
		return err
	}

	for _, rec := range rows {
		if rec.Total <= 0 {
			// nothing to decay
			_ = r.SetUserSetting(rec.UserID, "sacrifice_decay_week", weekKey, nil)
			continue
		}

		// idempotence check
		var setting UserSetting
		getErr := r.db.Get(&setting, `SELECT * FROM user_settings WHERE user_id = ? AND setting_type = 'sacrifice_decay_week'`, rec.UserID)
		if getErr == nil && setting.SettingValue == weekKey {
			continue
		}

		// Compute decay: 10% floor, minimum 1
		dec := rec.Total / 10 // floor 10%
		if dec < 1 {
			dec = 1
		}
		newTotal := rec.Total - dec
		if newTotal < 0 {
			newTotal = 0
		}

		tx, err := r.db.Begin()
		if err != nil {
			return err
		}
		// Remove all existing sacrifice rows for user
		if _, err = tx.Exec(`
            DELETE FROM user_inventory 
            WHERE user_id = ? AND shop_item_id IN (
                SELECT id FROM shop_items WHERE item_type = 'sacrifice'
            )`, rec.UserID); err != nil {
			tx.Rollback()
			return err
		}

		// Reinsert with decayed quantity if any left
		if newTotal > 0 {
			if _, err = tx.Exec(`
                INSERT INTO user_inventory (user_id, shop_item_id, quantity, created_at)
                VALUES (?, ?, ?, datetime('now'))`, rec.UserID, sacrificeItemID, newTotal); err != nil {
				tx.Rollback()
				return err
			}
		}

		// Mark as decayed this week
		if _, err = tx.Exec(`
            INSERT INTO user_settings (user_id, setting_type, setting_value, updated_at)
            VALUES (?, 'sacrifice_decay_week', ?, datetime('now'))
            ON CONFLICT(user_id, setting_type) DO UPDATE SET
                setting_value = excluded.setting_value,
                updated_at = datetime('now')
        `, rec.UserID, weekKey); err != nil {
			tx.Rollback()
			return err
		}

		if err = tx.Commit(); err != nil {
			return err
		}
		log.Printf("Applied weekly sacrifice decay to user %d: -%d (from %d to %d)", rec.UserID, dec, rec.Total, newTotal)
	}

	return nil
}
