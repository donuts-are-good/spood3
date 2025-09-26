package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"spoodblort/database"
	"spoodblort/fight"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

// FightBroadcaster manages live fight broadcasting
type FightBroadcaster struct {
	repo       *database.Repository
	engine     FightEngine                      // Interface to log actions
	clients    map[int]map[*websocket.Conn]bool // fightID -> connections
	clientsMux sync.RWMutex
	broadcast  map[int]chan fight.LiveAction // fightID -> broadcast channel

	// In-memory clap rate limiting
	userClaps map[string][]time.Time // userID_fightID -> timestamps
	clapsMux  sync.RWMutex

	// Clap totals per round
	roundClapTotals map[string]map[int]int // "fightID_round" -> userID -> count
	roundTotalsMux  sync.RWMutex
}

// FightEngine interface for logging actions
type FightEngine interface {
	LogAction(fightID int, text string)
}

type ClapMessage struct {
	Type        string `json:"type"`
	FighterID   int    `json:"fighter_id"`
	FighterName string `json:"fighter_name"`
	Round       int    `json:"round"`
}

// NewFightBroadcaster creates a new fight broadcasting system
func NewFightBroadcaster(repo *database.Repository) *FightBroadcaster {
	return &FightBroadcaster{
		repo:            repo,
		clients:         make(map[int]map[*websocket.Conn]bool),
		broadcast:       make(map[int]chan fight.LiveAction),
		userClaps:       make(map[string][]time.Time),
		roundClapTotals: make(map[string]map[int]int),
	}
}

// SetEngine sets the fight engine for logging purposes
func (fb *FightBroadcaster) SetEngine(engine FightEngine) {
	fb.engine = engine
}

// CanUserClap checks if user can clap (rate limiting)
func (fb *FightBroadcaster) CanUserClap(userID, fightID int) bool {
	fb.clapsMux.Lock()
	defer fb.clapsMux.Unlock()

	key := fmt.Sprintf("%d_%d", userID, fightID)
	now := time.Now()

	// Clean up old claps (older than 1 second)
	claps := fb.userClaps[key]
	validClaps := []time.Time{}
	for _, clapTime := range claps {
		if now.Sub(clapTime) < time.Second {
			validClaps = append(validClaps, clapTime)
		}
	}
	fb.userClaps[key] = validClaps

	// Check if under limit (10 per second)
	return len(validClaps) < 10
}

// RecordClap records a clap event for rate limiting
func (fb *FightBroadcaster) RecordClap(userID, fightID int) {
	fb.clapsMux.Lock()
	defer fb.clapsMux.Unlock()

	key := fmt.Sprintf("%d_%d", userID, fightID)
	fb.userClaps[key] = append(fb.userClaps[key], time.Now())
}

