package web

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"spoodblort/database"

	"golang.org/x/oauth2"
)

type AuthHandler struct {
	repo        *database.Repository
	authMW      *AuthMiddleware
	oauthConfig *oauth2.Config
}

type DiscordUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
}

func NewAuthHandler(repo *database.Repository, authMW *AuthMiddleware) *AuthHandler {
	config := &oauth2.Config{
		ClientID:     os.Getenv("DISCORD_CLIENT_ID"),
		ClientSecret: os.Getenv("DISCORD_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("DISCORD_REDIRECT_URL"),
		Scopes:       []string{"identify"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://discord.com/api/oauth2/authorize",
			TokenURL: "https://discord.com/api/oauth2/token",
		},
	}

	return &AuthHandler{
		repo:        repo,
		authMW:      authMW,
		oauthConfig: config,
	}
}

// HandleLogin redirects to Discord OAuth
func (ah *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	// Generate random state for CSRF protection
	state := generateRandomState()

	// Store state in session for verification
	session, _ := ah.authMW.store.Get(r, "spoodblort-session")
	session.Values["oauth_state"] = state
	session.Save(r, w)

	url := ah.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// HandleCallback processes Discord OAuth callback
func (ah *AuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	// Verify state parameter
	session, err := ah.authMW.store.Get(r, "spoodblort-session")
	if err != nil {
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	storedState, ok := session.Values["oauth_state"].(string)
	if !ok || storedState != r.URL.Query().Get("state") {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	// Clean up state
	delete(session.Values, "oauth_state")
	session.Save(r, w)

	// Exchange code for token
	code := r.URL.Query().Get("code")
	token, err := ah.oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("OAuth token exchange failed: %v", err)
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}

	// Get user info from Discord
	discordUser, err := ah.getDiscordUser(token.AccessToken)
	if err != nil {
		log.Printf("Failed to get Discord user: %v", err)
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}

	// Check if user exists, create if not
	user, err := ah.repo.GetUserByDiscordID(discordUser.ID)
	if err != nil {
		// User doesn't exist, create new one
		avatarURL := ""
		if discordUser.Avatar != "" {
			avatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", discordUser.ID, discordUser.Avatar)
		}

		user, err = ah.repo.CreateUser(discordUser.ID, discordUser.Username, avatarURL)
		if err != nil {
			log.Printf("Failed to create user: %v", err)
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}
		log.Printf("Created new user: %s (ID: %d)", user.Username, user.ID)
	}

	// Create session
	err = ah.authMW.CreateSession(w, r, user.ID)
	if err != nil {
		log.Printf("Failed to create session: %v", err)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	log.Printf("User %s logged in successfully", user.Username)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// HandleLogout destroys user session
func (ah *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	err := ah.authMW.DestroySession(w, r)
	if err != nil {
		log.Printf("Error destroying session: %v", err)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// getDiscordUser fetches user info from Discord API
func (ah *AuthHandler) getDiscordUser(accessToken string) (*DiscordUser, error) {
	req, err := http.NewRequest("GET", "https://discord.com/api/users/@me", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Discord API returned status %d", resp.StatusCode)
	}

	var user DiscordUser
	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// generateRandomState creates a random state string for CSRF protection
func generateRandomState() string {
	token, _ := generateSessionToken()
	return token
}
