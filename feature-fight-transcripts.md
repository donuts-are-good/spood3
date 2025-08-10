# Fight Transcript & AI Blog Generation Feature Specification

## Overview
Capture all live fight commentary, actions, and data into structured transcripts, then use AI to generate daily blog posts written in each announcer's distinctive voice. This creates a persistent record of the day's violence and builds engaging content for the community. The AI will have access to fighter histories and previous encounters to create richer, more contextual commentary.

## Core Mechanics

### Transcript Capture System
Every fight generates a complete transcript containing:
- All `LiveAction` objects broadcast during the fight
- Viewer count changes
- Fight metadata (fighters, stats, scheduled time, etc.)
- Final results and betting payouts
- Applied effects (blessings/curses)

### Enhanced Context System
For each fight, gather historical context:
- **Fighter Career Stats**: Complete win/loss/draw records for both fighters
- **Previous Encounters**: Full history of these two fighters facing each other
- **Recent Performance**: Last 5 fights for each fighter with outcomes
- **Mutation History**: Any mutations that have occurred to either fighter
- **Death/Resurrection Events**: If either fighter has died and been revived
- **MVP History**: If either fighter has been someone's MVP pick

### Daily Blog Generation
At the end of each day (after all fights conclude):
1. Combine all fight transcripts into a daily summary
2. **NEW**: Gather historical context for all fighters who competed
3. Feed transcripts + fighter histories to AI (Claude) with specific prompts for each announcer
4. Generate 4 different blog posts in each announcer's voice with rich historical references
5. Post to a dedicated blog section of the site

## Database Schema

### New Table: `fight_transcripts`
```sql
CREATE TABLE fight_transcripts (
    id INTEGER PRIMARY KEY,
    fight_id INTEGER NOT NULL,
    transcript_data TEXT NOT NULL, -- JSON array of all LiveActions + metadata
    fight_summary TEXT, -- Human-readable summary
    viewer_stats TEXT, -- JSON with peak viewers, total messages, etc.
    fighter_context TEXT, -- JSON with historical context for both fighters
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(fight_id) REFERENCES fights(id)
);
```

### New Table: `daily_blog_posts`
```sql
CREATE TABLE daily_blog_posts (
    id INTEGER PRIMARY KEY,
    date DATE NOT NULL,
    announcer_name TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    fight_count INTEGER NOT NULL,
    total_deaths INTEGER NOT NULL,
    credits_wagered INTEGER NOT NULL,
    featured_rivalries INTEGER NOT NULL, -- Count of rematches
    ai_model_used TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(date, announcer_name)
);
```

## Implementation Files

### New Files to Create
- `transcripts/recorder.go` - Captures and stores fight transcripts
- `transcripts/generator.go` - Processes transcripts into blog content
- `transcripts/context.go` - Gathers fighter historical context
- `ai/blog_writer.go` - AI integration for generating posts
- `templates/blog-index.html` - Blog listing page
- `templates/blog-post.html` - Individual blog post display
- `templates/blog-archive.html` - Archive by date/announcer

### Files to Modify
- `fight/engine.go` - Add transcript recording to live fight simulation
- `web/websocket.go` - Capture all broadcasted actions
- `web/server.go` - Add blog routes and handlers
- `database/repository.go` - Add transcript, blog, and fighter history methods

## Enhanced Transcript Data Structure

