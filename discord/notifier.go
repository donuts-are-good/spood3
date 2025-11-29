package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"spoodblort/database"
)

// FightState represents the final state of a completed fight
// This mirrors the FightState from fight/engine.go to avoid import cycles
type FightState struct {
	Fighter1Health int
	Fighter2Health int
	TickNumber     int
	LastDamage1    int
	LastDamage2    int
	CurrentRound   int
	IsComplete     bool
	WinnerID       int
	DeathOccurred  bool
}

type Notifier struct {
	repo             *database.Repository
	botToken         string
	channelID        string
	actionChannelID  string
	generalChannelID string
	webhookURL       string
	serverBaseURL    string
}

type DiscordEmbed struct {
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Color       int                    `json:"color"`
	Timestamp   string                 `json:"timestamp"`
	URL         string                 `json:"url,omitempty"`
	Fields      []DiscordEmbedField    `json:"fields"`
	Footer      *DiscordEmbedFooter    `json:"footer,omitempty"`
	Thumbnail   *DiscordEmbedThumbnail `json:"thumbnail,omitempty"`
}

type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type DiscordEmbedFooter struct {
	Text string `json:"text"`
}

type DiscordEmbedThumbnail struct {
	URL string `json:"url"`
}

type DiscordMessage struct {
	Content string         `json:"content,omitempty"`
	Embeds  []DiscordEmbed `json:"embeds"`
}

func NewNotifier(repo *database.Repository) *Notifier {
	return &Notifier{
		repo:             repo,
		botToken:         os.Getenv("DISCORD_BOT_TOKEN"),
		channelID:        os.Getenv("DISCORD_CHANNEL_ID"),
		actionChannelID:  "1419508683171168296", // hard-coded action channel ID
		generalChannelID: "1398829103615971380", // general chat channel ID
		webhookURL:       os.Getenv("DISCORD_WEBHOOK_URL"),
		serverBaseURL:    getServerBaseURL(),
	}
}

// getServerBaseURL determines the server's base URL
func getServerBaseURL() string {
	if url := os.Getenv("SERVER_BASE_URL"); url != "" {
		return url
	}

	// Fallback to localhost in development
	if os.Getenv("ENVIRONMENT") != "production" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		return fmt.Sprintf("http://localhost:%s", port)
	}

	return "https://your-domain.com" // You'll need to set this
}

// NotifyFightResult sends a fight result card to Discord
func (n *Notifier) NotifyFightResult(fightData database.Fight, state *FightState, fighter1, fighter2 database.Fighter) error {
	if n.botToken == "" && n.webhookURL == "" {
		log.Printf("No Discord bot token or webhook URL configured, skipping notification")
		return nil
	}

	// Get applied effects for both fighters, limited to the fight's day (Central Time),
	// matching how the website displays them
	var fighter1Effects, fighter2Effects []database.AppliedEffect
	{
		// Determine date window based on fight status
		// For completed fights (which this notifier handles), use the fight's scheduled date
		effectDate := fightData.ScheduledTime
		// Compute day bounds in the same timezone used in web layer
		centralTime, _ := time.LoadLocation("America/Chicago")
		startDate := time.Date(effectDate.In(centralTime).Year(), effectDate.In(centralTime).Month(), effectDate.In(centralTime).Day(), 0, 0, 0, 0, centralTime)
		endDate := startDate.Add(24 * time.Hour)

		fighter1Effects, _ = n.repo.GetAppliedEffectsForDate("fighter", fightData.Fighter1ID, startDate, endDate)
		fighter2Effects, _ = n.repo.GetAppliedEffectsForDate("fighter", fightData.Fighter2ID, startDate, endDate)
	}

	// Get betting information
	allBets, _ := n.repo.GetAllBetsOnFight(fightData.ID)

	// Create the embed
	embed := n.createFightResultEmbed(fightData, state, fighter1, fighter2, fighter1Effects, fighter2Effects, allBets)

	// Send via webhook if available (preferred), otherwise use bot
	var sendErr error
	if n.webhookURL != "" {
		sendErr = n.sendViaWebhook(embed)
	} else if n.channelID != "" {
		sendErr = n.sendViaBot(embed)
	} else {
		return fmt.Errorf("no Discord channel configured")
	}

	// Also post a simple death notice to general chat if a death occurred
	if state.DeathOccurred && n.botToken != "" && n.generalChannelID != "" {
		var killed, killer string
		if state.WinnerID == fighter1.ID {
			killer = fighter1.Name
			killed = fighter2.Name
		} else {
			killer = fighter2.Name
			killed = fighter1.Name
		}
		content := fmt.Sprintf("%s has been killed in battle by %s.", killed, killer)
		if err := n.sendTextViaBot(n.generalChannelID, content); err != nil {
			log.Printf("failed to send general death notice: %v", err)
		}
	}

	return sendErr
}

