package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"spoodblort/database"
	"strings"
	"time"
)

type EventsManager struct {
	botToken string
	guildID  string
	repo     *database.Repository
}

type DiscordEvent struct {
	Name               string     `json:"name"`
	Description        string     `json:"description,omitempty"`
	ScheduledStartTime string     `json:"scheduled_start_time"`
	ScheduledEndTime   string     `json:"scheduled_end_time"`
	PrivacyLevel       int        `json:"privacy_level"` // 2 = GUILD_ONLY
	EntityType         int        `json:"entity_type"`   // 3 = EXTERNAL
	EntityMetadata     EntityMeta `json:"entity_metadata,omitempty"`
}

type EntityMeta struct {
	Location string `json:"location,omitempty"`
}

type DiscordEventResponse struct {
	ID                 string     `json:"id"`
	Name               string     `json:"name"`
	ScheduledStartTime string     `json:"scheduled_start_time"`
	ScheduledEndTime   string     `json:"scheduled_end_time"`
	Status             int        `json:"status"` // 1 = SCHEDULED, 2 = ACTIVE, 3 = COMPLETED, 4 = CANCELLED
	EntityMetadata     EntityMeta `json:"entity_metadata,omitempty"`
}

func NewEventsManager(repo *database.Repository) *EventsManager {
	return &EventsManager{
		botToken: os.Getenv("DISCORD_BOT_TOKEN"),
		guildID:  os.Getenv("DISCORD_GUILD_ID"),
		repo:     repo,
	}
}

// SyncFightEvents creates Discord Events for today's fights
func (em *EventsManager) SyncFightEvents(fights []database.Fight, serverBaseURL string) error {
	if em.botToken == "" || em.guildID == "" {
		log.Printf("Discord bot token or guild ID not configured, skipping events sync")
		return nil
	}

	// Get existing events to avoid duplicates
	existingEvents, err := em.getGuildEvents()
	if err != nil {
		log.Printf("Failed to get existing Discord events: %v", err)
		// Continue anyway - we'll just create new ones
	}

	log.Printf("Found %d existing Discord events", len(existingEvents))

	// Cancel old events that are no longer scheduled
	// DISABLED: Let Discord handle expired events automatically
	// err = em.cancelOutdatedEvents(existingEvents, fights)
	// if err != nil {
	//	log.Printf("Failed to cancel outdated events: %v", err)
	// }

	// Create events for today's fights (with rate limiting)
	eventsCreated := 0
	for _, fight := range fights {
		if fight.Status != "scheduled" {
			continue // Only create events for scheduled fights
		}

		// Check if event already exists for this fight
		if em.eventExistsForFight(existingEvents, fight) {
			continue
		}

		// Rate limiting: wait 2 seconds between requests to avoid 429s
		if eventsCreated > 0 {
			time.Sleep(2 * time.Second)
		}

		err := em.createFightEvent(fight, serverBaseURL)
		if err != nil {
			log.Printf("Failed to create Discord event for fight %d: %v", fight.ID, err)
			continue
		}

		eventsCreated++
		log.Printf("Created Discord event for fight: %s vs %s", fight.Fighter1Name, fight.Fighter2Name)
	}

	if eventsCreated > 0 {
		log.Printf("Successfully created %d Discord events", eventsCreated)
	}

	return nil
}

// createFightEvent creates a single Discord Event for a fight
func (em *EventsManager) createFightEvent(fight database.Fight, serverBaseURL string) error {
	endTime := fight.ScheduledTime.Add(30 * time.Minute) // Fights last 30 minutes

	// Get tournament info for the description
	tournament, err := em.repo.GetTournament(fight.TournamentID)
	var tournamentInfo string
	if err != nil {
		log.Printf("Failed to get tournament info for event: %v", err)
		tournamentInfo = "Tournament info unavailable"
	} else {
		tournamentInfo = fmt.Sprintf("**%s**\nSponsored by %s", tournament.Name, tournament.Sponsor)
	}

	event := DiscordEvent{
		Name:               fmt.Sprintf("🥊 %s vs %s", fight.Fighter1Name, fight.Fighter2Name),
		Description:        fmt.Sprintf("%s\n\n🎟️ Place your bets and witness the chaos!\n\n🔗 [Watch Live](%s/watch/%d)", tournamentInfo, serverBaseURL, fight.ID),
		ScheduledStartTime: fight.ScheduledTime.Format(time.RFC3339),
		ScheduledEndTime:   endTime.Format(time.RFC3339),
		PrivacyLevel:       2, // GUILD_ONLY
		EntityType:         3, // EXTERNAL
		EntityMetadata: EntityMeta{
			Location: fmt.Sprintf("%s/watch/%d", serverBaseURL, fight.ID),
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	url := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/scheduled-events", em.guildID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bot "+em.botToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("Discord API returned status %d", resp.StatusCode)
	}

	return nil
}

// getGuildEvents retrieves all scheduled events for the guild
func (em *EventsManager) getGuildEvents() ([]DiscordEventResponse, error) {
	url := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/scheduled-events", em.guildID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bot "+em.botToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Discord API returned status %d", resp.StatusCode)
	}

	var events []DiscordEventResponse
	err = json.NewDecoder(resp.Body).Decode(&events)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return events, nil
}

