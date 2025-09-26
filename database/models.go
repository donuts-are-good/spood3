package database

import (
	"database/sql"
	"time"
)

type User struct {
	ID             int       `db:"id"`
	DiscordID      string    `db:"discord_id"`
	Username       string    `db:"username"`
	CustomUsername string    `db:"custom_username"`
	AvatarURL      string    `db:"avatar_url"`
	Credits        int       `db:"credits"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

type Fighter struct {
	ID               int       `db:"id"`
	Name             string    `db:"name"`
	Team             string    `db:"team"`
	Strength         int       `db:"strength"`
	Speed            int       `db:"speed"`
	Endurance        int       `db:"endurance"`
	Technique        int       `db:"technique"`
	BloodType        string    `db:"blood_type"`
	Horoscope        string    `db:"horoscope"`
	MolecularDensity float64   `db:"molecular_density"`
	ExistentialDread int       `db:"existential_dread"`
	Fingers          int       `db:"fingers"`
	Toes             int       `db:"toes"`
	Ancestors        int       `db:"ancestors"`
	FighterClass     string    `db:"fighter_class"`
	Wins             int       `db:"wins"`
	Losses           int       `db:"losses"`
	Draws            int       `db:"draws"`
	IsDead           bool      `db:"is_dead"`
	CreatedAt        time.Time `db:"created_at"`
	// Custom fighter fields
	CreatedByUserID   *int       `db:"created_by_user_id"`
	IsCustom          bool       `db:"is_custom"`
	CreationDate      *time.Time `db:"creation_date"`
	CustomDescription *string    `db:"custom_description"`
	Lore              string     `db:"lore"`
}

type ChampionLegacyRecord struct {
	ID             int       `db:"id"`
	FightID        int       `db:"fight_id"`
	FighterID      int       `db:"fighter_id"`
	TournamentID   int       `db:"tournament_id"`
	TournamentWeek int       `db:"tournament_week"`
	TournamentName string    `db:"tournament_name"`
	StatAwarded    string    `db:"stat_awarded"`
	StatDelta      int       `db:"stat_delta"`
	TotalWagered   int       `db:"total_wagered"`
	TotalPayout    int       `db:"total_payout"`
	BlessingsCount int       `db:"blessings_count"`
	CursesCount    int       `db:"curses_count"`
	AwardedAt      time.Time `db:"awarded_at"`
	CreatedAt      time.Time `db:"created_at"`
}

type ChampionLegacyEntry struct {
	ChampionLegacyRecord
	FighterName   string    `db:"fighter_name"`
	Fighter1Name  string    `db:"fighter1_name"`
	Fighter2Name  string    `db:"fighter2_name"`
	ScheduledTime time.Time `db:"scheduled_time"`
}

type ChampionTitleCount struct {
	FighterID   int    `db:"fighter_id"`
	FighterName string `db:"fighter_name"`
	TitleCount  int    `db:"title_count"`
}

type Tournament struct {
	ID         int       `db:"id"`
	WeekNumber int       `db:"week_number"`
	Name       string    `db:"name"`
	Sponsor    string    `db:"sponsor"`
	StartDate  time.Time `db:"start_date"`
	CreatedAt  time.Time `db:"created_at"`
}

type Fight struct {
	ID            int            `db:"id"`
	TournamentID  int            `db:"tournament_id"`
	Fighter1ID    int            `db:"fighter1_id"`
	Fighter2ID    int            `db:"fighter2_id"`
	Fighter1Name  string         `db:"fighter1_name"`
	Fighter2Name  string         `db:"fighter2_name"`
	ScheduledTime time.Time      `db:"scheduled_time"`
	Status        string         `db:"status"`
	WinnerID      sql.NullInt64  `db:"winner_id"`
	FinalScore1   sql.NullInt64  `db:"final_score1"`
	FinalScore2   sql.NullInt64  `db:"final_score2"`
	CompletedAt   sql.NullTime   `db:"completed_at"`
	VoidedReason  sql.NullString `db:"voided_reason"`
	CreatedAt     time.Time      `db:"created_at"`
}

type Bet struct {
	ID         int           `db:"id"`
	UserID     int           `db:"user_id"`
	FightID    int           `db:"fight_id"`
	FighterID  int           `db:"fighter_id"`
	Amount     int           `db:"amount"`
	Status     string        `db:"status"`
	Payout     sql.NullInt64 `db:"payout"`
	CreatedAt  time.Time     `db:"created_at"`
	ResolvedAt sql.NullTime  `db:"resolved_at"`
}

type BetWithUser struct {
	Bet
	Username       string `db:"username"`
	CustomUsername string `db:"custom_username"`
	FighterName    string `db:"fighter_name"`
}

type BetWithFight struct {
	Bet
	Fighter1Name  string    `db:"fighter1_name"`
	Fighter2Name  string    `db:"fighter2_name"`
	ScheduledTime time.Time `db:"scheduled_time"`
	FightStatus   string    `db:"fight_status"`
	FighterName   string    `db:"fighter_name"`
}

type ShopItem struct {
	ID          int       `db:"id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	Emoji       string    `db:"emoji"`
	Price       int       `db:"price"`
	ItemType    string    `db:"item_type"`
	EffectValue int       `db:"effect_value"`
	CreatedAt   time.Time `db:"created_at"`
}

type UserInventoryItem struct {
	ID         int       `db:"id"`
	UserID     int       `db:"user_id"`
	ShopItemID int       `db:"shop_item_id"`
	Quantity   int       `db:"quantity"`
	CreatedAt  time.Time `db:"created_at"`
	// Joined fields from shop_items
	Name        string `db:"name"`
	Description string `db:"description"`
	Emoji       string `db:"emoji"`
	ItemType    string `db:"item_type"`
	EffectValue int    `db:"effect_value"`
}

type AppliedEffect struct {
	ID          int       `db:"id"`
	UserID      int       `db:"user_id"`
	TargetType  string    `db:"target_type"`
	TargetID    int       `db:"target_id"`
	EffectType  string    `db:"effect_type"`
	EffectValue int       `db:"effect_value"`
	CreatedAt   time.Time `db:"created_at"`
}

type AppliedEffectWithUser struct {
	AppliedEffect
	Username       string `db:"username"`
	CustomUsername string `db:"custom_username"`
	TargetName     string `db:"target_name"`
}

type UserSetting struct {
	ID           int          `db:"id"`
	UserID       int          `db:"user_id"`
	SettingType  string       `db:"setting_type"`
	SettingValue string       `db:"setting_value"`
	CanChangeAt  sql.NullTime `db:"can_change_at"`
	CreatedAt    time.Time    `db:"created_at"`
	UpdatedAt    time.Time    `db:"updated_at"`
}

type BettingStats struct {
	TotalBets     int     `db:"total_bets"`
	BetsWon       int     `db:"bets_won"`
	BetsLost      int     `db:"bets_lost"`
	BetsVoided    int     `db:"bets_voided"`
	ActiveBets    int     `db:"active_bets"`
	TotalWinnings int     `db:"total_winnings"`
	TotalLosses   int     `db:"total_losses"`
	AvgBetSize    float64 `db:"avg_bet_size"`
	BiggestWin    int     `db:"biggest_win"`
	BiggestLoss   int     `db:"biggest_loss"`
	// Calculated fields
	WinRate      float64 `json:"win_rate"`
	WinLossRatio float64 `json:"win_loss_ratio"`
	NetProfit    int     `json:"net_profit"`
}
