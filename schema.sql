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

-- Sample data to get started (only insert if tables are empty)
INSERT OR IGNORE INTO tournaments (week_number, name, sponsor, start_date) VALUES 
(1, 'The Brown Cup', 'Dude Wipes Recycling Co.', '2025-07-21'),
(2, 'Chaos Championship', 'Red Bull Energy Drinks', '2025-07-28'),
(3, 'The Existential Crisis Classic', 'Prozac by Pfizer', '2025-08-04'),
(4, 'Beef Supreme Beatdown', 'Arby''s Restaurant Group', '2025-08-11'),
(5, 'The Temporal Displacement Derby', 'DeLorean Motor Company', '2025-08-18'),
(6, 'Finger Counting Finals', 'Texas Instruments Calculators', '2025-08-25'),
(7, 'The Molecular Meltdown', 'Tide Laundry Detergent', '2025-09-01'),
(8, 'Ancestors Anonymous', 'Ancestry.com DNA Testing', '2025-09-08'),
(9, 'The Horoscope Havoc', 'Susan Miller''s Astrology Zone', '2025-09-15'),
(10, 'Blood Type Brawl', 'American Red Cross', '2025-09-22'),
(11, 'The Participation Trophy Massacre', 'Everyone Gets A Medal Co.', '2025-09-29'),
(12, 'Midlife Crisis Championship', 'Harley-Davidson Motorcycles', '2025-10-06'),
(13, 'The Finger Lickin Fuckdown', 'KFC Original Recipe', '2025-10-13'),
(14, 'Daddy Issues Derby', 'Therapy.com Online Counseling', '2025-10-20'),
(15, 'The Social Media Meltdown', 'Meta (Facebook) Corporation', '2025-10-27'),
(16, 'Karen Manager Kombat', 'Whole Foods Market', '2025-11-03'),
(17, 'The Florida Man Fiesta', 'Bath Salts R Us', '2025-11-10'),
(18, 'The Gluten Games', 'Celiac Disease', '2025-11-17'),
(19, 'The Cryptocurrency Catastrophe', 'Dogecoin Foundation', '2025-11-24'),
(20, 'Influencer Implosion Invitational', 'OnlyFans Content Platform', '2025-12-01'),
(21, 'The Debt Derby', 'Sallie Mae Servicing', '2025-12-08'),
(22, 'Climate Chaos Cup', 'ExxonMobil Corporation', '2025-12-15'),
(23, 'The NFTs Are Totally Back Cup', 'OpenSea Marketplace', '2025-12-22'),
(24, 'Avocado Toast Apocalypse', 'Millennial Mortgage Denials Inc.', '2025-12-29');

