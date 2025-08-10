# Fighter Mutation System Feature Specification

## Overview
A dynamic system that allows fighters to undergo random or induced mutations that permanently alter their stats, appearance, and quirky characteristics. Mutations add unpredictability to the fighter roster and create emergent storylines as fighters evolve over time.

## Core Mechanics

### Mutation Triggers
1. **Post-Fight Mutation (5% chance)**
   - Occurs after any completed fight (win, loss, or draw)
   - Chance increases to 15% if fighter took >80% damage
   - Chance increases to 25% if fighter died and was revived

2. **Mutation Serum (Shop Item)**
   - Purchasable item: "Experimental Mutation Serum" - 25,000 credits
   - Guarantees mutation when used on a specific fighter
   - Can only be used on fighters currently alive
   - Cannot be used on same fighter more than once per week

3. **Environmental Mutations**
   - Certain weather conditions can trigger mutations (1% chance)
   - Legendary weather events have higher mutation rates
   - Exposure to chaos singularities, void winds, etc.

4. **Cascade Mutations (Rare)**
   - When a fighter with 3+ mutations fights another mutated fighter
   - 2% chance of "mutation resonance" affecting both fighters
   - Can create linked mutations or opposing transformations

### Mutation Categories

#### Physical Mutations (40% of mutations)
1. **Extra Limbs**
   - "Grew Third Arm": +15 Strength, fingers become "2 per hand, 3 hands"
   - "Sprouted Tail": +10 Speed, +5 Technique, toes become "N/A (has tail)"
   - "Additional Legs": +20 Speed, -10 Technique, "can no longer fit in normal clothing"

2. **Sensory Changes**
   - "Developed Echolocation": +25 Technique, -5 Speed, "sees in sound waves"
   - "Eyes Became Kaleidoscopes": +10 all stats, "perceives reality in fractals"
   - "Smell Became Time-Sensitive": Can smell the past, +15 Endurance

3. **Size Alterations**
   - "Molecular Compression": -20% all stats but gains "density multiplier x2"
   - "Gigantification": +30 Strength, -15 Speed, "requires industrial-sized clothing"
   - "Became Partially Hollow": -10 Endurance, +20 Speed, "echoes when walking"

#### Biological Mutations (35% of mutations)
4. **Blood System Changes**
   - Blood type changes to exotic variants: "Pure Mathematics", "Liquid Starlight", "Condensed Regret"
   - "Blood Became Sentient": Blood type shows as "Bob (my blood)" - gains autonomous healing
   - "Circulatory Efficiency": +25 Endurance, blood type becomes "Optimized"

5. **Metabolic Shifts**
   - "Photosynthetic Skin": +10 Endurance, turns slightly green, "powered by fluorescent lights"
   - "Hibernation Capable": Can skip fights by sleeping, gains "seasonal fighter" tag
   - "Feeds on Violence": Gains health when other fighters take damage

6. **Aging Anomalies**
   - "Temporal Displacement": Ages backwards during fights, ancestors count decreases
   - "Accelerated Healing": Regenerates between rounds, +30 Endurance
   - "Became Chronologically Unstable": Age becomes "Yes/No/Maybe"

#### Mental/Existential Mutations (20% of mutations)
7. **Consciousness Alterations**
   - "Achieved Hive Mind": Shares thoughts with all other mutated fighters
   - "Existential Enlightenment": Existential dread becomes negative number
   - "Developed Precognition": Can see 3 seconds into the future, +20 Technique

8. **Reality Perception**
   - "Sees in Additional Dimensions": Molecular density becomes "4D"
   - "Became Philosophically Dense": All attacks deal existential damage
   - "Transcended Mathematics": Stats occasionally display as equations

#### Chaos Mutations (5% of mutations - Legendary)
9. **Department Interference**
   - "Became Department Asset": Gains Commissioner oversight, all stats +10
   - "Filed Paperwork": Bureaucratic immunity to 10% of damage
   - "Appointed as Regional Violence Coordinator": Can influence other fights

10. **Reality Glitches**
    - "Duplicated Across Timelines": Sometimes fights as two separate entities
    - "Became Conceptual": Exists as an idea rather than physical being
    - "Merged with Fighter Class": Name becomes "[Original Name] the Living [Class]"

## Database Schema

### New Table: `fighter_mutations`
```sql
CREATE TABLE fighter_mutations (
    id INTEGER PRIMARY KEY,
    fighter_id INTEGER NOT NULL,
    mutation_type TEXT NOT NULL,
    mutation_name TEXT NOT NULL,
    description TEXT NOT NULL,
    stat_changes_json TEXT, -- JSON object with stat modifications
    special_properties_json TEXT, -- JSON for special abilities/quirks
    mutation_trigger TEXT NOT NULL, -- 'post_fight', 'serum', 'environmental', 'cascade'
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(fighter_id) REFERENCES fighters(id)
);
```

### Mutation Data Storage
```json
{
    "stat_changes": {
        "strength": 15,
        "speed": -5,
        "endurance": 0,
        "technique": 10,
        "fingers": "2 per hand, 3 hands",
        "blood_type": "Pure Mathematics"
    },
    "special_properties": {
        "damage_resistance": 0.1,
        "healing_rate": 1.5,
        "unique_abilities": ["can_fight_twice", "immune_to_void"]
    }
}
```

