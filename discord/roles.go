package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"spoodblort/database"
)

type RoleManager struct {
	botToken string
	guildID  string
	repo     *database.Repository
}

type DiscordMember struct {
	User  map[string]interface{} `json:"user"` // Simplified - we don't need the full user object
	Roles []string               `json:"roles"`
}

type RoleConfig struct {
	Name       string
	MinCredits int
	MaxCredits int
	ColorHex   string // Optional: role color
}

// Credit-based role configurations
var CreditRoles = []RoleConfig{
	{"💀 Broke", 0, 999, "8B0000"},              // Dark red
	{"🆕 Newbie", 1000, 9999, "00FF00"},         // Green
	{"💰 Gambler", 10000, 49999, "FFD700"},      // Gold
	{"🎰 High Roller", 50000, 99999, "9932CC"},  // Purple
	{"💎 Whale", 100000, 999999, "00FFFF"},      // Cyan
	{"🏆 Legend", 1000000, 999999999, "FF69B4"}, // Hot pink
}

func NewRoleManager(repo *database.Repository) *RoleManager {
	return &RoleManager{
		botToken: os.Getenv("DISCORD_BOT_TOKEN"),
		guildID:  os.Getenv("DISCORD_GUILD_ID"),
		repo:     repo,
	}
}

// UpdateUserRole updates a user's credit-based role in Discord
func (rm *RoleManager) UpdateUserRole(user *database.User) error {
	if rm.botToken == "" || rm.guildID == "" {
		return nil // Discord not configured
	}

	// Determine the appropriate role based on credits
	targetRole := rm.getRoleForCredits(user.Credits)

	// Get current member info from Discord
	member, err := rm.getGuildMember(user.DiscordID)
	if err != nil {
		// User not in Discord server - this is fine, just skip silently
		return nil
	}

	// Check if user already has the correct role
	if rm.memberHasRole(member, targetRole.Name) {
		return nil // Already has correct role
	}

	// Remove old credit roles
	err = rm.removeAllCreditRoles(user.DiscordID, member)
	if err != nil {
		log.Printf("Failed to remove old roles for %s: %v", user.Username, err)
	}

	// Add new role
	err = rm.addRoleToUser(user.DiscordID, targetRole.Name)
	if err != nil {
		log.Printf("Failed to add role %s to %s: %v", targetRole.Name, user.Username, err)
		return err
	}

	log.Printf("Updated Discord role for %s: %s (%d credits)", user.Username, targetRole.Name, user.Credits)
	return nil
}

// SyncAllUserRoles updates roles for all users (run periodically)
func (rm *RoleManager) SyncAllUserRoles() error {
	if rm.botToken == "" || rm.guildID == "" {
		log.Printf("Discord not configured, skipping role sync")
		return nil
	}

	users, err := rm.repo.GetAllUsersByCredits()
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	updated := 0
	for _, user := range users {
		err = rm.UpdateUserRole(&user)
		if err != nil {
			log.Printf("Failed to update role for %s: %v", user.Username, err)
			continue
		}
		updated++
	}

	log.Printf("Synced Discord roles for %d users", updated)
	return nil
}

// getRoleForCredits determines which role a user should have based on credits
func (rm *RoleManager) getRoleForCredits(credits int) RoleConfig {
	for _, role := range CreditRoles {
		if credits >= role.MinCredits && credits <= role.MaxCredits {
			return role
		}
	}
	// Default to highest role if credits exceed all ranges
	lastRole := CreditRoles[len(CreditRoles)-1]
	return lastRole
}

// getGuildMember gets member info from Discord
func (rm *RoleManager) getGuildMember(discordID string) (*DiscordMember, error) {
	url := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/members/%s", rm.guildID, discordID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bot "+rm.botToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("user not in guild")
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Discord API returned status %d", resp.StatusCode)
	}

	var member DiscordMember
	err = json.NewDecoder(resp.Body).Decode(&member)
	return &member, err
}

// memberHasRole checks if member already has a specific role
func (rm *RoleManager) memberHasRole(member *DiscordMember, roleName string) bool {
	// Get all guild roles to resolve role name to ID
	guildRoles, err := rm.getGuildRoles()
	if err != nil {
		log.Printf("Failed to get guild roles for role check: %v", err)
		return false // If we can't check, assume they don't have it
	}

	// Find the role ID for the role name
	var targetRoleID string
	for _, role := range guildRoles {
		if role.Name == roleName {
			targetRoleID = role.ID
			break
		}
	}

	if targetRoleID == "" {
		return false // Role doesn't exist, so user definitely doesn't have it
	}

	// Check if user has this role ID
	for _, userRoleID := range member.Roles {
		if userRoleID == targetRoleID {
			return true
		}
	}

	return false
}