// ProcessClapMessage handles incoming clap events
func (fb *FightBroadcaster) ProcessClapMessage(userID, fightID int, clap ClapMessage) error {
	// Check if round is divisible by 5 (clap rounds)
	if clap.Round%5 != 0 {
		return fmt.Errorf("clapping not enabled for round %d", clap.Round)
	}

	// Check rate limit
	if !fb.CanUserClap(userID, fightID) {
		return fmt.Errorf("rate limit exceeded")
	}

	// Record the clap
	fb.RecordClap(userID, fightID)

	// Track clap totals for this round
	fb.roundTotalsMux.Lock()
	roundKey := fmt.Sprintf("%d_%d", fightID, clap.Round)
	if fb.roundClapTotals[roundKey] == nil {
		fb.roundClapTotals[roundKey] = make(map[int]int)
	}
	fb.roundClapTotals[roundKey][userID]++
	fb.roundTotalsMux.Unlock()

	// Get user info for display
	user, err := fb.repo.GetUser(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Create clap action for broadcasting
	displayName := user.Username
	if user.CustomUsername != "" {
		displayName = user.CustomUsername
	}

	clapAction := fight.LiveAction{
		Type:       "clap",
		Action:     fmt.Sprintf("%s cheered for %s! 👏👏 +20 health", displayName, clap.FighterName),
		Announcer:  "",
		Commentary: "",
		Round:      clap.Round,
	}

	// Broadcast to all viewers
	fb.BroadcastAction(fightID, clapAction)

	log.Printf("User %s clapped for fighter %s in fight %d, round %d", displayName, clap.FighterName, fightID, clap.Round)
	return nil
}

// BroadcastRoundClapSummary sends a summary of claps when a clapping round ends
func (fb *FightBroadcaster) BroadcastRoundClapSummary(fightID, round int) {
	// Only broadcast summaries for rounds that just ended clapping (were divisible by 5)
	if round%5 != 1 {
		return
	}

	previousRound := round - 1
	if previousRound%5 != 0 {
		return
	}

	fb.roundTotalsMux.RLock()
	roundKey := fmt.Sprintf("%d_%d", fightID, previousRound)
	clapTotals := fb.roundClapTotals[roundKey]
	fb.roundTotalsMux.RUnlock()

	if len(clapTotals) == 0 {
		return
	}

	// Sort users by clap count (highest first)
	type userClaps struct {
		displayName string
		count       int
	}

	var sortedClaps []userClaps
	for userID, count := range clapTotals {
		user, err := fb.repo.GetUser(userID)
		if err != nil {
			continue
		}

		displayName := user.Username
		if user.CustomUsername != "" {
			displayName = user.CustomUsername
		}

		sortedClaps = append(sortedClaps, userClaps{
			displayName: displayName,
			count:       count,
		})
	}

	// Sort by count (highest first)
	for i := 0; i < len(sortedClaps)-1; i++ {
		for j := i + 1; j < len(sortedClaps); j++ {
			if sortedClaps[j].count > sortedClaps[i].count {
				sortedClaps[i], sortedClaps[j] = sortedClaps[j], sortedClaps[i]
			}
		}
	}

	// Create summary message
	var summaryParts []string
	for _, uc := range sortedClaps {
		summaryParts = append(summaryParts, fmt.Sprintf("%s cheered %s times", uc.displayName, addCommas(uc.count)))
	}

	if len(summaryParts) > 0 {
		summaryAction := fight.LiveAction{
			Type:       "clap_summary",
			Action:     fmt.Sprintf("🎉 ROUND %d CLAP TOTALS: %s!", previousRound, joinWithCommasAnd(summaryParts)),
			Announcer:  "THE COMMISSIONER",
			Commentary: "The Department has recorded these displays of crowd enthusiasm for statistical analysis.",
			Round:      round,
		}

		fb.BroadcastAction(fightID, summaryAction)

		// Log the clap summary to fight log
		if fb.engine != nil {
			fb.engine.LogAction(fightID, summaryAction.Action)
			if summaryAction.Commentary != "" {
				fb.engine.LogAction(fightID, fmt.Sprintf("%s: \"%s\"", summaryAction.Announcer, summaryAction.Commentary))
			}
		}
	}

	// Clean up the round data to save memory
	fb.roundTotalsMux.Lock()
	delete(fb.roundClapTotals, roundKey)
	fb.roundTotalsMux.Unlock()
}

// Helper function to add commas to numbers
func addCommas(n int) string {
	str := fmt.Sprintf("%d", n)
	if len(str) <= 3 {
		return str
	}

	result := ""
	for i, digit := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result += ","
		}
		result += string(digit)
	}
	return result
}

// Helper function to join strings with commas and "and"
func joinWithCommasAnd(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	if len(parts) == 2 {
		return parts[0] + " and " + parts[1]
	}

	result := ""
	for i, part := range parts {
		if i == len(parts)-1 {
			result += "and " + part
		} else if i == len(parts)-2 {
			result += part + " "
		} else {
			result += part + ", "
		}
	}
	return result
}

// HandleWebSocket handles individual WebSocket connections for a fight
func (fb *FightBroadcaster) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fightID, err := strconv.Atoi(vars["id"])
	if err != nil {
		log.Printf("WebSocket: Invalid fight ID: %v", err)
		http.Error(w, "Invalid fight ID", http.StatusBadRequest)
		return
	}

	log.Printf("WebSocket: Attempting to upgrade connection for fight %d from %s", fightID, r.RemoteAddr)

	// Upgrade connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket: Upgrade failed for fight %d: %v", fightID, err)
		return
	}
	defer conn.Close()

	log.Printf("WebSocket: Successfully upgraded connection for fight %d", fightID)

	// Get user ID from session if authenticated
	userID := 0
	if user := GetUserFromContext(r.Context()); user != nil {
		userID = user.ID
	}

	// Register client
	fb.clientsMux.Lock()
	if fb.clients[fightID] == nil {
		fb.clients[fightID] = make(map[*websocket.Conn]bool)
	}
	fb.clients[fightID][conn] = true
	viewerCount := len(fb.clients[fightID])
	fb.clientsMux.Unlock()

	log.Printf("WebSocket: New viewer connected to fight %d (total: %d, userID: %d)", fightID, viewerCount, userID)

	// Broadcast updated viewer count to all viewers
	fb.BroadcastViewerCount(fightID)

	// Send initial fight state
	err = fb.sendInitialState(conn, fightID)
	if err != nil {
		log.Printf("WebSocket: Failed to send initial state for fight %d: %v", fightID, err)
		return
	}

	log.Printf("WebSocket: Sent initial state for fight %d", fightID)

	// Handle client disconnection
	defer func() {
		fb.clientsMux.Lock()
		delete(fb.clients[fightID], conn)
		if len(fb.clients[fightID]) == 0 {
			delete(fb.clients, fightID)
		}
		fb.clientsMux.Unlock()

		log.Printf("WebSocket: Viewer disconnected from fight %d", fightID)
		// Broadcast updated viewer count after disconnection
		fb.BroadcastViewerCount(fightID)
	}()

	// Handle client messages in a separate function with user context
	fb.handleClientMessages(conn, fightID, userID)
}