-- Sample fighters (only if fighters table is empty)
INSERT OR IGNORE INTO fighters (name, team, strength, speed, endurance, technique, blood_type, horoscope, molecular_density, existential_dread, fingers, toes, ancestors, fighter_class) VALUES 
('Beef Supreme', 'The Meat Sweats', 95, 60, 80, 70, 'AB+', 'Capricorn', 2.4, 15, 10, 10, 47, 'Existential'),
('Pork Chop Paradise', 'The Meat Sweats', 90, 65, 85, 75, 'nacho cheese', 'Taurus', 3.2, 8, 9, 11, 23, 'Existential'),
('Bacon Windfall', 'The Meat Sweats', 88, 70, 82, 78, 'O+', 'Leo', 2.8, 18, 10, 10, 89, 'Conceptual'),
('Sausage Festival', 'The Meat Sweats', 85, 68, 90, 72, 'A-', 'Sagittarius', 3.1, 22, 8, 12, 156, 'Existential'),
('Lady Chaos', 'Entropy United', 70, 90, 75, 85, 'O-', 'Scorpio', 1.8, 95, 8, 12, 3, 'Temporal'),
('Disorder McFrenzy', 'Entropy United', 68, 95, 70, 88, 'static electricity', 'Aquarius', 1.5, 92, 12, 8, 7, 'Temporal'),
('Mayhem Breakfast', 'Entropy United', 75, 88, 72, 90, 'AB-', 'Gemini', 1.9, 87, 9, 11, 15, 'Temporal'),
('Anarchy Moonbeam', 'Entropy United', 65, 92, 68, 95, 'B+', 'Pisces', 1.6, 98, 11, 9, 2, 'Temporal'),
('Chaos Windfall', 'Entropy United', 72, 85, 75, 85, 'O+', 'Scorpio', 1.7, 90, 10, 10, 12, 'Temporal'),
('Professor Punches', 'Free Agent', 80, 75, 85, 95, 'B+', 'Gemini', 3.1, 40, 11, 9, 156, 'Conceptual'),
('The Finger Counter', 'Free Agent', 65, 85, 70, 90, 'A-', 'Virgo', 2.7, 25, 23, 7, 89, 'Metaphysical'),
('Glizzy Goblin', 'Free Agent', 75, 70, 95, 80, 'O+', 'Virgo', 1.5, 78, 6, 14, 1247, 'Emotional'), 
('Donny "Dirty Mouth" McBubble', 'The Tide Pods', 85, 70, 80, 88, 'fabric softener', 'Aquarius', 2.9, 25, 10, 10, 67, 'Conceptual'),
('Suds Sullivan', 'The Tide Pods', 90, 65, 85, 82, 'A+', 'Virgo', 3.2, 15, 10, 10, 89, 'Conceptual'),
('Fresh Rodriguez', 'The Tide Pods', 75, 90, 70, 95, 'B-', 'Gemini', 2.1, 35, 12, 8, 123, 'Temporal'),
('Rinse Jackson', 'The Tide Pods', 95, 55, 95, 75, 'O+', 'Taurus', 3.8, 10, 10, 10, 45, 'Existential'),
('Spotless Chen', 'The Tide Pods', 70, 85, 75, 98, 'AB-', 'Scorpio', 2.4, 45, 11, 9, 234, 'Metaphysical'),
('Polymer McGillicuddy', 'The Dupont Mutants', 88, 75, 90, 80, 'polytetrafluoroethylene', 'Cancer', 4.2, 65, 13, 7, 234, 'Conceptual'),
('Nylon Dreams', 'The Dupont Mutants', 70, 95, 75, 85, 'A-', 'Cancer', 1.8, 55, 10, 10, 156, 'Temporal'),
('Kevlar Breakfast', 'The Dupont Mutants', 95, 65, 95, 70, 'B+', 'Cancer', 5.1, 45, 10, 10, 89, 'Existential'),
('Teflon Moonbeam', 'The Dupont Mutants', 80, 85, 80, 88, 'synthetic polymers', 'Cancer', 3.7, 70, 11, 9, 345, 'Metaphysical'),
('Spandex Nightmare', 'The Dupont Mutants', 75, 90, 78, 92, 'O+', 'Cancer', 2.9, 50, 12, 8, 567, 'Emotional'),
('Beef Injection', 'Needle Exchange', 92, 85, 88, 75, 'steroids', 'Aries', 3.8, 5, 10, 10, 234, 'Existential'),
('Swole Patroll', 'Needle Exchange', 98, 70, 95, 65, 'A+', 'Aries', 4.2, 8, 10, 10, 156, 'Existential'),
('Gains McGainface', 'Needle Exchange', 95, 75, 90, 70, 'Dat Dere Celltech', 'Aries', 3.9, 12, 10, 10, 89, 'Existential'),
('Booger McGillicuddy', 'The Participation Trophies', 45, 55, 50, 60, 'glue stick residue', 'Virgo', 1.2, 85, 7, 13, 12, 'Emotional'),
('Fart Sniffer Supreme', 'The Participation Trophies', 40, 50, 45, 65, 'A-', 'Cancer', 0.8, 90, 9, 11, 23, 'Existential'),
('Wedgie Magnet', 'The Participation Trophies', 35, 60, 55, 55, 'lunch money', 'Pisces', 1.1, 88, 8, 12, 34, 'Emotional'),
('Pocket Protector Pete', 'The Participation Trophies', 50, 40, 60, 70, 'B+', 'Virgo', 1.5, 75, 10, 10, 456, 'Conceptual'),
('Artisanal Windbreaker', 'The Participation Trophies', 42, 65, 48, 68, 'organic methane', 'Aquarius', 0.9, 92, 11, 9, 67, 'Metaphysical'),
('Social Anxiety Sam', 'The Participation Trophies', 38, 45, 52, 72, 'O-', 'Cancer', 1.3, 95, 6, 14, 89, 'Emotional'),
('Mommy Issues Mike', 'The Participation Trophies', 55, 48, 65, 58, 'AB+', 'Scorpio', 1.4, 87, 10, 10, 123, 'Emotional'),
('Participation Trophy Tom', 'The Participation Trophies', 46, 52, 58, 62, 'participation ribbon', 'Libra', 1.0, 80, 12, 8, 234, 'Existential'),
('Last Pick Larry', 'The Participation Trophies', 41, 58, 44, 66, 'B-', 'Gemini', 1.2, 83, 9, 11, 345, 'Emotional'),
('Basement Dweller Brad', 'The Participation Trophies', 43, 42, 62, 75, 'mountain dew code red', 'Capricorn', 2.1, 78, 14, 6, 567, 'Metaphysical'),
('Snotty Scotty', 'Garbage Pail Kids', 60, 85, 55, 70, 'boogers', 'Gemini', 1.8, 25, 10, 10, 89, 'Conceptual'),
('Messy Tessie', 'Garbage Pail Kids', 65, 80, 60, 75, 'A+', 'Leo', 2.1, 30, 11, 9, 123, 'Emotional'),
('Nasty Nick', 'Garbage Pail Kids', 70, 75, 65, 80, 'garbage juice', 'Scorpio', 1.9, 35, 9, 11, 156, 'Existential'),
('Stinky Pete', 'Garbage Pail Kids', 55, 90, 50, 85, 'B-', 'Aries', 1.7, 40, 12, 8, 234, 'Temporal'),
('Gross Gary', 'Garbage Pail Kids', 75, 70, 70, 65, 'expired milk', 'Taurus', 2.3, 20, 8, 12, 345, 'Conceptual'),
('Chorby Short', 'Free Agent', 45, 95, 40, 85, 'AB+', 'Gemini', 1.2, 65, 10, 10, 234, 'Temporal'),
('Beans McBlase', 'Free Agent', 70, 80, 75, 90, 'legume juice', 'Virgo', 2.1, 45, 10, 10, 156, 'Conceptual'),
('Baby Triumphant', 'Free Agent', 35, 60, 30, 98, 'formula', 'Leo', 0.8, 5, 5, 5, 1, 'Existential'),
('Brisket Friendo', 'Free Agent', 85, 65, 80, 75, 'O+', 'Taurus', 2.8, 25, 10, 10, 89, 'Emotional'),
('Cell Longarms', 'Free Agent', 60, 85, 70, 88, 'mitochondria', 'Cancer', 1.5, 55, 22, 8, 345, 'Metaphysical'),
('Comfort Septemberish', 'Free Agent', 50, 70, 95, 85, 'A-', 'Libra', 1.9, 15, 10, 10, 567, 'Emotional'),
('Engine Eberhardt', 'Free Agent', 90, 75, 85, 70, 'motor oil', 'Aries', 3.5, 20, 10, 10, 123, 'Conceptual'),
('Fish Summer', 'Free Agent', 40, 98, 60, 82, 'B+', 'Pisces', 1.1, 40, 10, 10, 789, 'Temporal'),
('Grollis Zephyr', 'Free Agent', 75, 90, 65, 88, 'wind essence', 'Aquarius', 0.9, 70, 14, 6, 234, 'Metaphysical'),
('Jelly Burgertoes', 'Free Agent', 55, 80, 70, 92, 'O-', 'Scorpio', 1.7, 50, 8, 12, 456, 'Emotional'),
('Kennedy Loser', 'Free Agent', 30, 40, 35, 45, 'tears of defeat', 'Capricorn', 1.0, 99, 10, 10, 678, 'Existential'),
('Millipede Aqualuft', 'Free Agent', 65, 85, 75, 80, 'A+', 'Gemini', 2.2, 60, 1000, 1000, 890, 'Metaphysical'),
('Pangolin Ruiz', 'Free Agent', 80, 50, 95, 70, 'armor plating', 'Virgo', 4.8, 35, 10, 10, 123, 'Existential'),
('Sixpack Santiago', 'Free Agent', 88, 70, 92, 65, 'beer', 'Sagittarius', 2.6, 30, 10, 10, 345, 'Conceptual'),
('Slosh Gulp', 'Free Agent', 60, 75, 80, 85, 'AB-', 'Pisces', 1.4, 85, 10, 10, 567, 'Emotional'),
('Stretch Filigree', 'Free Agent', 50, 95, 85, 90, 'elastic polymers', 'Libra', 0.7, 45, 20, 4, 789, 'Temporal'),
('Stu Trololol', 'Free Agent', 25, 60, 40, 95, 'meme juice', 'Gemini', 1.3, 75, 10, 10, 2008, 'Metaphysical'),
('Tahini Poinsettia', 'Free Agent', 70, 85, 75, 88, 'B-', 'Capricorn', 2.3, 55, 10, 10, 234, 'Emotional'),
('Xandra Pancakes', 'Free Agent', 65, 80, 70, 92, 'maple syrup', 'Cancer', 2.0, 25, 10, 10, 456, 'Conceptual'),
('Yurts Buttercup', 'Free Agent', 78, 88, 82, 75, 'O+', 'Taurus', 2.4, 40, 10, 10, 678, 'Temporal'),
('War Machine', 'Free Agent', 85, 70, 80, 60, 'roid rage', 'Aries', 2.8, 10, 10, 10, 234, 'Existential'),
('Stone Cold Steve Austin', 'Free Agent', 95, 75, 90, 80, 'beer', 'Scorpio', 3.2, 15, 10, 10, 456, 'Existential'),
('The Rock Johnson', 'Free Agent', 90, 80, 85, 85, 'peoples elbow grease', 'Taurus', 3.5, 20, 10, 10, 789, 'Conceptual'),
('Mike Tyson', 'Free Agent', 98, 85, 80, 70, 'ear wax', 'Cancer', 2.9, 25, 10, 10, 234, 'Existential'),
('Arnold Schwarzenegger', 'Free Agent', 95, 65, 88, 75, 'A+', 'Leo', 4.2, 10, 10, 10, 567, 'Temporal'),
('Chuck Norris', 'Free Agent', 100, 90, 100, 100, 'pure awesome', 'All of them', 9.9, 0, 10, 10, 999999, 'Metaphysical'),
('Jean-Claude Van Damme', 'Free Agent', 80, 95, 75, 92, 'splits energy', 'Libra', 2.1, 30, 10, 10, 345, 'Temporal'),
('Steven Seagal', 'Free Agent', 60, 40, 70, 85, 'ponytail grease', 'Virgo', 2.8, 85, 10, 10, 123, 'Emotional'),
('John Cena', 'Free Agent', 95, 85, 90, 80, 'muscle milk', 'Aries', 3.1, 15, 10, 10, 456, 'Existential'),
('Marilyn Manson', 'Free Agent', 70, 80, 75, 88, 'black eyeliner', 'Scorpio', 1.8, 95, 10, 10, 666, 'Existential'),
('Nick Carter', 'Free Agent', 45, 70, 60, 85, 'hair gel', 'Aquarius', 1.5, 75, 10, 10, 234, 'Emotional'),
('Brian Littrell', 'Free Agent', 50, 75, 65, 90, 'A+', 'Pisces', 1.6, 70, 10, 10, 345, 'Emotional'),
('AJ McLean', 'Free Agent', 55, 80, 70, 88, 'frosted tips', 'Gemini', 1.7, 80, 10, 10, 456, 'Emotional'),
('Howie Dorough', 'Free Agent', 40, 65, 55, 82, 'B-', 'Leo', 1.4, 85, 10, 10, 567, 'Emotional'),
('Kevin Richardson', 'Free Agent', 60, 70, 75, 85, 'O+', 'Libra', 1.8, 65, 10, 10, 678, 'Emotional'),
('Justin Timberlake', 'Free Agent', 65, 85, 80, 92, 'ramen noodles', 'Aquarius', 2.0, 45, 10, 10, 789, 'Temporal'),
('JC Chasez', 'Free Agent', 55, 80, 70, 88, 'AB-', 'Virgo', 1.7, 60, 10, 10, 234, 'Emotional'),
('Chris Kirkpatrick', 'Free Agent', 50, 75, 65, 85, 'purple hair dye', 'Aries', 1.5, 70, 10, 10, 345, 'Emotional'),
('Joey Fatone', 'Free Agent', 70, 60, 85, 75, 'marinara sauce', 'Taurus', 2.5, 55, 10, 10, 456, 'Conceptual'),
('Lance Bass', 'Free Agent', 45, 70, 60, 80, 'space dust', 'Sagittarius', 1.3, 75, 10, 10, 567, 'Metaphysical'),
('James Hetfield', 'Free Agent', 90, 75, 95, 85, 'molten metal', 'Leo', 3.8, 40, 10, 10, 678, 'Existential'),
('Lars Ulrich', 'Napster', 60, 95, 70, 90, 'cat farts', 'Aries', 2.2, 65, 12, 8, 789, 'Temporal'),
('Kirk Hammett', 'Free Agent', 75, 85, 80, 98, 'guitar picks', 'Scorpio', 2.6, 50, 11, 9, 234, 'Conceptual'),
('Robert Trujillo', 'Free Agent', 80, 80, 88, 85, 'A-', 'Capricorn', 2.8, 35, 10, 10, 345, 'Existential'),
('John Lennon', 'Free Agent', 50, 70, 65, 95, 'peace and love', 'Libra', 1.9, 90, 10, 10, 456, 'Emotional'),
('Paul McCartney', 'Free Agent', 55, 75, 80, 92, 'B+', 'Gemini', 2.1, 25, 10, 10, 567, 'Temporal'),
('George Harrison', 'Free Agent', 60, 80, 75, 98, 'spiritual enlightenment', 'Pisces', 1.7, 15, 10, 10, 678, 'Metaphysical'),
('Ringo Starr', 'Free Agent', 45, 70, 70, 85, 'yellow submarine fuel', 'Cancer', 1.8, 40, 10, 10, 789, 'Emotional'),
('Gordon Ramsay', 'Free Agent', 85, 90, 80, 95, 'lamb sauce', 'Scorpio', 2.8, 60, 10, 10, 345, 'Emotional'),
('Bob Ross', 'Free Agent', 30, 40, 95, 100, 'happy little accidents', 'Pisces', 1.2, 5, 10, 10, 789, 'Emotional'),
('Steve Irwin', 'Free Agent', 80, 85, 90, 88, 'crocodile tears', 'Sagittarius', 2.5, 20, 10, 10, 456, 'Conceptual'),
('Mr. Rogers', 'Free Agent', 25, 35, 100, 100, 'pure kindness', 'Cancer', 1.0, 0, 10, 10, 234, 'Emotional'),
('Danny DeVito', 'Free Agent', 40, 60, 70, 85, 'trash juice', 'Scorpio', 3.8, 75, 8, 12, 567, 'Existential'),
('Shrek', 'Free Agent', 95, 50, 90, 60, 'onion layers', 'Taurus', 4.2, 45, 10, 10, 1, 'Existential'),
('Nicolas Cage', 'Free Agent', 70, 80, 75, 100, 'declaration of independence ink', 'Gemini', 2.3, 85, 10, 10, 678, 'Temporal'),
('Keanu Reeves', 'Free Agent', 85, 95, 80, 90, 'breathtaking', 'Libra', 2.1, 10, 10, 10, 2000, 'Metaphysical'),
('Betty White', 'Free Agent', 60, 75, 100, 95, 'golden girl essence', 'Capricorn', 1.8, 5, 10, 10, 99, 'Emotional'),
('Gilbert Gottfried', 'Free Agent', 30, 70, 65, 88, 'vocal cord cheese', 'Aries', 1.5, 90, 10, 10, 345, 'Existential'),
('Verne Troyer', 'Free Agent', 35, 70, 60, 85, 'XXS', 'Capricorn', 1.1, 55, 8, 8, 345, 'Emotional'),
('Warwick Davis', 'Free Agent', 50, 75, 70, 90, 'magic', 'Pisces', 1.4, 35, 10, 10, 456, 'Metaphysical'),
('Hornswoggle', 'Free Agent', 55, 90, 70, 80, 'WWE contract tears', 'Aries', 1.6, 70, 10, 10, 678, 'Existential'),