### Fight Transcript JSON Format with Context
```json
{
    "fight_id": 123,
    "scheduled_time": "2024-01-15T14:30:00Z",
    "fighter1": {
        "id": 45,
        "name": "Blood Thunder",
        "starting_health": 100000,
        "final_health": 25000,
        "stats": {...},
        "career_record": {"wins": 23, "losses": 7, "draws": 2},
        "recent_fights": [
            {
                "date": "2024-01-10",
                "opponent": "Lightning Fist",
                "result": "won",
                "method": "KO"
            }
        ],
        "mutations": [
            {
                "name": "Grew Third Arm",
                "date": "2024-01-08",
                "effect": "+15 Strength"
            }
        ],
        "death_history": {
            "times_died": 1,
            "last_death": "2023-12-15",
            "resurrection_method": "Fan petition"
        }
    },
    "fighter2": {
        "id": 67, 
        "name": "Chaos Bringer",
        "starting_health": 100000,
        "final_health": 0,
        "stats": {...},
        "career_record": {"wins": 18, "losses": 11, "draws": 1},
        "recent_fights": [...],
        "mutations": [],
        "death_history": {"times_died": 0}
    },
    "previous_encounters": [
        {
            "date": "2023-11-20",
            "winner": "Blood Thunder",
            "method": "Decision",
            "final_scores": [45000, 23000],
            "notable_events": ["First mutual combat between these fighters"]
        }
    ],
    "actions": [
        {
            "timestamp": "2024-01-15T14:30:15Z",
            "type": "damage",
            "action": "SKULL-FRACTURING HAMMER FIST! Blood Thunder connects for 2,847 damage!",
            "attacker": "Blood Thunder",
            "victim": "Chaos Bringer", 
            "damage": 2847,
            "announcer": "\"Screaming\" Sally Bloodworth",
            "commentary": "YES! CRUSH THEIR DREAMS INTO FINE POWDER!",
            "health1": 97000,
            "health2": 75000,
            "round": 2,
            "tick_number": 12
        }
    ],
    "viewer_stats": {
        "peak_viewers": 47,
        "total_actions": 156,
        "fight_duration_ticks": 89
    },
    "betting_info": {
        "total_wagered": 125000,
        "winning_bets": 23,
        "losing_bets": 41,
        "favorite": "Blood Thunder" // Based on betting volume
    },
    "final_result": {
        "winner_id": 45,
        "winner_name": "Blood Thunder",
        "victory_type": "KO",
        "deaths_occurred": 1,
        "revenge_achieved": false, // If this was payback for previous loss
        "series_record": "Blood Thunder leads 2-0"
    }
}
```

## Enhanced AI Blog Generation

### Context-Aware Prompts

#### Chad Puncherson (Enthusiastic with History)
```
You are Chad Puncherson, an enthusiastic sports announcer for the Department of Recreational Violence. Write a blog post about today's fights using your signature overly-positive, family-friendly enthusiasm. Use phrases like "GOLLY!", "HOLY MOLY!", and treat extreme violence as wholesome entertainment.

Pay special attention to:
- Fighter rivalries and rematches (get excited about seeing familiar faces!)
- Career milestones (celebrate big wins, sympathize with tough losses)
- Redemption stories (fighters bouncing back from deaths or losing streaks)
- Mutation developments (treat them as exciting character growth)

Here's today's fight data with full context: [TRANSCRIPT_WITH_CONTEXT]

Make the post 400-600 words, include specific fight highlights with historical references, and maintain your cheerful, naive personality throughout.
```

#### Dr. Mayhem PhD (Scientific with Analysis)
```
You are Dr. Mayhem PhD, a scientific analyst for the Department of Recreational Violence. Write an analytical blog post examining today's combat data from a pseudo-scientific perspective. Reference impossible medical/physics concepts, cite made-up studies, and analyze fighter performance clinically.

Focus your analysis on:
- Performance trends over multiple fights
- Statistical significance of win/loss patterns
- Mutation effects on combat effectiveness
- Comparative analysis between rematches
- Hypothesis about fighter development trajectories

Here's today's comprehensive data: [TRANSCRIPT_WITH_CONTEXT]

Make it 500-700 words, include statistical analysis with historical comparisons, and use scientific jargon mixed with absurd medical observations.
```