### Fighter Table Modifications
```sql
ALTER TABLE fighters ADD COLUMN mutation_count INTEGER DEFAULT 0;
ALTER TABLE fighters ADD COLUMN last_mutation_date DATE;
ALTER TABLE fighters ADD COLUMN mutation_notes TEXT; -- Human-readable mutation summary
```

## Implementation Files

### New Files to Create
- `mutations/generator.go` - Mutation generation and randomization logic
- `mutations/effects.go` - Application of mutation effects during fights
- `mutations/types.go` - Mutation type definitions and stat calculations
- `mutations/triggers.go` - Logic for when mutations occur
- `database/mutations.go` - Database operations for mutations
- `templates/mutation-display.html` - UI component for showing mutations

### Files to Modify
- `fight/engine.go` - Apply mutation effects during combat
- `fight/commentary.go` - Add mutation-specific commentary
- `database/repository.go` - Add mutation tracking methods
- `web/server.go` - Add mutation data to fighter pages
- `templates/fighter.html` - Display mutations and their effects
- `templates/fighters.html` - Show mutation indicators in fighter lists

## User Interface

### Mutation Display
- **Fighter Pages**: Dedicated mutations section with expandable details
- **Fighter Lists**: Mutation count badges (🧬 x3) next to mutated fighters
- **Fight Pages**: Mutation effects tooltips and visual indicators
- **Mutation History**: Timeline of when mutations occurred

### Mutation Visualization
- **Mutation Icons**: Unique emoji/symbols for each mutation type
- **Color Coding**: Different colors for mutation categories
- **Stat Overlays**: Modified stats highlighted differently from base stats
- **Hover Details**: Comprehensive tooltips explaining mutation effects

## Mutation Logic and Balance

### Stat Modification Rules
1. **Stat Bounds**: Mutations can push stats below 0 or above 100
2. **Cumulative Effects**: Multiple mutations stack (can create super-fighters or broken ones)
3. **Diminishing Returns**: Each mutation has slightly less impact on heavily mutated fighters
4. **Negative Mutations**: Some mutations are detrimental but come with unique abilities

### Mutation Interaction Rules
1. **Conflicting Mutations**: Some mutations cancel each other out
2. **Synergistic Mutations**: Certain combinations create bonus effects
3. **Mutation Chains**: Some mutations can only occur if prerequisite mutations exist
4. **Reversion Chance**: 1% chance per fight that a mutation reverses itself

## Game Integration

### Betting Implications
- Mutation history affects betting odds calculations
- Recent mutations create uncertainty in fighter performance
- Players can bet on whether mutations will occur
- Mutation reveals happen after betting closes to maintain fairness

### Fight Commentary Integration
- Announcers comment on mutation effects during fights
- Special callouts when mutations directly affect fight outcomes
- Dr. Mayhem provides "scientific" explanations for impossible mutations
- The Commissioner notes mutations in Department files

### Tournament Effects
- Heavily mutated fighters may get special tournament brackets
- Mutation-based achievements and recognition
- End-of-tournament mutation summaries
- Sponsors may react to fighter mutations

## Edge Cases and Considerations

### Technical Challenges
1. **Mutation Persistence**: All mutations must survive server restarts
2. **Fight Simulation**: Historical fight recreation must account for mutations at time of fight
3. **Stat Overflow**: Handle negative stats and stats >100 gracefully
4. **Display Complexity**: UI must accommodate wildly different stat ranges

### Game Balance Issues
1. **Power Creep**: Prevent mutations from making fighters too powerful over time
2. **Broken Fighters**: Some mutation combinations might break fight mechanics
3. **Mutation Fatigue**: Too many mutations might make fighters unrecognizable
4. **Reversion Mechanics**: Ways to occasionally reset overly mutated fighters

### User Experience
1. **Complexity Management**: Don't overwhelm new players with mutation details
2. **Mutation Tracking**: Help users understand how mutations affect their bets
3. **Accessibility**: Ensure mutation displays work for screen readers
4. **Mobile Optimization**: Complex mutation data must work on small screens

## Special Mutation Events

### Mutation Storms (Weekly Events)
- Rare events where mutation rates increase 10x for 24 hours
- Announced in advance to create excitement
- Special mutation types only available during storms
- Community-wide effects (all fighters slightly affected)

### Evolutionary Pressure
- Fighters that consistently lose develop survival mutations
- Winners develop dominance mutations
- Environmental pressures from weather create themed mutations
- Long-term trends shape the fighter population

## Future Expansions

### Advanced Features
1. **Mutation Breeding**: Custom fighters inherit mutation tendencies
2. **Mutation Markets**: Trade mutation serums between players
3. **Research Lab**: Spend credits to study and predict mutations
4. **Mutation Artifacts**: Special items that influence mutation types
5. **Department Experiments**: The Commissioner occasionally forces mutations

### Integration Opportunities
- Weather-specific mutations
- Mutation-based custom fighter creation
- Crossover mutations between user-created fighters
- Mutation-dependent shop items and abilities
- Tournament formats based on mutation levels 