// handleClientMessages processes WebSocket messages for a specific connection
func (fb *FightBroadcaster) handleClientMessages(conn *websocket.Conn, fightID, userID int) {
	// Keep connection alive and handle client messages
	for {
		// Read message (could be ping/pong or clap events)
		_, messageBytes, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket: Unexpected close error for fight %d: %v", fightID, err)
			} else {
				log.Printf("WebSocket: Connection closed normally for fight %d: %v", fightID, err)
			}
			break
		}

		// Try to parse as JSON message
		var message map[string]interface{}
		if err := json.Unmarshal(messageBytes, &message); err != nil {
			// Not JSON, probably a ping/pong - ignore
			continue
		}

		// Handle different message types
		messageType, ok := message["type"].(string)
		if !ok {
			continue
		}

		switch messageType {
		case "clap":
			if userID == 0 {
				log.Printf("WebSocket: Clap from unauthenticated user")
				continue
			}

			// Parse clap message
			var clapMsg ClapMessage
			if err := json.Unmarshal(messageBytes, &clapMsg); err != nil {
				log.Printf("WebSocket: Invalid clap message: %v", err)
				continue
			}

			// Process the clap in a goroutine to avoid blocking other messages
			go func() {
				if err := fb.ProcessClapMessage(userID, fightID, clapMsg); err != nil {
					log.Printf("WebSocket: Clap processing failed: %v", err)
				}
			}()
		}
	}
}

// BroadcastAction sends a live action to all viewers of a fight
func (fb *FightBroadcaster) BroadcastAction(fightID int, action fight.LiveAction) {
	fb.clientsMux.RLock()
	clients := fb.clients[fightID]
	fb.clientsMux.RUnlock()

	if len(clients) == 0 {
		return // No viewers
	}

	message, err := json.Marshal(map[string]interface{}{
		"type":   "action",
		"action": action,
	})
	if err != nil {
		log.Printf("Failed to marshal action: %v", err)
		return
	}

	// Send to all connected clients
	for conn := range clients {
		err := conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("Failed to send message: %v", err)
			// Remove broken connection
			fb.clientsMux.Lock()
			delete(fb.clients[fightID], conn)
			fb.clientsMux.Unlock()
			conn.Close()
		}
	}

	log.Printf("Broadcasted action to %d viewers of fight %d", len(clients), fightID)
}

// BroadcastViewerCount sends updated viewer count to all clients
func (fb *FightBroadcaster) BroadcastViewerCount(fightID int) {
	fb.clientsMux.RLock()
	clients := fb.clients[fightID]
	viewerCount := len(clients)
	fb.clientsMux.RUnlock()

	if viewerCount == 0 {
		return
	}

	message, err := json.Marshal(map[string]interface{}{
		"type":         "viewer_count",
		"viewer_count": viewerCount,
	})
	if err != nil {
		return
	}

	for conn := range clients {
		conn.WriteMessage(websocket.TextMessage, message)
	}
}