#### "Screaming" Sally Bloodworth (Intense with Vendettas)
```
You are "Screaming" Sally Bloodworth, the most violent and intense announcer for the Department. Write a blog post celebrating today's carnage with maximum intensity and bloodlust. USE CAPS FREQUENTLY, demand more violence, and treat destruction as beautiful art.

Get especially excited about:
- REVENGE MATCHES and settling old scores
- FIGHTERS COMING BACK FROM THE DEAD
- BREAKING WINNING/LOSING STREAKS WITH VIOLENCE
- MUTATION-ENHANCED BRUTALITY
- HISTORIC RIVALRIES REACHING NEW HEIGHTS

Here's today's glorious violence with full backstory: [TRANSCRIPT_WITH_CONTEXT]

Make it 400-600 words of pure aggressive enthusiasm for chaos and destruction, with references to the fighters' violent histories.
```

#### THE COMMISSIONER (Mysterious with Files)
```
You are THE COMMISSIONER, the mysterious bureaucratic overseer of the Department of Recreational Violence. Write an official report-style blog post analyzing today's activities from an administrative perspective. Use bureaucratic language, reference Department protocols, and maintain an ominous tone.

Your report should include:
- Updates on fighter performance metrics over time
- Analysis of recurring matchup outcomes for Department records
- Notes on fighter "development" (mutations, deaths, resurrections)
- Assessment of violence efficiency trends
- Cryptic references to longer-term Department objectives

Access the following classified data: [TRANSCRIPT_WITH_CONTEXT]

Make it 500-700 words in formal bureaucratic style with subtle threats and references to unknowable Department goals, incorporating historical fighter assessments.
```

## Historical Context Gathering System

### Fighter History Query
```go
type FighterContext struct {
    Fighter            database.Fighter
    CareerRecord       CareerStats
    RecentFights       []FightSummary
    Mutations          []MutationEvent
    DeathHistory       DeathRecord
    MVPStatus          MVPRecord
    NotableAchievements []Achievement
}

type PreviousEncounter struct {
    Date           time.Time
    Winner         string
    Method         string
    FinalScores    [2]int
    NotableEvents  []string
    BettingOdds    string
}

func GatherFightContext(fight database.Fight) (*FightContextData, error) {
    // Get detailed fighter histories
    fighter1Context := getFighterHistory(fight.Fighter1ID, 30) // Last 30 days
    fighter2Context := getFighterHistory(fight.Fighter2ID, 30)
    
    // Find all previous encounters between these two fighters
    previousFights := getPreviousEncounters(fight.Fighter1ID, fight.Fighter2ID)
    
    // Calculate rivalry status
    rivalryData := analyzeRivalry(previousFights)
    
    return &FightContextData{
        Fighter1:            fighter1Context,
        Fighter2:            fighter2Context,
        PreviousEncounters: previousFights,
        RivalryAnalysis:    rivalryData,
    }, nil
}
```

## Enhanced Blog Website Integration

### Blog Routes
- `/blog` - Main blog index with recent posts
- `/blog/{announcer}` - Posts filtered by specific announcer  
- `/blog/{date}` - All posts from specific date
- `/blog/post/{id}` - Individual blog post
- `/blog/archive` - Archive browsing by month/year
- `/blog/rivalries` - **NEW**: Posts featuring fighter rivalries
- `/blog/comebacks` - **NEW**: Posts about resurrection/redemption stories

### Blog UI Features
- **Announcer Avatars** - Unique icons for each personality
- **Fight Highlights** - Embedded fight result summaries with historical context
- **Rivalry Indicators** - Special badges for rematch coverage
- **Related Fights** - Links to mentioned fights and previous encounters
- **Search Function** - Search within blog content including historical references
- **RSS Feeds** - Subscribe to specific announcers
- **Fighter Tags** - Click fighter names to see all blog mentions

## Enhanced Implementation Process

### Phase 1: Enhanced Transcript Recording
```go
// In fight/engine.go - modify fight completion
func (e *Engine) CompleteFight(fight database.Fight, state *FightState) error {
    // ... existing logic ...
    
    // NEW: Gather historical context
    context, err := transcripts.GatherFightContext(fight)
    if err != nil {
        log.Printf("Failed to gather fight context: %v", err)
    }
    
    // Record transcript with context
    err = e.recorder.RecordFightWithContext(fight.ID, state, context)
    if err != nil {
        log.Printf("Failed to record fight transcript: %v", err)
    }
}
```