-- The Corporate Overlords - Soul-crushing capitalism incarnate
('Regional Manager Dave', 'The Corporate Overlords', 70, 45, 85, 90, 'corporate synergy', 'Capricorn', 2.8, 95, 10, 10, 847, 'Existential'),
('Karen "Compliance" Thompson', 'The Corporate Overlords', 65, 70, 80, 95, 'passive aggression', 'Virgo', 2.1, 88, 10, 10, 156, 'Emotional'),
('Brad "PowerPoint" Stevens', 'The Corporate Overlords', 55, 80, 75, 85, 'buzzword juice', 'Gemini', 1.9, 78, 10, 10, 234, 'Conceptual'),
('The Efficiency Expert', 'The Corporate Overlords', 85, 85, 70, 98, 'liquidated dreams', 'Scorpio', 3.4, 92, 10, 10, 445, 'Temporal'),
('Quarterly Report Jones', 'The Corporate Overlords', 60, 75, 90, 88, 'performance metrics', 'Aquarius', 2.6, 87, 10, 10, 567, 'Metaphysical'),

-- Suburban Nightmares - The terror of middle-class conformity
('Soccer Mom Supreme', 'Suburban Nightmares', 75, 95, 80, 70, 'organic kale smoothie', 'Cancer', 2.2, 65, 10, 10, 123, 'Emotional'),
('HOA President Peterson', 'Suburban Nightmares', 80, 60, 95, 85, 'property value anxiety', 'Taurus', 3.1, 75, 10, 10, 345, 'Existential'),
('Grill Master Gary', 'Suburban Nightmares', 90, 55, 85, 75, 'propane and propane accessories', 'Leo', 2.9, 45, 10, 10, 789, 'Conceptual'),
('PTA Meeting Patricia', 'Suburban Nightmares', 65, 85, 70, 92, 'fundraising fury', 'Libra', 2.0, 82, 10, 10, 234, 'Emotional'),
('Lawn Care Larry', 'Suburban Nightmares', 70, 75, 88, 80, 'pesticide residue', 'Virgo', 2.5, 55, 10, 10, 456, 'Temporal'),