// sendInitialState sends the current fight state to a new viewer
func (fb *FightBroadcaster) sendInitialState(conn *websocket.Conn, fightID int) error {
	// Get fight details
	fight, err := fb.repo.GetFight(fightID)
	if err != nil {
		return err
	}

	// Get fighters
	fighter1, err := fb.repo.GetFighter(fight.Fighter1ID)
	if err != nil {
		return err
	}

	fighter2, err := fb.repo.GetFighter(fight.Fighter2ID)
	if err != nil {
		return err
	}

	// Calculate actual starting health with effects
	var fighter1Health, fighter2Health int

	// Determine which date to use for effects based on fight status
	var effectDate time.Time
	if fight.Status == "active" {
		// For active fights, use today's effects (live viewing) in Central Time
		centralTime, _ := time.LoadLocation("America/Chicago")
		effectDate = time.Now().In(centralTime)
	} else {
		// For scheduled/completed/voided fights, use the fight's scheduled date (historical viewing)
		effectDate = fight.ScheduledTime
	}

	// Get day bounds for the effect date
	startDate := time.Date(effectDate.Year(), effectDate.Month(), effectDate.Day(), 0, 0, 0, 0, effectDate.Location())
	endDate := startDate.Add(24 * time.Hour)

	// Calculate health with applied effects from the specific date
	fighter1Effects, _ := fb.repo.GetAppliedEffectsForDate("fighter", fight.Fighter1ID, startDate, endDate)
	fighter2Effects, _ := fb.repo.GetAppliedEffectsForDate("fighter", fight.Fighter2ID, startDate, endDate)

	// Base health
	fighter1Health = 100000 // STARTING_HEALTH constant
	fighter2Health = 100000

	// Apply effects
	for _, effect := range fighter1Effects {
		switch effect.EffectType {
		case "fighter_blessing":
			fighter1Health += effect.EffectValue
		case "fighter_curse":
			fighter1Health -= effect.EffectValue
		}
	}

	for _, effect := range fighter2Effects {
		switch effect.EffectType {
		case "fighter_blessing":
			fighter2Health += effect.EffectValue
		case "fighter_curse":
			fighter2Health -= effect.EffectValue
		}
	}

	// Ensure health never goes below 1
	if fighter1Health < 1 {
		fighter1Health = 1
	}
	if fighter2Health < 1 {
		fighter2Health = 1
	}

	// Determine current state based on fight status
	var state map[string]interface{}

	switch fight.Status {
	case "scheduled":
		state = map[string]interface{}{
			"type":     "initial",
			"status":   "scheduled",
			"fighter1": fighter1,
			"fighter2": fighter2,
			"fight":    fight,
			"message":  "Fight begins soon! Violence is imminent!",
			"health1":  fighter1Health,
			"health2":  fighter2Health,
			"round":    0,
		}

	case "active":
		// For active fights, we'd need to calculate current state
		// For now, send starting health with effects
		state = map[string]interface{}{
			"type":     "initial",
			"status":   "active",
			"fighter1": fighter1,
			"fighter2": fighter2,
			"fight":    fight,
			"message":  "🔴 LIVE VIOLENCE IN PROGRESS! 🔴",
			"health1":  fighter1Health, // Now uses calculated health
			"health2":  fighter2Health, // Now uses calculated health
			"round":    1,
		}

	case "completed":
		// For completed fights, show the actual final results
		finalHealth1 := 0
		finalHealth2 := 0

		if fight.FinalScore1.Valid {
			finalHealth1 = int(fight.FinalScore1.Int64)
		}
		if fight.FinalScore2.Valid {
			finalHealth2 = int(fight.FinalScore2.Int64)
		}

		// Determine winner message
		winnerMessage := "Violence has concluded. The chaos gods are satisfied."
		if fight.WinnerID.Valid {
			if int(fight.WinnerID.Int64) == fight.Fighter1ID {
				winnerMessage = fmt.Sprintf("🏆 %s emerged victorious from the carnage! 🏆", fight.Fighter1Name)
			} else if int(fight.WinnerID.Int64) == fight.Fighter2ID {
				winnerMessage = fmt.Sprintf("🏆 %s emerged victorious from the carnage! 🏆", fight.Fighter2Name)
			}
		} else {
			winnerMessage = "💀 BOTH FIGHTERS DIED IN MUTUAL DESTRUCTION! 💀"
		}

		state = map[string]interface{}{
			"type":     "initial",
			"status":   "completed",
			"fighter1": fighter1,
			"fighter2": fighter2,
			"fight":    fight,
			"message":  winnerMessage,
			"health1":  finalHealth1,
			"health2":  finalHealth2,
			"round":    "FINAL",
		}

	case "voided":
		state = map[string]interface{}{
			"type":     "initial",
			"status":   "voided",
			"fighter1": fighter1,
			"fighter2": fighter2,
			"fight":    fight,
			"message":  "⚰️ This violence was absorbed by the chaos void. ⚰️",
			"health1":  fighter1Health,
			"health2":  fighter2Health,
			"round":    "VOID",
		}

	default:
		state = map[string]interface{}{
			"type":    "initial",
			"status":  fight.Status,
			"message": "Fight status unknown. The void consumes all.",
		}
	}

	message, err := json.Marshal(state)
	if err != nil {
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, message)
}

// GetViewerCount returns the number of viewers for a fight
func (fb *FightBroadcaster) GetViewerCount(fightID int) int {
	fb.clientsMux.RLock()
	defer fb.clientsMux.RUnlock()
	return len(fb.clients[fightID])
}