### Phase 2: Context-Aware Blog Generation
```go
// Enhanced daily blog generation
func GenerateDailyBlogs(date time.Time) error {
    // Get all fight transcripts for the day (now includes context)
    transcripts := repo.GetDayTranscriptsWithContext(date)
    
    // Analyze daily patterns
    dailyStats := analyzeDailyPatterns(transcripts)
    
    // Generate blog for each announcer with enhanced context
    for _, announcer := range announcers {
        prompt := buildContextualPrompt(announcer, transcripts, dailyStats)
        content := ai.GenerateBlogPost(prompt)
        
        // Extract metadata for better categorization
        metadata := extractBlogMetadata(content, transcripts)
        repo.SaveBlogPostWithMetadata(date, announcer, content, metadata)
    }
}
```

## Example Enhanced Generated Content

### Chad's Blog Post (Sample with Context)
```
GOLLY GEE! Redemption Stories and Rivalry Renewals!

Holy moly, fight fans! What an absolutely SPECTACULAR day of recreational violence we witnessed! Today wasn't just about the carnage - it was about STORIES unfolding before our very eyes!

Our headline bout featured the AMAZING rematch between Blood Thunder and Chaos Bringer! You might remember these two tangled back in November, where Blood Thunder squeaked out a decision victory. Well, Chaos Bringer came back with VENGEANCE in their heart! Unfortunately for them, Blood Thunder's recent "Grew Third Arm" mutation gave them a decisive advantage - that extra appendage delivered the knockout punch that sent Chaos Bringer to their first-ever trip to the great violence dimension in the sky!

And can we talk about Lightning McPunch's INCREDIBLE comeback story? After dying THREE TIMES this season, they returned today with a fire in their belly (and possibly literal fire, thanks to their new Photosynthetic Skin mutation)! The way they demolished Iron Jaw in Round 3 was just heartwarming - you could really see how much they'd grown as a person-fighter-entity!

The Department's violence efficiency ratings were off the charts today, with Commissioner approval ratings reaching an all-time high of "Satisfactory Plus!" That's government speak for "absolutely wonderful," folks!

-Chad Puncherson, Certified Violence Enthusiast & Historical Archivist
```

## Technical Considerations

### Enhanced Storage Requirements
- Each fight transcript with context: ~150-200KB JSON
- Daily blog posts: ~3-7KB each (longer with context)
- Fighter history cache: ~50KB per active fighter
- Estimated storage: ~1MB per day total

### AI Costs (Updated)
- 4 blog posts per day with enhanced context
- ~4000-6000 tokens per transcript + prompt (increased due to context)
- Estimated: $5-12 per day for Claude API

### Performance Optimizations
- Cache fighter histories to avoid repeated queries
- Pre-compute rivalry relationships
- Lazy-load full context only when generating blogs
- Index previous encounters for fast lookup

## Future Enhancements

### Advanced Context Features
1. **Fighter Personality Profiles** - AI learns fighter "personalities" from their combat patterns
2. **Seasonal Story Arcs** - Track longer narrative threads across tournaments
3. **Community Sentiment Tracking** - Include betting patterns and user reactions in context
4. **Cross-Tournament Analysis** - Reference events from previous seasons/tournaments
5. **Injury/Mutation Progression** - Track how mutations affect performance over time

### Advanced Blog Features
1. **Interactive Timelines** - Click through a fighter's complete history
2. **Rivalry Relationship Maps** - Visual network of who has fought whom
3. **Predictive Analysis Posts** - AI predicts future matchup outcomes based on history
4. **Memorial Posts** - Special posts when long-standing fighters finally die
5. **Anniversary Coverage** - Posts commemorating historic fights from previous years 