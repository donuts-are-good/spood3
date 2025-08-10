# Custom Fighter Creation Feature Specification

## Overview
A premium shop feature that allows players to create and add their own fighters to the general fighter pool. Players purchase a "Combat License" and can design a fighter with a custom name while the system generates balanced stats and quirky characteristics. These custom fighters become permanent additions to the game and can be selected for daily fight schedules alongside existing fighters.

## Core Mechanics

### Purchase and Access
- **Shop Item**: "Combat License lvl 1" - 50,000 credits
- **Usage**: Single-use consumable item
- **Restrictions**: 
  - One fighter creation per user per week
  - Maximum 3 custom fighters per user total for lvl 1 license
  - Maximum 5 custom fighters per user total for lvl 2 license, increased cost, etc.
  - Username must be displayed on the created fighter's profile

### Fighter Creation Process
1. **Name Input**: User provides fighter name (3-50 characters)
2. **Stat Allocation**: User manually distributes 300 total stat points across 4 combat stats
3. **Chaos Stat Generation**: System assigns random quirky characteristics (gacha mechanic)
4. **Class Assignment**: Random fighter class from existing pool + new custom classes
5. **Final Review**: User can review their creation before finalizing
6. **Integration**: Fighter immediately enters the available fighter pool

### Stat Point Allocation System
- **Total Budget**: 300 stat points across 4 combat stats
- **Minimum per stat**: 20 points (ensures no completely useless stats)
- **Maximum per stat**: 120 points (prevents complete specialization)
- **User Control**: Player manually allocates points with real-time UI feedback
- **Strategic Choice**: Players can create specialized builds (glass cannon, tank, balanced, etc.)

#### Interactive Allocation Interface
```
Stat Point Allocation (300 total points)

Strength:     [████████████████████████████████████████████] 75/120  [+] [-]
Speed:        [████████████████████████████████████████] 60/120     [+] [-]
Endurance:    [████████████████████████████████████████████████] 85/120 [+] [-]
Technique:    [████████████████████████████████████████████████████████] 80/120 [+] [-]

Points Remaining: 0/300

[Distribute Evenly] [Reset to Minimums] [Quick Build: Glass Cannon] [Quick Build: Tank]
```

### Gacha Mechanic: Chaos Stats
The random element becomes the **chaos stats generation** - this creates excitement and surprise while giving players control over what actually matters for combat:

#### Chaos Stat Gacha Roll
When creating a fighter, players get a "chaos roll" that determines:
- **Blood Type**: Random from expanded pool
- **Horoscope**: Random from expanded pool  
- **Molecular Density**: 0.1 - 99.9
- **Existential Dread**: 1-100
- **Fingers**: Weighted random (usually 8-12, but can be wild)
- **Toes**: Weighted random (usually 8-12, but can be wild)
- **Ancestors**: 0-10,000
- **Fighter Class**: Random from custom creation pool

#### Gacha Rarity System
Some chaos stat combinations could be considered "rare":

**Common** (70% chance):
- Normal blood types, standard horoscopes
- Fingers/toes in 8-12 range
- Molecular density 10-90
- Existential dread 20-80

**Uncommon** (20% chance):
- Quirky blood types ("Caffeinated", "Meme Energy")
- Internet-age horoscopes ("Reply Guy", "Oversharer")
- Slightly weird finger/toe counts (6-7, 13-15)
- Extreme molecular density or existential dread

**Rare** (8% chance):
- Absurd blood types ("Monday Morning", "Imposter Syndrome")
- Very weird finger/toe counts (0-5, 16-20)
- Ultra-low or ultra-high chaos stats

**Legendary** (2% chance):
- Impossible combinations (0 fingers, 50+ toes)
- Maximum or minimum chaos stats
- Special hidden blood types unlocked only through creation

## Database Schema

### Shop Items Table Addition
```sql
INSERT INTO shop_items (name, description, emoji, price, item_type, effect_value) 
VALUES ('Combat License lvl 1', 'Create your own custom fighter to join the roster! Name them yourself, then let the Department of Recreational Violence handle the rest. Your fighter will be added to the pool and could be selected for any daily tournament.', '🥊', 50000, 'fighter_creation', 1);
```

### Fighter Table Modifications
```sql
ALTER TABLE fighters ADD COLUMN created_by_user_id INTEGER;
ALTER TABLE fighters ADD COLUMN is_custom BOOLEAN DEFAULT FALSE;
ALTER TABLE fighters ADD COLUMN creation_date DATETIME;
ALTER TABLE fighters ADD COLUMN custom_description TEXT;

-- Index for querying custom fighters
CREATE INDEX idx_fighters_custom ON fighters(is_custom, created_by_user_id);
```