// createFightResultEmbed builds the Discord embed for fight results
func (n *Notifier) createFightResultEmbed(fightData database.Fight, state *FightState, fighter1, fighter2 database.Fighter, fighter1Effects, fighter2Effects []database.AppliedEffect, bets []database.BetWithUser) DiscordEmbed {
	// Determine fight outcome
	var title, description string
	var color int

	if state.DeathOccurred {
		title = "💀 DEATH IN THE ARENA! 💀"
		color = 0x8B0000 // Dark red
		if state.WinnerID == fighter1.ID {
			description = fmt.Sprintf("**%s** has KILLED **%s** in brutal combat!", fighter1.Name, fighter2.Name)
		} else {
			description = fmt.Sprintf("**%s** has KILLED **%s** in brutal combat!", fighter2.Name, fighter1.Name)
		}
	} else if state.WinnerID != 0 {
		title = "🏆 VIOLENCE CONCLUDED! 🏆"
		color = 0xFFD700 // Gold
		if state.WinnerID == fighter1.ID {
			description = fmt.Sprintf("**%s** emerged victorious over **%s**!", fighter1.Name, fighter2.Name)
		} else {
			description = fmt.Sprintf("**%s** emerged victorious over **%s**!", fighter2.Name, fighter1.Name)
		}
	} else {
		title = "🤝 MUTUAL DESTRUCTION! 🤝"
		color = 0x808080 // Gray
		description = fmt.Sprintf("**%s** and **%s** fought to a draw!", fighter1.Name, fighter2.Name)
	}

	// Build fields
	var fields []DiscordEmbedField

	// Fight stats
	fields = append(fields, DiscordEmbedField{
		Name:   "Final Health",
		Value:  fmt.Sprintf("%s: **%s HP**\n%s: **%s HP**", fighter1.Name, formatNumber(state.Fighter1Health), fighter2.Name, formatNumber(state.Fighter2Health)),
		Inline: true,
	})

	// Fighter records
	fields = append(fields, DiscordEmbedField{
		Name:   "Fighter Records",
		Value:  fmt.Sprintf("%s: **%d-%d-%d**\n%s: **%d-%d-%d**", fighter1.Name, fighter1.Wins, fighter1.Losses, fighter1.Draws, fighter2.Name, fighter2.Wins, fighter2.Losses, fighter2.Draws),
		Inline: true,
	})

	// Applied effects summary
	effectsSummary := n.buildEffectsSummary(fighter1.Name, fighter1Effects, fighter2.Name, fighter2Effects)
	if effectsSummary != "" {
		fields = append(fields, DiscordEmbedField{
			Name:   "⚡ Effects Applied",
			Value:  effectsSummary,
			Inline: false,
		})
	}

	// Betting summary
	if len(bets) > 0 {
		bettingSummary := n.buildBettingSummary(bets, state.WinnerID)
		fields = append(fields, DiscordEmbedField{
			Name:   "💰 Betting Results",
			Value:  bettingSummary,
			Inline: false,
		})
	}

	// Build embed
	embed := DiscordEmbed{
		Title:       title,
		Description: description,
		Color:       color,
		Timestamp:   time.Now().Format(time.RFC3339),
		URL:         fmt.Sprintf("%s/fight/%d", n.serverBaseURL, fightData.ID),
		Fields:      fields,
		Footer: &DiscordEmbedFooter{
			Text: "Department of Recreational Violence",
		},
	}

	return embed
}

