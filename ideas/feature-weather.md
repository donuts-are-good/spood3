# Weather System Feature Specification

## Overview
A daily weather system that applies chaotic atmospheric effects to all fights on a given day. Weather changes daily at midnight Central Time and affects fight mechanics, damage calculations, and adds flavor text to commentary.

## Core Mechanics

### Weather Generation
- **Timing**: Weather is determined once per day at midnight Central Time
- **Persistence**: Weather lasts for the entire 24-hour period
- **Randomization**: Uses a daily seed based on the date to ensure consistency across server restarts
- **Probability**: Each weather type has different rarity levels (common, uncommon, rare, legendary)

### Weather Types

#### Common Weather (60% chance)
1. **Clear Skies** 
   - No mechanical effects
   - Flavor: "Perfect conditions for recreational violence"

2. **Light Drizzle**
   - 5% chance fighters slip and miss attacks (deal 0 damage that tick)
   - Flavor: "The moisture makes everything slippery and sad"

3. **Overcast**
   - Existential dread increases by 10 for all fighters
   - Flavor: "The gray sky reflects the meaninglessness of combat"

#### Uncommon Weather (25% chance)
4. **Blood Rain**
   - All damage increased by 50%
   - Fighter health bars turn deep red
   - Flavor: "The violence gods are pleased with today's offerings"

5. **Temporal Fog**
   - Fight rounds have random lengths (3-9 ticks instead of 6)
   - Round transitions happen unpredictably
   - Flavor: "Time itself is confused and disoriented"

6. **Static Storm**
   - All fighter stats fluctuate ±10 each tick
   - Stats can temporarily go negative or exceed 100
   - Flavor: "Electromagnetic interference scrambles neural pathways"

#### Rare Weather (12% chance)
7. **Quantum Uncertainty**
   - 10% chance per tick that fighters swap one random stat
   - Effects last until fight ends
   - Flavor: "Reality operates on suggestions rather than laws"

8. **Void Winds**
   - 5% chance per tick that a fighter becomes "partially voided" (takes 50% damage for 3 ticks)
   - Flavor: "The emptiness between atoms grows hungry"

9. **Molecular Instability**
   - All stats become decimal numbers (displayed with 2 decimal places)
   - Damage calculations become more chaotic
   - Flavor: "Atomic bonds are merely polite recommendations"

#### Legendary Weather (3% chance)
10. **Reality Malfunction**
    - Multiple weather effects active simultaneously
    - Commentary becomes increasingly incoherent
    - The Commissioner's comments become more frequent and ominous
    - Flavor: "The Department's reality maintenance budget was insufficient"

11. **Existential Eclipse**
    - Death chance increases to 1 in 10,000 (from 1 in 100,000)
    - All fighters gain 50 existential dread
    - Commentary focuses on mortality and meaninglessness
    - Flavor: "The sun has decided to take a mental health day"

12. **Chaos Singularity**
    - All normal fight rules suspended
    - Damage becomes completely random (1-10,000 per tick)
    - Fighter names occasionally get scrambled
    - Flavor: "Mathematics has filed a restraining order against physics"

## Database Schema

### New Table: `weather_events`
```sql
CREATE TABLE weather_events (
    id INTEGER PRIMARY KEY,
    date DATE UNIQUE NOT NULL,
    weather_type TEXT NOT NULL,
    weather_name TEXT NOT NULL,
    description TEXT NOT NULL,
    effects_json TEXT, -- JSON object containing effect parameters
    rarity TEXT NOT NULL, -- 'common', 'uncommon', 'rare', 'legendary'
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Weather Effect Storage
Effects are stored as JSON in the `effects_json` column:
```json
{
    "damage_multiplier": 1.5,
    "stat_fluctuation": 10,
    "death_chance_modifier": 0.1,
    "special_effects": ["stat_swap", "partial_void"]
}
```

## Implementation Files

### New Files to Create
- `weather/generator.go` - Weather generation logic and seeding
- `weather/effects.go` - Weather effect application during fights
- `weather/types.go` - Weather type definitions and constants
- `database/weather.go` - Database operations for weather events
- `templates/weather-widget.html` - UI component showing current weather

### Files to Modify
- `fight/engine.go` - Apply weather effects during fight simulation
- `fight/commentary.go` - Add weather-specific commentary
- `web/server.go` - Add weather data to page templates
- `templates/index.html` - Display current weather
- `templates/fight.html` - Show weather effects on fight page
- `scheduler/scheduler.go` - Generate daily weather during schedule creation

## User Interface

### Weather Display
- **Homepage**: Prominent weather widget showing current conditions
- **Fight Pages**: Weather effects indicator with tooltip explaining impacts
- **Fighter Pages**: Note if weather would affect this fighter's performance
- **Watch Pages**: Weather effects integrated into live commentary

### Weather Widget Components
- Weather icon/emoji
- Weather name and description
- Effect summary ("All damage +50%")
- Flavor text
- Time until weather changes

## Edge Cases and Considerations

### Technical Edge Cases
1. **Server Restart**: Weather must persist through restarts (database-backed)
2. **Timezone Handling**: All weather changes at midnight Central Time regardless of user location
3. **Fight Consistency**: All fights on same day must use same weather
4. **Live Fights**: Weather effects apply to fights that start after weather change
5. **Catch-up Simulation**: Historical weather must be retrievable for fight recovery

### Game Balance
1. **Betting Impact**: Weather is announced before betting closes so users can factor it in
2. **Death Rate**: Legendary weather with increased death shouldn't be too frequent
3. **Stat Limits**: Weather-modified stats should have reasonable bounds
4. **Effect Stacking**: Multiple weather effects should be carefully balanced

### User Experience
1. **Predictability**: Users should understand how weather affects their bets
2. **Notification**: Weather changes should be clearly communicated
3. **History**: Users should be able to see recent weather history
4. **Mobile**: Weather widget should work well on mobile devices

## Commentary Integration

### Weather-Specific Lines
Each announcer gets weather-appropriate commentary:
- **Chad**: "GOLLY! This [weather] is making the violence extra spicy!"
- **Dr. Mayhem**: "The atmospheric pressure is affecting their molecular cohesion!"
- **Sally**: "YES! THE [weather] FEEDS MY HUNGER FOR CHAOS!"
- **Commissioner**: "Weather patterns align with Department projections."

### Dynamic Commentary
- Weather effects mentioned in fight actions
- Special callouts when weather causes unusual events
- End-of-fight summaries include weather impact

## Future Expansions

### Potential Additions
1. **Seasonal Weather**: Different weather pools for different times of year
2. **Weather Betting**: Side bets on tomorrow's weather
3. **Weather Items**: Shop items that protect against or enhance weather effects
4. **Weather Memory**: Fighters remember and are affected by weather they've experienced
5. **Climate Change**: Long-term weather pattern shifts
6. **Weather Spirits**: Anthropomorphized weather that can be appeased or angered

### Integration Points
- Weather could affect tournament outcomes
- Special achievements for fighting in specific weather
- Weather-based fighter mutations
- Sponsor reactions to weather conditions 