## Custom Fighter Generation

### Combat Stat Allocation (User-Controlled)
```go
// Validation function for user-submitted stats
func ValidateStatAllocation(strength, speed, endurance, technique int) error {
    total := strength + speed + endurance + technique
    if total != 300 {
        return fmt.Errorf("total stat points must equal 300, got %d", total)
    }
    
    stats := []int{strength, speed, endurance, technique}
    statNames := []string{"strength", "speed", "endurance", "technique"}
    
    for i, stat := range stats {
        if stat < 20 {
            return fmt.Errorf("%s must be at least 20, got %d", statNames[i], stat)
        }
        if stat > 120 {
            return fmt.Errorf("%s cannot exceed 120, got %d", statNames[i], stat)
        }
    }
    
    return nil
}

// Quick build presets for convenience
func GetQuickBuildPresets() map[string][4]int {
    return map[string][4]int{
        "balanced":     {75, 75, 75, 75},
        "glass_cannon": {120, 100, 20, 60},
        "tank":         {60, 20, 120, 100},
        "speedster":    {40, 120, 60, 80},
        "technical":    {50, 70, 60, 120},
    }
}
```

### Chaos Stat Gacha Generation
```go
func GenerateChaosStatsGacha(seed int64) ChaosStats {
    rng := rand.New(rand.NewSource(seed))
    
    // Determine rarity tier first
    rarityRoll := rng.Float64()
    var rarity string
    switch {
    case rarityRoll < 0.02: // 2%
        rarity = "legendary"
    case rarityRoll < 0.10: // 8%
        rarity = "rare"
    case rarityRoll < 0.30: // 20%
        rarity = "uncommon"
    default: // 70%
        rarity = "common"
    }
    
    return ChaosStats{
        BloodType:        generateBloodType(rng, rarity),
        Horoscope:        generateHoroscope(rng, rarity),
        MolecularDensity: generateMolecularDensity(rng, rarity),
        ExistentialDread: generateExistentialDread(rng, rarity),
        Fingers:          generateFingers(rng, rarity),
        Toes:             generateToes(rng, rarity),
        Ancestors:        generateAncestors(rng, rarity),
        FighterClass:     generateFighterClass(rng, rarity),
        Rarity:           rarity,
    }
}

func generateBloodType(rng *rand.Rand, rarity string) string {
    switch rarity {
    case "legendary":
        types := []string{"Quantum Uncertainty", "The Void Itself", "Pure Determination"}
        return types[rng.Intn(len(types))]
    case "rare":
        types := []string{"Monday Morning", "Imposter Syndrome", "Social Anxiety", "Main Character Syndrome"}
        return types[rng.Intn(len(types))]
    case "uncommon":
        types := []string{"Caffeinated", "Meme Energy", "Discord Moderator", "Cryptocurrency Believer"}
        return types[rng.Intn(len(types))]
    default: // common
        types := []string{"A+", "B+", "AB+", "O+", "A-", "B-", "AB-", "O-", "Nacho Cheese"}
        return types[rng.Intn(len(types))]
    }
}

func generateFingers(rng *rand.Rand, rarity string) int {
    switch rarity {
    case "legendary":
        // Impossible finger counts
        extremes := []int{0, 1, 25, 30, 50, 100}
        return extremes[rng.Intn(len(extremes))]
    case "rare":
        // Very weird but not impossible
        return rng.Intn(5) + rng.Intn(16) // 0-20 range with weighting
    case "uncommon":
        // Slightly off normal
        if rng.Float64() < 0.5 {
            return rng.Intn(2) + 6 // 6-7
        } else {
            return rng.Intn(3) + 13 // 13-15
        }
    default: // common
        // Mostly normal with slight variation
        return rng.Intn(5) + 8 // 8-12
    }
}

// Similar functions for toes, molecular density, existential dread, etc.
```

### Chaos Stat Generation
Custom fighters get randomized chaos stats from expanded pools:

#### Blood Types (Custom Fighter Pool)
- All existing blood types +
- "User-Generated" 
- "Community Spirit"
- "Pure Determination"
- "Caffeinated"
- "Meme Energy"
- "Social Anxiety"
- "Imposter Syndrome"
- "Monday Morning"

#### Fighter Classes (Custom Fighter Pool)
- All existing classes +
- "Community-Forged"
- "User-Defined" 
- "Bespoke Violence"
- "Artisanal Combat"
- "Crowdsourced Chaos"
- "Democratic Destruction"
- "Collaborative Carnage"