// buildEffectsSummary creates a summary of effects applied to fighters
func (n *Notifier) buildEffectsSummary(fighter1Name string, fighter1Effects []database.AppliedEffect, fighter2Name string, fighter2Effects []database.AppliedEffect) string {
	var parts []string

	// Count effects for fighter 1
	var f1Blessings, f1Curses int
	for _, effect := range fighter1Effects {
		// Effects are stored as stat-suffixed types (e.g., speed_blessing, strength_curse)
		if strings.HasSuffix(effect.EffectType, "_blessing") {
			f1Blessings++
		} else if strings.HasSuffix(effect.EffectType, "_curse") {
			f1Curses++
		}
	}

	// Count effects for fighter 2
	var f2Blessings, f2Curses int
	for _, effect := range fighter2Effects {
		if strings.HasSuffix(effect.EffectType, "_blessing") {
			f2Blessings++
		} else if strings.HasSuffix(effect.EffectType, "_curse") {
			f2Curses++
		}
	}

	if f1Blessings > 0 || f1Curses > 0 {
		effectStr := fmt.Sprintf("**%s**: ", fighter1Name)
		if f1Blessings > 0 {
			effectStr += fmt.Sprintf("✨ %d blessing%s", f1Blessings, pluralize(f1Blessings))
		}
		if f1Curses > 0 {
			if f1Blessings > 0 {
				effectStr += ", "
			}
			effectStr += fmt.Sprintf("💀 %d curse%s", f1Curses, pluralize(f1Curses))
		}
		parts = append(parts, effectStr)
	}

	if f2Blessings > 0 || f2Curses > 0 {
		effectStr := fmt.Sprintf("**%s**: ", fighter2Name)
		if f2Blessings > 0 {
			effectStr += fmt.Sprintf("✨ %d blessing%s", f2Blessings, pluralize(f2Blessings))
		}
		if f2Curses > 0 {
			if f2Blessings > 0 {
				effectStr += ", "
			}
			effectStr += fmt.Sprintf("💀 %d curse%s", f2Curses, pluralize(f2Curses))
		}
		parts = append(parts, effectStr)
	}

	if len(parts) == 0 {
		return "No effects applied"
	}

	return strings.Join(parts, "\n")
}

// buildBettingSummary creates a summary of betting results
func (n *Notifier) buildBettingSummary(bets []database.BetWithUser, winnerID int) string {
	totalBets := len(bets)
	totalAmount := 0
	winners := 0
	winAmount := 0

	for _, bet := range bets {
		totalAmount += bet.Amount
		if bet.FighterID == winnerID {
			winners++
			if bet.Payout.Valid {
				winAmount += int(bet.Payout.Int64)
			}
		}
	}

	losers := totalBets - winners

	summary := fmt.Sprintf("**%d** total bets (**%s** credits wagered)\n", totalBets, formatNumber(totalAmount))

	if winnerID != 0 {
		summary += fmt.Sprintf("🎉 **%d** winners (paid **%s** credits)\n", winners, formatNumber(winAmount))
		summary += fmt.Sprintf("😭 **%d** losers", losers)
	} else {
		summary += "💀 All bets voided (draw)"
	}

	return summary
}

// sendViaWebhook sends the message using a Discord webhook
func (n *Notifier) sendViaWebhook(embed DiscordEmbed) error {
	message := DiscordMessage{
		Embeds: []DiscordEmbed{embed},
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	resp, err := http.Post(n.webhookURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	log.Printf("Successfully sent fight result to Discord via webhook")
	return nil
}

// sendViaBot sends the message using the Discord bot API
func (n *Notifier) sendViaBot(embed DiscordEmbed) error {
	message := DiscordMessage{
		Embeds: []DiscordEmbed{embed},
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages", n.channelID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bot "+n.botToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Discord API returned status %d", resp.StatusCode)
	}

	log.Printf("Successfully sent fight result to Discord via bot")
	return nil
}

// sendTextViaBot posts a plain text message to a specific channel via the bot API
func (n *Notifier) sendTextViaBot(channelID, content string) error {
	if n.botToken == "" {
		return fmt.Errorf("no bot token configured")
	}
	payload := map[string]string{"content": content}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages", channelID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bot "+n.botToken)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Discord API returned status %d", resp.StatusCode)
	}
	return nil
}