// removeAllCreditRoles removes all credit-based roles from user
func (rm *RoleManager) removeAllCreditRoles(discordID string, member *DiscordMember) error {
	// Get all guild roles first to find IDs
	guildRoles, err := rm.getGuildRoles()
	if err != nil {
		return err
	}

	// Find credit role IDs to remove
	for _, role := range guildRoles {
		for _, creditRole := range CreditRoles {
			if role.Name == creditRole.Name {
				err = rm.removeRoleFromUser(discordID, role.ID)
				if err != nil {
					log.Printf("Failed to remove role %s: %v", role.Name, err)
				}
			}
		}
	}

	return nil
}

// addRoleToUser adds a role to a Discord user
func (rm *RoleManager) addRoleToUser(discordID, roleName string) error {
	// First, ensure the role exists
	roleID, err := rm.ensureRoleExists(roleName)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/members/%s/roles/%s", rm.guildID, discordID, roleID)
	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bot "+rm.botToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		return fmt.Errorf("failed to add role, status: %d", resp.StatusCode)
	}

	return nil
}

// removeRoleFromUser removes a role from a Discord user
func (rm *RoleManager) removeRoleFromUser(discordID, roleID string) error {
	url := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/members/%s/roles/%s", rm.guildID, discordID, roleID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bot "+rm.botToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		return fmt.Errorf("failed to remove role, status: %d", resp.StatusCode)
	}

	return nil
}

type GuildRole struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// getGuildRoles gets all roles in the guild
func (rm *RoleManager) getGuildRoles() ([]GuildRole, error) {
	url := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/roles", rm.guildID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bot "+rm.botToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get roles, status: %d", resp.StatusCode)
	}

	var roles []GuildRole
	err = json.NewDecoder(resp.Body).Decode(&roles)
	return roles, err
}

// ensureRoleExists creates role if it doesn't exist, returns role ID
func (rm *RoleManager) ensureRoleExists(roleName string) (string, error) {
	// Get existing roles
	roles, err := rm.getGuildRoles()
	if err != nil {
		return "", err
	}

	// Check if role already exists
	for _, role := range roles {
		if role.Name == roleName {
			return role.ID, nil
		}
	}

	// Create the role
	return rm.createRole(roleName)
}

// createRole creates a new role in the guild
func (rm *RoleManager) createRole(roleName string) (string, error) {
	roleData := map[string]interface{}{
		"name":        roleName,
		"permissions": "0", // No special permissions
		"color":       0,   // Default color for now
		"hoist":       false,
		"mentionable": false,
	}

	data, err := json.Marshal(roleData)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/roles", rm.guildID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bot "+rm.botToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return "", fmt.Errorf("failed to create role, status: %d", resp.StatusCode)
	}

	var newRole GuildRole
	err = json.NewDecoder(resp.Body).Decode(&newRole)
	if err != nil {
		return "", err
	}

	log.Printf("Created Discord role: %s", roleName)
	return newRole.ID, nil
}

// AssignVIPRole assigns the VIP role to a user who has accessed the casino
func (rm *RoleManager) AssignVIPRole(user *database.User) error {
	if rm.botToken == "" || rm.guildID == "" {
		log.Printf("Discord bot token or guild ID not configured")
		return nil // Don't error out, just skip
	}

	vipRoleName := "🎰 VIP"

	// Get member info
	member, err := rm.getGuildMember(user.DiscordID)
	if err != nil {
		return fmt.Errorf("failed to get guild member: %w", err)
	}

	// Check if user already has VIP role
	hasVIP := rm.memberHasRole(member, vipRoleName)
	if hasVIP {
		log.Printf("User %s already has VIP role", user.Username)
		return nil
	}

	// Assign VIP role
	err = rm.addRoleToUser(user.DiscordID, vipRoleName)
	if err != nil {
		return fmt.Errorf("failed to assign VIP role: %w", err)
	}

	log.Printf("🎰 Assigned VIP role to %s for discovering The Commissioner's underground casino", user.Username)
	return nil
}