#### Horoscopes (Custom Fighter Pool)
- Standard zodiac signs +
- "Aspiring Streamer"
- "Discord Moderator"
- "Cryptocurrency Believer"
- "Oversharer"
- "Reply Guy"
- "Main Character Syndrome"
- "Eternal Lurker"

#### Other Chaos Stats
- **Molecular Density**: 0.1 - 99.9 (same as existing)
- **Existential Dread**: 1-100 (same as existing)
- **Fingers**: Bias toward normal numbers but can still be absurd
- **Toes**: Similar to fingers
- **Ancestors**: 0-10,000 (same range as existing)

## Implementation Files

### New Files to Create
- `fighter/creation.go` - Custom fighter generation logic
- `fighter/validation.go` - Name validation and profanity filtering
- `templates/create-fighter.html` - Fighter creation interface
- `templates/custom-fighter-preview.html` - Preview before finalizing
- `static/js/fighter-creation.js` - Interactive stat generation UI

### Files to Modify
- `web/server.go` - Add fighter creation routes and handlers
- `database/repository.go` - Add custom fighter creation methods
- `web/server.go` - Add shop purchase handler for creation licenses
- `templates/fighter.html` - Show creator credit for custom fighters
- `templates/fighters.html` - Indicate custom fighters with special badges
- `fight/generator.go` - Include custom fighters in daily selection

## User Interface

### Fighter Creation Flow
1. **Purchase License**: Standard shop purchase flow
2. **Creation Page**: 
   - Name input field with real-time validation
   - **Interactive Stat Allocation**:
     - Four horizontal bars with +/- buttons
     - Real-time remaining points counter
     - Quick build preset buttons
     - Visual feedback when hitting min/max limits
   - **Chaos Stats Preview**: "??? ??? ??? ???" (hidden until generated)
   - "Generate Chaos Stats" button (single use)

3. **Chaos Stats Reveal**:
   - Display all generated chaos stats
   - Rarity indicator (Common/Uncommon/Rare/Legendary glow)
   - Celebration animation for rare rolls

4. **Preview Page**:
   - Full fighter profile preview
   - Combat stat visualization with your custom allocation
   - Chaos stats display with rarity indicators
   - "Confirm Creation" or "Go Back to Edit Stats"

5. **Success Page**:
   - Congratulations message
   - Rarity achievement notification
   - Link to view created fighter
   - Information about when fighter might appear in schedule

### Interactive Stat Allocation Interface
```html
<div class="stat-allocation">
    <h3>Allocate Combat Stats (300 Points Total)</h3>
    <div class="points-remaining">
        <span id="remaining-points">80</span> points remaining
    </div>
    
    <div class="stat-row">
        <label>Strength</label>
        <button class="stat-btn" onclick="adjustStat('strength', -1)">-</button>
        <div class="stat-bar">
            <div class="stat-fill" style="width: 62.5%"></div>
            <span class="stat-value">75/120</span>
        </div>
        <button class="stat-btn" onclick="adjustStat('strength', 1)">+</button>
    </div>
    
    <!-- Similar rows for Speed, Endurance, Technique -->
    
    <div class="quick-builds">
        <button onclick="applyPreset('balanced')">Balanced Build</button>
        <button onclick="applyPreset('glass_cannon')">Glass Cannon</button>
        <button onclick="applyPreset('tank')">Tank Build</button>
        <button onclick="applyPreset('speedster')">Speedster</button>
        <button onclick="applyPreset('technical')">Technical Fighter</button>
    </div>
</div>
```

### Gacha Chaos Stats Interface
```html
<div class="chaos-gacha">
    <h3>Chaos Stats Generation</h3>
    <div class="gacha-info">
        <p>Your fighter's personality and quirks are determined by fate!</p>
    </div>
    
    <div class="chaos-stats-grid">
        <div class="chaos-stat" data-rarity="unknown">
            <label>Blood Type</label>
            <div class="gacha-slot" id="blood-type">???</div>
        </div>
        <div class="chaos-stat" data-rarity="unknown">
            <label>Horoscope</label>
            <div class="gacha-slot" id="horoscope">???</div>
        </div>
        <div class="chaos-stat" data-rarity="unknown">
            <label>Fingers</label>
            <div class="gacha-slot" id="fingers">?</div>
        </div>
        <div class="chaos-stat" data-rarity="unknown">
            <label>Fighter Class</label>
            <div class="gacha-slot" id="fighter-class">???</div>
        </div>
        <!-- More chaos stats... -->
    </div>
    
    <div class="gacha-controls">
        <button id="generate-chaos" onclick="generateChaosStats()">🎲 Generate Chaos Stats!</button>
    </div>
    
    <div class="rarity-indicator" id="overall-rarity" style="display: none;">
        <span class="rarity-badge legendary">✨ LEGENDARY ROLL! ✨</span>
    </div>
</div>
```

