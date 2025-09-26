-- Spoodblort Database Schema
-- Safe to run multiple times - uses IF NOT EXISTS patterns

-- Enable foreign key constraints
PRAGMA foreign_keys = ON;

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    discord_id TEXT UNIQUE NOT NULL,
    username TEXT NOT NULL,
    custom_username TEXT,
    avatar_url TEXT,
    credits INTEGER DEFAULT 1000,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Tournaments table (weekly tournaments)
CREATE TABLE IF NOT EXISTS tournaments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    week_number INTEGER UNIQUE NOT NULL,
    name TEXT NOT NULL,
    sponsor TEXT NOT NULL,
    start_date DATE NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Fighters table
CREATE TABLE IF NOT EXISTS fighters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    team TEXT,
    -- Traditional stats
    strength INTEGER NOT NULL,
    speed INTEGER NOT NULL,
    endurance INTEGER NOT NULL,
    technique INTEGER NOT NULL,
    -- Chaos stats
    blood_type TEXT,
    horoscope TEXT,
    molecular_density REAL,
    existential_dread INTEGER,
    fingers INTEGER,
    toes INTEGER,
    ancestors INTEGER,
    fighter_class TEXT, -- Conceptual, Temporal, Emotional, etc.
    -- Win/Loss record
    wins INTEGER DEFAULT 0,
    losses INTEGER DEFAULT 0,
    draws INTEGER DEFAULT 0,
    -- Status
    is_dead BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Fights table (denormalized fighter names for easy querying)
CREATE TABLE IF NOT EXISTS fights (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tournament_id INTEGER NOT NULL,
    -- Fighter references
    fighter1_id INTEGER NOT NULL,
    fighter2_id INTEGER NOT NULL,
    -- Denormalized for easy display (no JOINs needed)
    fighter1_name TEXT NOT NULL,
    fighter2_name TEXT NOT NULL,
    -- Scheduling
    scheduled_time DATETIME NOT NULL,
    status TEXT DEFAULT 'scheduled' CHECK (status IN ('scheduled', 'active', 'completed', 'voided')),
    -- Results
    winner_id INTEGER,
    final_score1 INTEGER,
    final_score2 INTEGER,
    completed_at DATETIME,
    voided_reason TEXT,
    -- Metadata
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tournament_id) REFERENCES tournaments(id),
    FOREIGN KEY (fighter1_id) REFERENCES fighters(id),
    FOREIGN KEY (fighter2_id) REFERENCES fighters(id),
    FOREIGN KEY (winner_id) REFERENCES fighters(id)
);

-- Bets table
CREATE TABLE IF NOT EXISTS bets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    fight_id INTEGER NOT NULL,
    fighter_id INTEGER NOT NULL, -- which fighter they bet on
    amount INTEGER NOT NULL,
    status TEXT DEFAULT 'pending' CHECK (status IN ('pending', 'won', 'lost', 'voided')),
    payout INTEGER, -- amount won (if won)
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    resolved_at DATETIME,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (fight_id) REFERENCES fights(id),
    FOREIGN KEY (fighter_id) REFERENCES fighters(id)
);

-- Sessions table for user authentication
CREATE TABLE IF NOT EXISTS sessions (
    token TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Shop items table
CREATE TABLE IF NOT EXISTS shop_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    emoji TEXT NOT NULL,
    price INTEGER NOT NULL,
    item_type TEXT NOT NULL, -- 'fighter_curse', 'fighter_blessing', 'sacrifice', 'governance_vote', 'player_curse', 'player_blessing'
    effect_value INTEGER DEFAULT 0, -- For curses/blessings magnitude
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- User inventory table
CREATE TABLE IF NOT EXISTS user_inventory (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    shop_item_id INTEGER NOT NULL,
    quantity INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (shop_item_id) REFERENCES shop_items(id)
);

-- Applied effects table (track what's been used on fights/players)
CREATE TABLE IF NOT EXISTS applied_effects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    target_type TEXT NOT NULL, -- 'fight', 'player'
    target_id INTEGER NOT NULL, -- fight_id or target_user_id
    effect_type TEXT NOT NULL,
    effect_value INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- User settings table for MVP tracking and other user preferences
CREATE TABLE IF NOT EXISTS user_settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    setting_type TEXT NOT NULL, -- 'mvp_player'
    setting_value TEXT NOT NULL, -- fighter_id for MVP
    can_change_at DATETIME, -- when they can change their MVP (next tournament or after paying)
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE(user_id, setting_type) -- One setting per type per user
);

-- Champion legacy history (Saturday winners)
CREATE TABLE IF NOT EXISTS champion_legacy_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    fight_id INTEGER NOT NULL,
    fighter_id INTEGER NOT NULL,
    tournament_id INTEGER NOT NULL,
    tournament_week INTEGER,
    tournament_name TEXT,
    stat_awarded TEXT NOT NULL,
    stat_delta INTEGER NOT NULL,
    total_wagered INTEGER DEFAULT 0,
    total_payout INTEGER DEFAULT 0,
    blessings_count INTEGER DEFAULT 0,
    curses_count INTEGER DEFAULT 0,
    awarded_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(fight_id),
    FOREIGN KEY (fight_id) REFERENCES fights(id),
    FOREIGN KEY (fighter_id) REFERENCES fighters(id),
    FOREIGN KEY (tournament_id) REFERENCES tournaments(id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_fights_tournament_date ON fights(tournament_id, scheduled_time);
CREATE INDEX IF NOT EXISTS idx_fights_status ON fights(status);
CREATE INDEX IF NOT EXISTS idx_fights_scheduled_time ON fights(scheduled_time);
CREATE INDEX IF NOT EXISTS idx_bets_user_id ON bets(user_id);
CREATE INDEX IF NOT EXISTS idx_bets_fight_id ON bets(fight_id);
CREATE INDEX IF NOT EXISTS idx_users_discord_id ON users(discord_id);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_user_inventory_user_id ON user_inventory(user_id);
CREATE INDEX IF NOT EXISTS idx_user_inventory_item_id ON user_inventory(shop_item_id);
CREATE INDEX IF NOT EXISTS idx_applied_effects_target ON applied_effects(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_applied_effects_user_id ON applied_effects(user_id);
CREATE INDEX IF NOT EXISTS idx_user_settings_user_id ON user_settings(user_id);
CREATE INDEX IF NOT EXISTS idx_user_settings_type ON user_settings(setting_type);

-- Example of how to safely add new columns later:
-- ALTER TABLE fighters ADD COLUMN new_stat INTEGER DEFAULT 0;
-- This won't break existing data!