// AnnounceReanimationAttempt posts a message to general chat when a user attempts to reanimate a fighter
func (n *Notifier) AnnounceReanimationAttempt(user *database.User, fighter database.Fighter) error {
	if n.botToken == "" || n.generalChannelID == "" {
		return nil
	}
	display := user.CustomUsername
	if strings.TrimSpace(display) == "" {
		display = user.Username
	}
	content := fmt.Sprintf("%s is attempting to reanimate %s...", display, fighter.Name)
	return n.sendTextViaBot(n.generalChannelID, content)
}

// AnnounceNecromancer posts a success message when a user successfully reanimates a fighter
func (n *Notifier) AnnounceNecromancer(user *database.User, fighter database.Fighter) error {
	if n.botToken == "" || n.generalChannelID == "" {
		return nil
	}
	display := user.CustomUsername
	if strings.TrimSpace(display) == "" {
		display = user.Username
	}
	content := fmt.Sprintf("🧟 %s has become a NECROMANCER by reanimating %s!", display, fighter.Name)
	return n.sendTextViaBot(n.generalChannelID, content)
}

// AnnounceSponsorship posts when a user sponsors a fighter
func (n *Notifier) AnnounceSponsorship(user *database.User, fighter database.Fighter) error {
	if n.botToken == "" || n.generalChannelID == "" || user == nil {
		return nil
	}
	display := strings.TrimSpace(user.CustomUsername)
	if display == "" {
		display = strings.TrimSpace(user.Username)
	}
	if display == "" {
		display = "Unknown Patron"
	}
	content := fmt.Sprintf("%s has sponsored %s!", display, fighter.Name)
	return n.sendTextViaBot(n.generalChannelID, content)
}

// NotifyActionSummary posts a terse, plaintext settlement summary to the action channel
// Only called for fights that had wagers and a decisive result (non-draw)
func (n *Notifier) NotifyActionSummary(fightData database.Fight, winnerID int) error {
	if n.botToken == "" {
		return nil
	}

	// Fetch all bets for this fight
	bets, err := n.repo.GetAllBetsOnFight(fightData.ID)
	if err != nil {
		return err
	}
	if len(bets) == 0 || winnerID == 0 {
		return nil // nothing to announce
	}

	// Header: strike loser, bold winner
	f1 := fightData.Fighter1Name
	f2 := fightData.Fighter2Name
	var header string
	if winnerID == fightData.Fighter1ID {
		header = fmt.Sprintf("~~%s~~ vs **%s** (%s/fight/%d)", f2, f1, n.serverBaseURL, fightData.ID)
	} else {
		header = fmt.Sprintf("~~%s~~ vs **%s** (%s/fight/%d)", f1, f2, n.serverBaseURL, fightData.ID)
	}

	// Build lines with net deltas
	type line struct {
		name  string
		delta int
	}
	var lines []line
	for _, b := range bets {
		displayName := b.CustomUsername
		if strings.TrimSpace(displayName) == "" {
			displayName = b.Username
		}
		var delta int
		switch b.Status {
		case "won":
			// net change = payout - amount debited at placement
			if b.Payout.Valid {
				delta = int(b.Payout.Int64) - b.Amount
			} else {
				delta = b.Amount
			}
		case "lost":
			delta = -b.Amount
		default:
			continue // skip voided/pending
		}
		lines = append(lines, line{name: displayName, delta: delta})
	}
	if len(lines) == 0 {
		return nil
	}

	// Sort winners first by biggest gain, then losers by most lost
	sort.Slice(lines, func(i, j int) bool { return lines[i].delta > lines[j].delta })

	var sb strings.Builder
	sb.WriteString(header)
	for _, l := range lines {
		sign := ""
		if l.delta >= 0 {
			sign = "+"
		}
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("%s %s%s", l.name, sign, formatNumber(l.delta)))
	}

	content := sb.String()
	return n.sendTextViaBot(n.actionChannelID, content)
}

// Helper functions

func formatNumber(n int) string {
	// Handle negatives cleanly and avoid malformed "-,467" outputs
	if n < 0 {
		return "-" + formatNumber(-n)
	}
	str := strconv.Itoa(n)
	if len(str) <= 3 {
		return str
	}

	var result strings.Builder
	for i, digit := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result.WriteString(",")
		}
		result.WriteRune(digit)
	}
	return result.String()
}

func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