### Gacha Animation System
```css
.gacha-slot {
    background: #222;
    border: 2px solid #444;
    padding: 10px;
    text-align: center;
    min-height: 40px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 5px;
    transition: all 0.3s ease;
}

.gacha-slot.rolling {
    animation: slot-spin 2s ease-in-out;
    background: linear-gradient(45deg, #ff6b6b, #4ecdc4, #45b7d1, #96ceb4);
    background-size: 400% 400%;
    animation: slot-spin 2s ease-in-out, rainbow-bg 1s ease-in-out infinite;
}

.chaos-stat[data-rarity="legendary"] .gacha-slot {
    border: 2px solid #ffd700;
    box-shadow: 0 0 20px rgba(255, 215, 0, 0.6);
    background: linear-gradient(135deg, #ffd700, #ffed4e);
    color: #000;
}

.chaos-stat[data-rarity="rare"] .gacha-slot {
    border: 2px solid #ff6b6b;
    box-shadow: 0 0 15px rgba(255, 107, 107, 0.4);
    background: linear-gradient(135deg, #ff6b6b, #ff8e8e);
    color: #fff;
}

.chaos-stat[data-rarity="uncommon"] .gacha-slot {
    border: 2px solid #4ecdc4;
    box-shadow: 0 0 10px rgba(78, 205, 196, 0.3);
    background: linear-gradient(135deg, #4ecdc4, #7ed7d1);
    color: #000;
}

@keyframes slot-spin {
    0% { transform: rotateY(0deg); }
    50% { transform: rotateY(180deg); }
    100% { transform: rotateY(360deg); }
}

@keyframes rainbow-bg {
    0% { background-position: 0% 50%; }
    50% { background-position: 100% 50%; }
    100% { background-position: 0% 50%; }
}
```

### Custom Fighter Display
- **Creator Badge**: Small indicator showing fighter was user-created
- **Creator Credit**: "Created by [Username]" on fighter profile
- **Custom Fighter Section**: Dedicated page listing all custom fighters
- **Search/Filter**: Ability to find custom fighters by creator

## Game Integration

### Fighter Pool Integration
- Custom fighters added to main `fighters` table with `is_custom=true`
- Included in daily fighter selection algorithm
- Same probability of selection as regular fighters
- Subject to all game mechanics (death, mutations, etc.)

### Daily Selection Algorithm Updates
```go
// In fight/generator.go SelectDailyFighters function
func (g *Generator) SelectDailyFighters(fighters []database.Fighter, date time.Time) []database.Fighter {
    // Include both regular and custom fighters
    allFighters := append(regularFighters, customFighters...)
    
    // Apply same selection logic
    // Custom fighters have equal chance of selection
}
```

## Economic Balance

### Pricing Strategy
- **Combat License lvl 1**: 50,000 credits (expensive but achievable)

### Player Engagement Mechanics
- **Strategic Depth**: Manual stat allocation creates meaningful choices
- **Gacha Excitement**: Random chaos stats provide thrill and collection appeal
- **Show-off Factor**: Rare chaos stat combinations become status symbols

### Credit Sink Purpose
- Removes credits from economy to prevent inflation
- Creates meaningful choice between betting and creating
- Encourages continued engagement for high-value purchases

### Psychological Engagement
- **Control + Chaos Balance**: Players control what matters (combat stats) but get surprised by personality
- **Sunk Cost**: Investment in stat allocation makes players more likely to accept chaos roll results
- **Collection Mentality**: Rare chaos combinations encourage multiple fighter creations

## Quality Control

### Name Validation
- Length: 3-50 characters
- Character restrictions: Letters, numbers, spaces, basic punctuation
- Duplicate name checking (must be unique) (can't use existing fighter names)


### Community Guidelines
- Department reserves right to modify or remove content

## Edge Cases and Considerations

### Technical Edge Cases
2. **Database Rollback**: Ensure fighter creation is atomic transaction
3. **Stat Validation**: Verify generated stats meet requirements
4. **Fighter Limits**: Enforce per-user and system-wide limits