-- The Retirement Home Raiders - Geriatric violence at its finest
('Bingo Bertha "The Destroyer"', 'The Retirement Home Raiders', 45, 40, 95, 98, 'arthritis cream', 'Pisces', 1.1, 25, 8, 9, 4567, 'Metaphysical'),
('Wheelchair Willie "No Mercy"', 'The Retirement Home Raiders', 60, 85, 75, 90, 'pension fund tears', 'Sagittarius', 1.8, 35, 7, 11, 3456, 'Temporal'),
('Denture Deborah', 'The Retirement Home Raiders', 55, 70, 80, 85, 'denture adhesive', 'Cancer', 1.4, 40, 6, 12, 2345, 'Existential'),
('Early Bird Eddie', 'The Retirement Home Raiders', 65, 90, 70, 88, 'early bird special sauce', 'Capricorn', 1.6, 60, 9, 8, 5678, 'Emotional'),
('Shuffleboard Shirley', 'The Retirement Home Raiders', 50, 75, 85, 95, 'medicare supplement', 'Aquarius', 1.3, 50, 10, 7, 6789, 'Conceptual');

-- Shop items seed data
INSERT OR IGNORE INTO shop_items (name, description, emoji, price, item_type, effect_value) VALUES 
('Fighter Curse', 'Weaken a fighter before their next fight', '💀', 100, 'fighter_curse', 10000),
('Fighter Blessing', 'Strengthen a fighter before their next fight', '✨', 100, 'fighter_blessing', 10000),
('Sacrifice to the Gods', 'Burn credits for cosmic luck', '🔥', 1, 'sacrifice', 0),
('Governance Vote', 'Voice in the chaos democracy', '🗳️', 20, 'governance_vote', 1),
('Player Curse', 'Tax another player''s winnings', '👹', 2000, 'player_curse', 10),
('Player Blessing', 'Boost another player''s winnings', '👼', 2000, 'player_blessing', 10),
('MVP Player lvl 1', 'Choose a favorite fighter - earn 10,000 credits when they win', '👑', 15000, 'mvp_player', 10000);