// eventExistsForFight checks if a Discord event already exists for this fight
func (em *EventsManager) eventExistsForFight(events []DiscordEventResponse, fight database.Fight) bool {
	expectedURL := fmt.Sprintf("/watch/%d", fight.ID)

	log.Printf("🔍 Checking for fight %d (URL: %s)", fight.ID, expectedURL)

	// Check if any existing event has our fight URL
	for _, event := range events {
		// Skip non-fight events
		if len(event.Name) < 4 || event.Name[0:4] != "🥊" {
			continue
		}

		// Check if this event's description contains our fight URL
		if event.EntityMetadata.Location != "" &&
			(event.EntityMetadata.Location == fmt.Sprintf("https://spoodblort.com/watch/%d", fight.ID) ||
				event.EntityMetadata.Location == fmt.Sprintf("http://localhost:8080/watch/%d", fight.ID) ||
				strings.Contains(event.EntityMetadata.Location, expectedURL)) {
			log.Printf("🚫 FOUND: Event already exists for fight %d", fight.ID)
			return true
		}
	}

	log.Printf("✅ No event found for fight %d - will create", fight.ID)
	return false
}

// cancelEvent cancels a Discord event by ID
func (em *EventsManager) cancelEvent(eventID string) error {
	url := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/scheduled-events/%s", em.guildID, eventID)

	// To cancel an event, we PATCH it with status 4 (CANCELLED)
	cancelData := map[string]interface{}{
		"status": 4,
	}

	data, err := json.Marshal(cancelData)
	if err != nil {
		return fmt.Errorf("failed to marshal cancel data: %w", err)
	}

	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bot "+em.botToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Discord API returned status %d", resp.StatusCode)
	}

	return nil
}

// UpdateEventAsCompleted updates a Discord event to mark it as completed at the current time
func (em *EventsManager) UpdateEventAsCompleted(fight database.Fight) error {
	if em.botToken == "" || em.guildID == "" {
		return nil // Discord not configured
	}

	// Get existing events to find the one for this fight
	events, err := em.getGuildEvents()
	if err != nil {
		return fmt.Errorf("failed to get guild events: %w", err)
	}

	// Find the event for this fight
	var eventID string
	expectedName := fmt.Sprintf("🥊 %s vs %s", fight.Fighter1Name, fight.Fighter2Name)

	for _, event := range events {
		if event.Name == expectedName {
			// Check if the event time matches (within reasonable range)
			eventTime, err := time.Parse(time.RFC3339, event.ScheduledStartTime)
			if err != nil {
				continue
			}

			// If event time is within 30 minutes of fight time, it's probably the right one
			timeDiff := eventTime.Sub(fight.ScheduledTime)
			if timeDiff >= -30*time.Minute && timeDiff <= 30*time.Minute {
				eventID = event.ID
				break
			}
		}
	}

	if eventID == "" {
		log.Printf("No Discord event found for completed fight: %s vs %s", fight.Fighter1Name, fight.Fighter2Name)
		return nil // Not an error, just log it
	}

	// Update the event to end now
	now := time.Now()
	updateData := map[string]interface{}{
		"scheduled_end_time": now.Format(time.RFC3339),
		"status":             3, // COMPLETED
	}

	data, err := json.Marshal(updateData)
	if err != nil {
		return fmt.Errorf("failed to marshal update data: %w", err)
	}

	url := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/scheduled-events/%s", em.guildID, eventID)
	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bot "+em.botToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Discord API returned status %d", resp.StatusCode)
	}

	log.Printf("Updated Discord event for completed fight: %s vs %s", fight.Fighter1Name, fight.Fighter2Name)
	return nil
}

// TEMPORARY: ClearAllFightEvents cancels all Discord events that start with 🥊
// This is a cleanup function to remove any Sunday events that were incorrectly created
// Only runs on Sundays
func (em *EventsManager) ClearAllFightEvents() error {
	// Only run on Sundays
	if time.Now().Weekday() != time.Sunday {
		return nil
	}

	if em.botToken == "" || em.guildID == "" {
		return fmt.Errorf("Discord bot token or guild ID not configured")
	}

	// Get all existing events
	events, err := em.getGuildEvents()
	if err != nil {
		return fmt.Errorf("failed to get guild events: %w", err)
	}

	eventsDeleted := 0
	for _, event := range events {
		// Only process fight events (ones that start with 🥊)
		if len(event.Name) >= 4 && event.Name[0:4] == "🥊" {
			// Rate limiting: wait 1 second between requests
			if eventsDeleted > 0 {
				time.Sleep(1 * time.Second)
			}

			err := em.cancelEvent(event.ID)
			if err != nil {
				log.Printf("Failed to cancel event %s: %v", event.Name, err)
				continue
			}

			log.Printf("Sunday cleanup: Cancelled Discord event: %s", event.Name)
			eventsDeleted++
		}
	}

	if eventsDeleted > 0 {
		log.Printf("Sunday cleanup: Successfully cancelled %d fight events", eventsDeleted)
	}
	return nil
}
