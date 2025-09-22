package web

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"spoodblort/database"
	"spoodblort/scheduler"
	"spoodblort/utils"
	"strconv"
	"strings"
	"time"

	"math/rand"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type Server struct {
	router      *mux.Router
	repo        *database.Repository
	scheduler   *scheduler.Scheduler
	authMW      *AuthMiddleware
	authH       *AuthHandler
	broadcaster *FightBroadcaster
}

// --- Signed state helpers ---
type signedState struct {
	Data string `json:"data"` // base64 JSON payload
	Sig  string `json:"sig"`  // base64 HMAC-SHA256
}

func (s *Server) signBytes(secret []byte, payload []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func (s *Server) verifyBytes(secret []byte, payload []byte, b64sig string) bool {
	expected := s.signBytes(secret, payload)
	// constant time compare
	exp, _ := base64.StdEncoding.DecodeString(expected)
	got, err := base64.StdEncoding.DecodeString(b64sig)
	if err != nil {
		return false
	}
	return hmac.Equal(exp, got)
}

type PageData struct {
	User           *database.User
	Title          string
	Tournament     *database.Tournament
	Fights         []database.Fight
	Fighter        *database.Fighter
	Fight          *database.Fight
	Users          []database.User
	Fighters       []database.Fighter
	Now            time.Time
	PrimaryColor   string
	SecondaryColor string
	UserBet        *database.Bet
	AllBets        []database.BetWithUser
	UserBets       []database.BetWithFight
	// Meta tags for social media
	MetaDescription    string
	MetaImage          string
	MetaType           string
	ViewerCount        int
	Fighter1           *database.Fighter
	Fighter2           *database.Fighter
	ShopItems          []database.ShopItem
	UserInventory      []database.UserInventoryItem // Added for user inventory
	Fighter1Effects    []database.AppliedEffect
	Fighter2Effects    []database.AppliedEffect
	Fighter1Curses     int
	Fighter1Blessings  int
	Fighter2Curses     int
	Fighter2Blessings  int
	UserEffectsOnFight []database.AppliedEffectWithUser // New field for user effects
	// MVP-related fields
	CurrentMVP   *database.UserSetting
	CanChangeMVP bool
	CreatorUser  *database.User         // For custom fighter creator info
	NextFight    *database.Fight        // For countdown timer across weekend gaps
	BettingStats *database.BettingStats // For comprehensive betting statistics
	// CSS optimization
	RequiredCSS []string // Page-specific CSS files to load
	// Access and limits
	FightBetMax  int // Per-user fight bet cap (min of credits and policy)
	CasinoBetMax int // Casino wager cap (100M unless sacrifice exemption)
}

func NewServer(repo *database.Repository, scheduler *scheduler.Scheduler, sessionSecret string) *Server {
	authMW := NewAuthMiddleware(repo, sessionSecret)
	authH := NewAuthHandler(repo, authMW)

	s := &Server{
		router:      mux.NewRouter().StrictSlash(true),
		repo:        repo,
		scheduler:   scheduler,
		authMW:      authMW,
		authH:       authH,
		broadcaster: NewFightBroadcaster(repo),
	}

	// Derive HMAC key from sessionSecret; reuse sessionSecret bytes directly
	_ = sessionSecret

	// Connect the broadcaster to the scheduler so fights can broadcast live
	scheduler.SetBroadcaster(s.broadcaster)

	s.setupRoutes()
	return s
}

// GetBroadcaster returns the fight broadcaster for use by background processes
func (s *Server) GetBroadcaster() *FightBroadcaster {
	return s.broadcaster
}

func (s *Server) setupRoutes() {
	// Static files
	s.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	// Public routes
	public := s.router.PathPrefix("").Subrouter()
	public.Use(s.authMW.LoadUser)

	public.HandleFunc("/", s.handleIndex).Methods("GET")
	public.HandleFunc("/auth", s.handleAuthPage).Methods("GET")
	public.HandleFunc("/auth/discord", s.authH.HandleLogin).Methods("GET")
	public.HandleFunc("/auth/discord/callback", s.authH.HandleCallback).Methods("GET")
	public.HandleFunc("/logout", s.authH.HandleLogout).Methods("POST")
	public.HandleFunc("/about", s.handleAbout).Methods("GET")
	public.HandleFunc("/fighters", s.handleFighters).Methods("GET")
	public.HandleFunc("/fighter/{id}", s.handleFighter).Methods("GET")
	public.HandleFunc("/fight/{id}", s.handleFight).Methods("GET")
	public.HandleFunc("/leaderboard", s.handleLeaderboard).Methods("GET")
	public.HandleFunc("/closed", s.handleClosedPage).Methods("GET")
	public.HandleFunc("/favicon.ico", s.handleFavicon).Methods("GET")
	public.HandleFunc("/user/@{username}", s.handleUserProfile).Methods("GET")

	// Shop routes (public so anyone can view, but purchase requires auth)
	public.HandleFunc("/shop", s.handleShop).Methods("GET")

	// Watch routes (public so anyone can watch)
	public.HandleFunc("/watch/{id:[0-9]+}", s.handleWatch).Methods("GET")

	// WebSocket route (public, no auth required for watching)
	public.HandleFunc("/ws/fight/{id:[0-9]+}", s.broadcaster.HandleWebSocket)

	// Protected routes (require authentication)
	protected := s.router.PathPrefix("/user").Subrouter()
	protected.Use(s.authMW.LoadUser)
	protected.Use(s.authMW.RequireAuth)

	protected.HandleFunc("/dashboard", s.handleUserDashboard).Methods("GET")
	protected.HandleFunc("/settings", s.handleUserSettings).Methods("GET")
	protected.HandleFunc("/settings", s.handleUserSettingsPost).Methods("POST")
	protected.HandleFunc("/settings/mvp", s.handleUpdateMVP).Methods("POST")
	protected.HandleFunc("/leaderboard", s.handleLeaderboard).Methods("GET")
	protected.HandleFunc("/fighters", s.handleFighters).Methods("GET")

	// Add betting routes
	protected.HandleFunc("/fight/{id}/bet", s.handlePlaceBet).Methods("POST")

	// Shop purchase route (requires auth)
	protected.HandleFunc("/shop/purchase", s.handleShopPurchase).Methods("POST")

	// Apply effect route (requires auth)
	protected.HandleFunc("/fight/apply-effect", s.handleApplyEffect).Methods("POST")

	// Fighter creation route (requires auth)
	protected.HandleFunc("/create-fighter", s.handleCreateFighter).Methods("GET")
	protected.HandleFunc("/create-fighter", s.handleCreateFighterPost).Methods("POST")

	// Casino routes (requires auth)
	protected.HandleFunc("/casino", s.handleCasino).Methods("GET")
	protected.HandleFunc("/casino/moonflip", s.handleMoonFlip).Methods("POST")
	protected.HandleFunc("/casino/hilow-step1", s.handleHiLowStep1).Methods("POST")
	protected.HandleFunc("/casino/hilow-step2", s.handleHiLowStep2).Methods("POST")
	protected.HandleFunc("/casino/slots", s.handleSlots).Methods("POST")
	protected.HandleFunc("/casino/jackpot", s.handleGetJackpot).Methods("GET")

	// Blackjack routes (stateless, like other games)
	protected.HandleFunc("/casino/blackjack/start", s.handleBlackjackStart).Methods("POST")
	protected.HandleFunc("/casino/blackjack/hit", s.handleBlackjackHit).Methods("POST")
	protected.HandleFunc("/casino/blackjack/stand", s.handleBlackjackStand).Methods("POST")

	// Extortion event resolver
	protected.HandleFunc("/casino/extortion", s.handleExtortionResolve).Methods("POST")
}

// userHasSacrificeExemption returns true if the user has at least 1000 sacrifices
func (s *Server) userHasSacrificeExemption(userID int) bool {
	inv, err := s.repo.GetUserInventory(userID)
	if err != nil {
		return false
	}
	for _, it := range inv {
		if it.ItemType == "sacrifice" && it.Quantity >= 1000 {
			return true
		}
	}
	return false
}

// getUserMaxFightBet returns the per-user fight bet cap. Default 1,000,000 unless
// the user owns a high_roller card, in which case we use the item's EffectValue
// (e.g., 100,000,000). Falls back safely on error.
func (s *Server) getUserMaxFightBet(user *database.User) int {
	if user == nil {
		return 1000000
	}
	inv, err := s.repo.GetUserInventory(user.ID)
	if err != nil {
		return 1000000
	}
	for _, it := range inv {
		if it.ItemType == "high_roller" && it.Quantity > 0 {
			if it.EffectValue > 0 {
				return it.EffectValue
			}
			return 100000000
		}
	}
	return 1000000
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	centralTime, _ := time.LoadLocation("America/Chicago")
	now := time.Now().In(centralTime)

	// Check if it's Sunday - serve closed page
	if now.Weekday() == time.Sunday {
		user := GetUserFromContext(r.Context())
		data := PageData{
			User:        user,
			Title:       "Department of Recreational Violence - CLOSED",
			RequiredCSS: []string{"closed.css"},
		}

		// Add colors if user is present
		if user != nil {
			primaryColor, secondaryColor := utils.GenerateUserColors(user.DiscordID)
			data.PrimaryColor = primaryColor
			data.SecondaryColor = secondaryColor
		}

		s.renderTemplate(w, "closed.html", data)
		return
	}

	// Ensure today's schedule exists
	// REMOVED: This was causing performance issues by running on every page load
	// Schedule is now ensured at startup in main.go
	// err := s.scheduler.EnsureTodaysSchedule(now)
	// if err != nil {
	//	log.Printf("Error ensuring schedule: %v", err)
	//	http.Error(w, "Internal server error", http.StatusInternalServerError)
	//	return
	// }

	// Get current tournament (once, reuse for all queries)
	tournament, err := s.scheduler.GetCurrentTournament(now)
	if err != nil {
		log.Printf("Error getting tournament: %v", err)
		tournament = nil
	}

	// Get today's fights (pass tournament to avoid re-querying)
	var fights []database.Fight
	if tournament != nil {
		today, tomorrow := utils.GetDayBounds(now)
		fights, err = s.repo.GetTodaysFights(tournament.ID, today, tomorrow)
		if err != nil {
			log.Printf("Error getting schedule: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	// Get the next fight (pass tournament to avoid re-querying)
	var nextFight *database.Fight
	if tournament != nil {
		nextFight, err = s.scheduler.GetNextFight(now)
		if err != nil {
			log.Printf("Error getting next fight: %v", err)
			// Continue without next fight data
			nextFight = nil
		}
	}

	user := GetUserFromContext(r.Context())
	data := PageData{
		User:            user,
		Title:           "Fight Schedule",
		Fights:          fights,
		NextFight:       nextFight,
		Tournament:      tournament,
		Now:             now,
		MetaDescription: "🔥 TODAY'S VIOLENCE SCHEDULE 🔥 24 IMPOSSIBLE FIGHTS EVERY 30 MINUTES. FIGHTERS WITH BLOOD TYPE 'NACHO CHEESE' AND 1000 TOES AWAIT YOUR DEGENERATE GAMBLING. WITNESS THE CHAOS. EMBRACE THE EXISTENTIAL DREAD.",
		MetaType:        "website",
		RequiredCSS:     []string{"schedule.css"},
	}

	// Add colors and MVP setting if user is present
	if user != nil {
		primaryColor, secondaryColor := utils.GenerateUserColors(user.DiscordID)
		data.PrimaryColor = primaryColor
		data.SecondaryColor = secondaryColor

		// Get user's MVP setting
		mvpSetting, err := s.repo.GetUserSetting(user.ID, "mvp_player")
		if err == nil {
			data.CurrentMVP = mvpSetting
		}
	}

	s.renderTemplate(w, "index.html", data)
}

func (s *Server) handleAuthPage(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		User:        GetUserFromContext(r.Context()),
		Title:       "Authentication",
		RequiredCSS: []string{"auth.css"},
	}

	s.renderTemplate(w, "auth.html", data)
}

func (s *Server) handleAbout(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	data := PageData{
		User:        user,
		Title:       "About & Rules",
		RequiredCSS: []string{"about.css"},
	}

	// Add colors if user is present
	if user != nil {
		primaryColor, secondaryColor := utils.GenerateUserColors(user.DiscordID)
		data.PrimaryColor = primaryColor
		data.SecondaryColor = secondaryColor
	}

	s.renderTemplate(w, "about.html", data)
}

func (s *Server) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())

	// Get all users ordered by credits descending
	users, err := s.repo.GetAllUsersByCredits()
	if err != nil {
		log.Printf("Error getting users for leaderboard: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := PageData{
		User:            user,
		Title:           "Violence Credit Leaderboard",
		Users:           users,
		MetaDescription: "🏆 VIOLENCE CREDIT LEADERBOARD 🏆 WITNESS THE MOST SUCCESSFUL DEGENERATE GAMBLERS IN THE CHAOS DIMENSION. THESE LEGENDS HAVE MASTERED THE ART OF BETTING ON IMPOSSIBLE FIGHTER STATS. FEAR THEIR PORTFOLIOS.",
		MetaType:        "website",
		RequiredCSS:     []string{"leaderboard.css"},
	}

	// Add colors if user is present
	if user != nil {
		primaryColor, secondaryColor := utils.GenerateUserColors(user.DiscordID)
		data.PrimaryColor = primaryColor
		data.SecondaryColor = secondaryColor
	}

	s.renderTemplate(w, "leaderboard.html", data)
}

func (s *Server) handleFighters(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())

	// Get all fighters ordered by wins/losses
	fighters, err := s.repo.GetAllFightersByRecord()
	if err != nil {
		log.Printf("Error getting fighters: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := PageData{
		User:            user,
		Title:           "Fighter Rankings",
		Fighters:        fighters,
		MetaDescription: "👊 FIGHTER RANKINGS 👊 DISCOVER THE MOST VIOLENT COMBATANTS IN THE CHAOS DIMENSION. THESE LEGENDS HAVE CONQUERED THE UNCONQUERABLE. FEAR THEIR POWER.",
		MetaType:        "website",
		RequiredCSS:     []string{"fighters.css"},
	}

	// Add colors if user is present
	if user != nil {
		primaryColor, secondaryColor := utils.GenerateUserColors(user.DiscordID)
		data.PrimaryColor = primaryColor
		data.SecondaryColor = secondaryColor
	}

	s.renderTemplate(w, "fighters.html", data)
}

func (s *Server) handleFighter(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fighterID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid fighter ID", http.StatusBadRequest)
		return
	}

	fighter, err := s.repo.GetFighter(fighterID)
	if err != nil {
		log.Printf("Fighter not found: %v", err)
		fighter = nil
	}

	user := GetUserFromContext(r.Context())
	data := PageData{
		User:        user,
		Title:       "Fighter Profile",
		Fighter:     fighter,
		RequiredCSS: []string{"fighter.css"},
	}

	// If this is a custom fighter with a creator, get the creator's info
	var creatorUser *database.User
	if fighter != nil && fighter.IsCustom && fighter.CreatedByUserID != nil {
		log.Printf("Debug: Looking up creator for fighter %s, creator ID: %d", fighter.Name, *fighter.CreatedByUserID)
		creatorUser, err = s.repo.GetUser(*fighter.CreatedByUserID)
		if err != nil {
			log.Printf("Error getting creator user info for ID %d: %v", *fighter.CreatedByUserID, err)
			creatorUser = nil
		} else {
			log.Printf("Debug: Found creator user: %s (custom: %s)", creatorUser.Username, creatorUser.CustomUsername)
		}
	}

	if fighter != nil {
		data.Title = fighter.Name
		data.MetaDescription = fmt.Sprintf("⚔️ %s ⚔️ %s CLASS FIGHTER OF PURE CHAOS. %d WINS, %d LOSSES. BLOOD TYPE: %s. HOROSCOPE: %s. EXISTENTIAL DREAD LEVEL: %d. MOLECULAR DENSITY UNKNOWN TO SCIENCE.",
			fighter.Name, strings.ToUpper(fighter.FighterClass), fighter.Wins, fighter.Losses, strings.ToUpper(fighter.BloodType), strings.ToUpper(fighter.Horoscope), fighter.ExistentialDread)
		data.MetaType = "profile"
	} else {
		data.MetaDescription = "💀 FIGHTER NOT FOUND IN THE VIOLENCE DATABASE. THEY MAY HAVE BEEN ABSORBED INTO THE CHAOS VOID. 💀"
	}

	// Add colors if user is present
	if user != nil {
		primaryColor, secondaryColor := utils.GenerateUserColors(user.DiscordID)
		data.PrimaryColor = primaryColor
		data.SecondaryColor = secondaryColor
	}

	// Add creator user to the data
	if creatorUser != nil {
		log.Printf("Debug: Adding creator user to template data: %s", creatorUser.Username)
		data.CreatorUser = creatorUser
	} else {
		log.Printf("Debug: No creator user to add to template data")
	}

	s.renderTemplate(w, "fighter.html", data)
}

func (s *Server) handleFight(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fightID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid fight ID", http.StatusBadRequest)
		return
	}

	fight, err := s.repo.GetFight(fightID)
	if err != nil {
		log.Printf("Fight not found: %v", err)
		fight = nil
	}

	user := GetUserFromContext(r.Context())
	data := PageData{
		User:        user,
		Title:       "Fight Details",
		Fight:       fight,
		RequiredCSS: []string{"fight.css"},
	}

	if fight != nil {
		data.Title = fmt.Sprintf("%s vs %s", fight.Fighter1Name, fight.Fighter2Name)
		statusText := "SCHEDULED FOR MAXIMUM VIOLENCE"
		if fight.Status == "active" {
			statusText = "🔴 LIVE VIOLENCE IN PROGRESS 🔴"
		} else if fight.Status == "completed" {
			statusText = "VIOLENCE CONCLUDED"
		} else if fight.Status == "voided" {
			statusText = "ABSORBED BY THE CHAOS VOID"
		}
		data.MetaDescription = fmt.Sprintf("💥 VIOLENCE BREAKDOWN 💥 %s VS %s (%s). IMPOSSIBLE STATS COLLIDE. BLOOD WILL BE SPILLED. CREDITS WILL BE LOST. BET ON THE CHAOS.",
			strings.ToUpper(fight.Fighter1Name), strings.ToUpper(fight.Fighter2Name), statusText)
		data.MetaType = "article"
	} else {
		data.MetaDescription = "💀 FIGHT NOT FOUND IN THE VIOLENCE DATABASE. IT MAY HAVE NEVER EXISTED. 💀"
	}

	// If user is logged in and fight exists, get betting data
	if user != nil && fight != nil {
		// Get user's existing bet on this fight
		userBet, err := s.repo.GetUserBetOnFight(user.ID, fightID)
		if err == nil {
			data.UserBet = userBet
		}

		// Get all bets on this fight
		allBets, err := s.repo.GetAllBetsOnFight(fightID)
		if err == nil {
			data.AllBets = allBets
		}

		// Get user's inventory for bless/curse options
		userInventory, err := s.repo.GetUserInventory(user.ID)
		if err == nil {
			data.UserInventory = userInventory
		}
	}

	// Get applied effects for both fighters (for display)
	if fight != nil {
		// Determine which date to use for effects based on fight status
		var effectDate time.Time
		if fight.Status == "active" || fight.Status == "scheduled" {
			// For active and scheduled fights, use today's effects (live viewing) in Central Time
			centralTime, _ := time.LoadLocation("America/Chicago")
			effectDate = time.Now().In(centralTime)
		} else {
			// For completed/voided fights, use the fight's scheduled date (historical viewing)
			effectDate = fight.ScheduledTime
		}

		// Get day bounds for the effect date
		startDate := time.Date(effectDate.Year(), effectDate.Month(), effectDate.Day(), 0, 0, 0, 0, effectDate.Location())
		endDate := startDate.Add(24 * time.Hour)

		fighter1Effects, err := s.repo.GetAppliedEffectsForDate("fighter", fight.Fighter1ID, startDate, endDate)
		if err == nil {
			data.Fighter1Effects = fighter1Effects
			// Count effects
			for _, effect := range fighter1Effects {
				if effect.EffectType == "fighter_curse" {
					data.Fighter1Curses++
				} else if effect.EffectType == "fighter_blessing" {
					data.Fighter1Blessings++
				}
			}
		}

		fighter2Effects, err := s.repo.GetAppliedEffectsForDate("fighter", fight.Fighter2ID, startDate, endDate)
		if err == nil {
			data.Fighter2Effects = fighter2Effects
			// Count effects
			for _, effect := range fighter2Effects {
				if effect.EffectType == "fighter_curse" {
					data.Fighter2Curses++
				} else if effect.EffectType == "fighter_blessing" {
					data.Fighter2Blessings++
				}
			}
		}

		// Get user effects applied to this fight
		userEffectsOnFight, err := s.repo.GetAppliedEffectsByUserForFight(fightID)
		if err != nil {
			log.Printf("Error getting user effects for fight %d: %v", fightID, err)
			userEffectsOnFight = nil
		}

		// Filter effects to only show ones from the same date range as fighter effects
		var filteredUserEffects []database.AppliedEffectWithUser
		for _, effect := range userEffectsOnFight {
			// Convert effect's created_at to Central Time for proper comparison
			centralTime, _ := time.LoadLocation("America/Chicago")
			effectTimeInCentral := effect.CreatedAt.In(centralTime)

			// Check if effect's created_at is within our date range
			if (effectTimeInCentral.After(startDate) || effectTimeInCentral.Equal(startDate)) && effectTimeInCentral.Before(endDate) {
				filteredUserEffects = append(filteredUserEffects, effect)
			}
		}
		data.UserEffectsOnFight = filteredUserEffects
	}

	// Add colors if user is present
	if user != nil {
		primaryColor, secondaryColor := utils.GenerateUserColors(user.DiscordID)
		data.PrimaryColor = primaryColor
		data.SecondaryColor = secondaryColor

		// Compute user's fight bet cap (frontend hint only; backend enforces)
		data.FightBetMax = s.getUserMaxFightBet(user)
	}

	s.renderTemplate(w, "fight.html", data)
}

func (s *Server) handlePlaceBet(w http.ResponseWriter, r *http.Request) {
	// User authentication is guaranteed by the RequireAuth middleware
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	fightID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid fight ID", http.StatusBadRequest)
		return
	}

	// Parse form data
	err = r.ParseForm()
	if err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	fighterID, err := strconv.Atoi(r.FormValue("fighter_id"))
	if err != nil {
		http.Error(w, "Invalid fighter ID", http.StatusBadRequest)
		return
	}

	amount, err := strconv.Atoi(r.FormValue("amount"))
	if err != nil || amount <= 0 {
		http.Error(w, "Invalid bet amount", http.StatusBadRequest)
		return
	}

	// Validate bet amount doesn't exceed user's credits or per-user cap
	if amount > user.Credits {
		http.Error(w, "Insufficient credits", http.StatusBadRequest)
		return
	}

	maxAllowed := s.getUserMaxFightBet(user)
	if amount > maxAllowed {
		http.Error(w, fmt.Sprintf("Bet exceeds allowed maximum (%d)", maxAllowed), http.StatusBadRequest)
		return
	}

	// Get fight to validate it's in scheduled status
	fight, err := s.repo.GetFight(fightID)
	if err != nil {
		http.Error(w, "Fight not found", http.StatusNotFound)
		return
	}

	if fight.Status != "scheduled" {
		http.Error(w, "Betting is closed for this fight", http.StatusBadRequest)
		return
	}

	// Validate fighter is actually in this fight
	if fighterID != fight.Fighter1ID && fighterID != fight.Fighter2ID {
		http.Error(w, "Invalid fighter for this fight", http.StatusBadRequest)
		return
	}

	// Check if user already has a bet on this fight
	existingBet, err := s.repo.GetUserBetOnFight(user.ID, fightID)
	if err == nil && existingBet != nil {
		http.Error(w, "You already have a bet on this fight", http.StatusBadRequest)
		return
	}

	// Create the bet and deduct credits in a transaction-like manner
	err = s.repo.CreateBet(user.ID, fightID, fighterID, amount)
	if err != nil {
		log.Printf("Failed to create bet: %v", err)
		http.Error(w, "Failed to place bet", http.StatusInternalServerError)
		return
	}

	// Deduct credits from user
	err = s.repo.UpdateUserCredits(user.ID, user.Credits-amount)
	if err != nil {
		log.Printf("Failed to deduct credits: %v", err)
		http.Error(w, "Failed to process bet", http.StatusInternalServerError)
		return
	}

	// Redirect back to fight page
	http.Redirect(w, r, "/fight/"+strconv.Itoa(fightID), http.StatusSeeOther)
}

func (s *Server) handleUserDashboard(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth", http.StatusSeeOther)
		return
	}

	// Generate colors for user
	primaryColor, secondaryColor := utils.GenerateUserColors(user.DiscordID)

	// Fetch user's betting history
	userBets, err := s.repo.GetUserBets(user.ID)
	if err != nil {
		log.Printf("Error fetching user bets: %v", err)
		userBets = nil // Ensure it's nil if fetching fails
	}

	// Fetch user's inventory
	userInventory, err := s.repo.GetUserInventory(user.ID)
	if err != nil {
		log.Printf("Error fetching user inventory: %v", err)
		userInventory = nil
	}

	// Fetch betting stats
	bettingStats, err := s.repo.GetUserBettingStats(user.ID)
	if err != nil {
		log.Printf("Error fetching betting stats: %v", err)
		bettingStats = nil
	}

	data := PageData{
		User:           user,
		Title:          "Dashboard",
		PrimaryColor:   primaryColor,
		SecondaryColor: secondaryColor,
		UserBets:       userBets,
		UserInventory:  userInventory,
		BettingStats:   bettingStats,
		RequiredCSS:    []string{"dashboard.css"},
	}

	s.renderTemplate(w, "dashboard.html", data)
}

func (s *Server) handleUserSettings(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth", http.StatusSeeOther)
		return
	}

	// Generate colors for user
	primaryColor, secondaryColor := utils.GenerateUserColors(user.DiscordID)

	// Get all fighters for MVP dropdown
	fighters, err := s.repo.GetAllFightersByRecord()
	if err != nil {
		log.Printf("Error getting fighters: %v", err)
		fighters = nil
	}

	// Get user's current MVP setting
	var currentMVP *database.UserSetting
	var canChangeMVP bool = true
	mvpSetting, err := s.repo.GetUserSetting(user.ID, "mvp_player")
	if err == nil {
		currentMVP = mvpSetting
		canChangeMVP, _ = s.repo.CanChangeUserSetting(user.ID, "mvp_player")
	}

	// Get user inventory to check if they have MVP item
	userInventory, err := s.repo.GetUserInventory(user.ID)
	if err != nil {
		log.Printf("Error getting user inventory: %v", err)
		userInventory = nil
	}

	data := PageData{
		User:           user,
		Title:          "Settings",
		PrimaryColor:   primaryColor,
		SecondaryColor: secondaryColor,
		Fighters:       fighters,
		UserInventory:  userInventory,
		CurrentMVP:     currentMVP,
		CanChangeMVP:   canChangeMVP,
		RequiredCSS:    []string{"settings.css"},
	}

	s.renderTemplate(w, "settings.html", data)
}

func (s *Server) handleUserSettingsPost(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth", http.StatusSeeOther)
		return
	}

	// Parse form data
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	customUsername := r.FormValue("custom_username")

	// Update user's custom username in database
	err = s.repo.UpdateUserCustomUsername(user.ID, customUsername)
	if err != nil {
		log.Printf("Failed to update custom username: %v", err)
		http.Error(w, "Failed to update settings", http.StatusInternalServerError)
		return
	}

	// Redirect back to settings page
	http.Redirect(w, r, "/user/settings", http.StatusSeeOther)
}

func (s *Server) handleUpdateMVP(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse JSON request
	var req struct {
		FighterID   int  `json:"fighter_id"`
		PayToChange bool `json:"pay_to_change"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Printf("Failed to decode MVP update request: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Check if user owns MVP Player lvl 1 item
	userInventory, err := s.repo.GetUserInventory(user.ID)
	if err != nil {
		log.Printf("Failed to get user inventory: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to check inventory",
		})
		return
	}

	hasMVPItem := false
	for _, item := range userInventory {
		if item.ItemType == "mvp_player" && item.Quantity > 0 {
			hasMVPItem = true
			break
		}
	}

	if !hasMVPItem {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "You need to purchase MVP Player lvl 1 first",
		})
		return
	}

	// Validate fighter exists
	_, err = s.repo.GetFighter(req.FighterID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Fighter not found",
		})
		return
	}

	if req.PayToChange {
		// Pay 1000 credits to change MVP
		if user.Credits < 1000 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Insufficient credits (need 1000)",
			})
			return
		}

		err = s.repo.PayToChangeUserSetting(user.ID, "mvp_player", fmt.Sprintf("%d", req.FighterID), 1000)
		if err != nil {
			log.Printf("Failed to pay for MVP change: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Failed to process payment",
			})
			return
		}
	} else {
		// Check if they can change for free
		canChange, err := s.repo.CanChangeUserSetting(user.ID, "mvp_player")
		if err != nil {
			log.Printf("Failed to check MVP change eligibility: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Failed to check eligibility",
			})
			return
		}

		if !canChange {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "You must wait until next tournament or pay 1000 credits to change MVP",
			})
			return
		}

		// Get current tournament week to set next change date
		centralTime, _ := time.LoadLocation("America/Chicago")
		now := time.Now().In(centralTime)

		// Calculate next tournament start (add 7 days)
		nextTournamentStart := now.AddDate(0, 0, 7)

		err = s.repo.SetUserSetting(user.ID, "mvp_player", fmt.Sprintf("%d", req.FighterID), &nextTournamentStart)
		if err != nil {
			log.Printf("Failed to set MVP: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Failed to set MVP",
			})
			return
		}
	}

	// Get fighter name for response
	fighter, _ := s.repo.GetFighter(req.FighterID)
	fighterName := "Unknown Fighter"
	if fighter != nil {
		fighterName = fighter.Name
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Successfully set %s as your MVP!", fighterName),
	})
}

func (s *Server) handleClosedPage(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		User:        GetUserFromContext(r.Context()),
		Title:       "Department of Recreational Violence - CLOSED",
		RequiredCSS: []string{"closed.css"},
	}
	s.renderTemplate(w, "closed.html", data)
}

func (s *Server) handleFavicon(w http.ResponseWriter, r *http.Request) {
	faviconSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 32 32">
		<rect width="32" height="32" fill="#000000"/>
		<text x="16" y="22" font-family="Times, serif" font-size="20" font-weight="bold" text-anchor="middle" fill="#FFFFFF">S</text>
	</svg>`

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=31536000") // Cache for 1 year
	w.Write([]byte(faviconSVG))
}

// handleWatch renders the live fight watch page
func (s *Server) handleWatch(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fightID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid fight ID", http.StatusBadRequest)
		return
	}

	// Get fight details
	fight, err := s.repo.GetFight(fightID)
	if err != nil {
		if err == sql.ErrNoRows {
			data := PageData{
				Title:       "Violence Not Found",
				Fight:       nil,
				RequiredCSS: []string{"watch.css"},
			}
			s.renderTemplate(w, "watch.html", data)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Get fighters (with nil check for safety)
	var fighter1, fighter2 *database.Fighter
	if fight != nil {
		fighter1, err = s.repo.GetFighter(fight.Fighter1ID)
		if err != nil {
			log.Printf("Error getting fighter1: %v", err)
			fighter1 = nil
		}

		fighter2, err = s.repo.GetFighter(fight.Fighter2ID)
		if err != nil {
			log.Printf("Error getting fighter2: %v", err)
			fighter2 = nil
		}
	}

	// Get viewer count
	viewerCount := s.broadcaster.GetViewerCount(fightID)

	// Get user from context (for navigation)
	user := GetUserFromContext(r.Context())

	// Get all bets on this fight
	allBets, err := s.repo.GetAllBetsOnFight(fightID)
	if err != nil {
		log.Printf("Error getting bets for fight %d: %v", fightID, err)
		allBets = nil
	}

	// Get applied effects for both fighters
	var fighter1Effects, fighter2Effects []database.AppliedEffect
	var fighter1Curses, fighter1Blessings, fighter2Curses, fighter2Blessings int

	if fight != nil {
		// Determine which date to use for effects based on fight status
		var effectDate time.Time
		if fight.Status == "active" || fight.Status == "scheduled" {
			// For active and scheduled fights, use today's effects (live viewing) in Central Time
			centralTime, _ := time.LoadLocation("America/Chicago")
			effectDate = time.Now().In(centralTime)
		} else {
			// For completed/voided fights, use the fight's scheduled date (historical viewing)
			effectDate = fight.ScheduledTime
		}

		// Get day bounds for the effect date
		startDate := time.Date(effectDate.Year(), effectDate.Month(), effectDate.Day(), 0, 0, 0, 0, effectDate.Location())
		endDate := startDate.Add(24 * time.Hour)

		fighter1Effects, err = s.repo.GetAppliedEffectsForDate("fighter", fight.Fighter1ID, startDate, endDate)
		if err == nil {
			// Count effects
			for _, effect := range fighter1Effects {
				if strings.Contains(effect.EffectType, "_curse") {
					fighter1Curses++
				} else if strings.Contains(effect.EffectType, "_blessing") {
					fighter1Blessings++
				}
			}
		}

		fighter2Effects, err = s.repo.GetAppliedEffectsForDate("fighter", fight.Fighter2ID, startDate, endDate)
		if err == nil {
			// Count effects
			for _, effect := range fighter2Effects {
				if strings.Contains(effect.EffectType, "_curse") {
					fighter2Curses++
				} else if strings.Contains(effect.EffectType, "_blessing") {
					fighter2Blessings++
				}
			}
		}
	}

	// Get user's bet on this fight if logged in
	var userBet *database.Bet
	if user != nil {
		userBet, err = s.repo.GetUserBetOnFight(user.ID, fightID)
		if err != nil {
			// No bet or error - that's fine
			userBet = nil
		}
	}

	// Get user effects applied to this fight
	var userEffectsOnFight []database.AppliedEffectWithUser
	if fight != nil {
		allUserEffects, err := s.repo.GetAppliedEffectsByUserForFight(fightID)
		if err != nil {
			log.Printf("Error getting user effects for fight %d: %v", fightID, err)
			allUserEffects = nil
		}

		// Filter effects to only show ones from the same date range as fighter effects
		// Use the same date calculation logic as above
		var effectDate time.Time
		if fight.Status == "active" || fight.Status == "scheduled" {
			centralTime, _ := time.LoadLocation("America/Chicago")
			effectDate = time.Now().In(centralTime)
		} else {
			effectDate = fight.ScheduledTime
		}

		startDate := time.Date(effectDate.Year(), effectDate.Month(), effectDate.Day(), 0, 0, 0, 0, effectDate.Location())
		endDate := startDate.Add(24 * time.Hour)

		for _, effect := range allUserEffects {
			// Check if effect's created_at is within our date range
			if (effect.CreatedAt.After(startDate) || effect.CreatedAt.Equal(startDate)) && effect.CreatedAt.Before(endDate) {
				userEffectsOnFight = append(userEffectsOnFight, effect)
			}
		}
	}

	data := PageData{
		Title:              fmt.Sprintf("🔴 %s vs %s - Live Violence", fight.Fighter1Name, fight.Fighter2Name),
		Fight:              fight,
		Fighter1:           fighter1,
		Fighter2:           fighter2,
		User:               user,
		ViewerCount:        viewerCount,
		AllBets:            allBets,
		UserBet:            userBet,
		Fighter1Effects:    fighter1Effects,
		Fighter2Effects:    fighter2Effects,
		Fighter1Curses:     fighter1Curses,
		Fighter1Blessings:  fighter1Blessings,
		Fighter2Curses:     fighter2Curses,
		Fighter2Blessings:  fighter2Blessings,
		UserEffectsOnFight: userEffectsOnFight,
		MetaDescription:    fmt.Sprintf("🔴 LIVE VIOLENCE! WITNESS %s BATTLE %s IN THE VIOLENCE THEATER! REAL-TIME CARNAGE WITH PREMIUM DEGENERATES COMMENTARY!", fight.Fighter1Name, fight.Fighter2Name),
		MetaType:           "article",
		RequiredCSS:        []string{"watch.css"},
	}

	// Add colors if user is present
	if user != nil {
		primaryColor, secondaryColor := utils.GenerateUserColors(user.DiscordID)
		data.PrimaryColor = primaryColor
		data.SecondaryColor = secondaryColor
	}

	s.renderTemplate(w, "watch.html", data)
}

// handleShop renders the shop page
func (s *Server) handleShop(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())

	// Get all shop items
	shopItems, err := s.repo.GetAllShopItems()
	if err != nil {
		log.Printf("Error getting shop items: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := PageData{
		User:            user,
		Title:           "The Chaos Marketplace",
		ShopItems:       shopItems,
		MetaDescription: "🛒 THE CHAOS MARKETPLACE 🛒 PURCHASE ITEMS TO MANIPULATE THE FABRIC OF REALITY ITSELF. CURSES, BLESSINGS, AND COSMIC SACRIFICE AWAIT YOUR CREDITS.",
		MetaType:        "website",
		RequiredCSS:     []string{"shop.css"},
	}

	// Get user inventory if logged in
	if user != nil {
		userInventory, err := s.repo.GetUserInventory(user.ID)
		if err != nil {
			log.Printf("Error getting user inventory: %v", err)
		} else {
			data.UserInventory = userInventory
		}

		primaryColor, secondaryColor := utils.GenerateUserColors(user.DiscordID)
		data.PrimaryColor = primaryColor
		data.SecondaryColor = secondaryColor
	}

	s.renderTemplate(w, "shop.html", data)
}

// handleShopPurchase handles item purchases via AJAX
func (s *Server) handleShopPurchase(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse JSON request
	var req struct {
		ItemID   int `json:"item_id"`
		Quantity int `json:"quantity"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Printf("Failed to decode purchase request: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate quantity
	if req.Quantity <= 0 {
		http.Error(w, "Invalid quantity", http.StatusBadRequest)
		return
	}

	// Get shop item
	item, err := s.repo.GetShopItem(req.ItemID)
	if err != nil {
		log.Printf("Shop item not found: %v", err)
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	// Check if this is a unique item and user already owns it
	if item.ItemType == "mvp_player" || item.ItemType == "high_roller" {
		userInventory, err := s.repo.GetUserInventory(user.ID)
		if err != nil {
			log.Printf("Failed to get user inventory: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Failed to check inventory",
			})
			return
		}

		// Check if user already owns this MVP item
		for _, inventoryItem := range userInventory {
			if inventoryItem.ShopItemID == req.ItemID && inventoryItem.Quantity > 0 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"error":   "You already own this item - it can only be purchased once",
				})
				return
			}
		}

		// Enforce quantity = 1 for unique items
		if req.Quantity != 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "This item is limited to 1 per account",
			})
			return
		}
	}

	// Calculate total cost
	totalCost := item.Price * req.Quantity

	// Check if user has enough credits
	if user.Credits < totalCost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Insufficient credits",
		})
		return
	}

	// Process purchase
	err = s.repo.PurchaseItem(user.ID, req.ItemID, req.Quantity, totalCost)
	if err != nil {
		log.Printf("Failed to process purchase: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to process purchase",
		})
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Successfully purchased %s!", item.Name),
	})
}

// handleApplyEffect handles applying bless/curse effects to fighters
func (s *Server) handleApplyEffect(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse JSON request
	var req struct {
		ItemID     int    `json:"item_id"`
		FighterID  int    `json:"fighter_id"`
		TargetType string `json:"target_type"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Printf("Failed to decode apply effect request: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate target type
	if req.TargetType != "fighter" {
		http.Error(w, "Invalid target type", http.StatusBadRequest)
		return
	}

	// Get user's inventory item
	userInventory, err := s.repo.GetUserInventory(user.ID)
	if err != nil {
		log.Printf("Failed to get user inventory: %v", err)
		http.Error(w, "Failed to access inventory", http.StatusInternalServerError)
		return
	}

	// Find the item in inventory
	var inventoryItem *database.UserInventoryItem
	for _, item := range userInventory {
		if item.ShopItemID == req.ItemID {
			inventoryItem = &item
			break
		}
	}

	if inventoryItem == nil || inventoryItem.Quantity <= 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Item not found in inventory",
		})
		return
	}

	// Validate item type
	if inventoryItem.ItemType != "fighter_curse" && inventoryItem.ItemType != "fighter_blessing" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid item type for this action",
		})
		return
	}

	// Use the inventory item
	err = s.repo.UseInventoryItem(user.ID, req.ItemID, 1)
	if err != nil {
		log.Printf("Failed to use inventory item: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to use item",
		})
		return
	}

	// Apply the effect
	err = s.repo.ApplyEffect(user.ID, req.TargetType, req.FighterID, inventoryItem.ItemType, inventoryItem.EffectValue)
	if err != nil {
		log.Printf("Failed to apply effect: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to apply effect",
		})
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Successfully applied %s!", inventoryItem.Name),
	})
}

// handleCreateFighter displays the fighter creation wizard
func (s *Server) handleCreateFighter(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth", http.StatusSeeOther)
		return
	}

	// Check if user has a Combat License
	userInventory, err := s.repo.GetUserInventory(user.ID)
	if err != nil {
		log.Printf("Error getting user inventory: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	hasLicense := false
	for _, item := range userInventory {
		if item.ItemType == "fighter_creation" && item.Quantity > 0 {
			hasLicense = true
			break
		}
	}

	if !hasLicense {
		// Redirect to shop with error message
		http.Redirect(w, r, "/shop?error=no_license", http.StatusSeeOther)
		return
	}

	// Generate colors for user
	primaryColor, secondaryColor := utils.GenerateUserColors(user.DiscordID)

	data := PageData{
		User:           user,
		Title:          "Create Your Fighter",
		PrimaryColor:   primaryColor,
		SecondaryColor: secondaryColor,
		RequiredCSS:    []string{"create-fighter.css"},
	}

	s.renderTemplate(w, "create-fighter.html", data)
}

// handleCreateFighterPost processes the fighter creation form
func (s *Server) handleCreateFighterPost(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Unauthorized",
		})
		return
	}

	// Parse JSON request
	var req struct {
		Name  string `json:"name"`
		Stats struct {
			Strength  int `json:"strength"`
			Speed     int `json:"speed"`
			Endurance int `json:"endurance"`
			Technique int `json:"technique"`
		} `json:"stats"`
		ChaosStats struct {
			BloodType        string  `json:"bloodType"`
			Horoscope        string  `json:"horoscope"`
			FighterClass     string  `json:"fighterClass"`
			Fingers          int     `json:"fingers"`
			Toes             int     `json:"toes"`
			MolecularDensity float64 `json:"molecularDensity"`
		} `json:"chaosStats"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Printf("Failed to decode fighter creation request: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	// Validate fighter name
	if len(req.Name) < 3 || len(req.Name) > 50 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Fighter name must be between 3 and 50 characters",
		})
		return
	}

	// Validate combat stats
	totalStats := req.Stats.Strength + req.Stats.Speed + req.Stats.Endurance + req.Stats.Technique
	if totalStats != 300 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Combat stats must total exactly 300 points (got %d)", totalStats),
		})
		return
	}

	// Validate individual stat ranges
	stats := []int{req.Stats.Strength, req.Stats.Speed, req.Stats.Endurance, req.Stats.Technique}
	for _, stat := range stats {
		if stat < 20 || stat > 120 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Each combat stat must be between 20 and 120",
			})
			return
		}
	}

	// Check if user has a Combat License
	userInventory, err := s.repo.GetUserInventory(user.ID)
	if err != nil {
		log.Printf("Error getting user inventory: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to check inventory",
		})
		return
	}

	hasLicense := false
	var licenseItemID int
	for _, item := range userInventory {
		if item.ItemType == "fighter_creation" && item.Quantity > 0 {
			hasLicense = true
			licenseItemID = item.ShopItemID
			break
		}
	}

	if !hasLicense {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "You need a Combat License to create a fighter",
		})
		return
	}

	// Check if name is already taken
	existingFighter, err := s.repo.GetFighterByName(req.Name)
	if err == nil && existingFighter != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Fighter name is already taken",
		})
		return
	}

	// Generate ALL chaos stats server-side (gacha system)
	ancestors := generateAncestors()
	existentialDread := generateExistentialDread()
	bloodType := generateBloodType()
	horoscope := generateHoroscope()
	molecularDensity := generateMolecularDensity()
	fingers := generateFingers()
	toes := generateToes()
	fighterClass := generateFighterClass()

	// Create the fighter
	now := time.Now()
	fighterID, err := s.repo.CreateCustomFighter(database.Fighter{
		Name:              req.Name,
		Team:              "Custom Fighters",
		Strength:          req.Stats.Strength,
		Speed:             req.Stats.Speed,
		Endurance:         req.Stats.Endurance,
		Technique:         req.Stats.Technique,
		BloodType:         bloodType,
		Horoscope:         horoscope,
		MolecularDensity:  molecularDensity,
		ExistentialDread:  existentialDread,
		Fingers:           fingers,
		Toes:              toes,
		Ancestors:         ancestors,
		FighterClass:      fighterClass,
		Wins:              0,
		Losses:            0,
		Draws:             0,
		IsDead:            false,
		CreatedByUserID:   &user.ID,
		IsCustom:          true,
		CreationDate:      &now,
		CustomDescription: nil, // Could be added later
	})

	if err != nil {
		log.Printf("Failed to create fighter: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to create fighter",
		})
		return
	}

	// Use the Combat License
	err = s.repo.UseInventoryItem(user.ID, licenseItemID, 1)
	if err != nil {
		log.Printf("Failed to use Combat License: %v", err)
		// Fighter was created but license wasn't consumed - this is not ideal but not critical
		// We could implement a rollback here if needed
	}

	log.Printf("User %s created custom fighter '%s' (ID: %d)", user.Username, req.Name, fighterID)

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"fighter_id": fighterID,
		"message":    fmt.Sprintf("Successfully created %s! Your fighter will join the violence arena soon.", req.Name),
	})
}

// Helper functions for generating additional chaos stats
func generateAncestors() int {
	// Generate a random number of ancestors (typically absurd numbers)
	ranges := [][]int{
		{0, 5},            // 30% - Normal range
		{100, 500},        // 40% - Hundreds
		{1000, 5000},      // 20% - Thousands
		{10000, 50000},    // 9% - Tens of thousands
		{100000, 1000000}, // 1% - Ridiculous numbers
	}

	roll := rand.Float64()
	var selectedRange []int

	if roll < 0.30 {
		selectedRange = ranges[0]
	} else if roll < 0.70 {
		selectedRange = ranges[1]
	} else if roll < 0.90 {
		selectedRange = ranges[2]
	} else if roll < 0.99 {
		selectedRange = ranges[3]
	} else {
		selectedRange = ranges[4]
	}

	return selectedRange[0] + rand.Intn(selectedRange[1]-selectedRange[0]+1)
}

func generateExistentialDread() int {
	// Generate existential dread level (0-100, with higher numbers being more common)
	// This creates a weighted distribution favoring higher dread levels
	roll := rand.Float64()

	if roll < 0.05 {
		return rand.Intn(20) // 0-19 (very low dread)
	} else if roll < 0.20 {
		return 20 + rand.Intn(30) // 20-49 (low dread)
	} else if roll < 0.50 {
		return 50 + rand.Intn(30) // 50-79 (medium dread)
	} else {
		return 80 + rand.Intn(21) // 80-100 (high dread)
	}
}

func generateBloodType() string {
	// First determine rarity
	rarity := generateRarity()

	switch rarity {
	case "legendary":
		types := []string{"Quantum Uncertainty", "The Void Itself", "Pure Determination", "Concentrated Chaos"}
		return types[rand.Intn(len(types))]
	case "rare":
		types := []string{"Monday Morning", "Imposter Syndrome", "Social Anxiety", "Main Character Syndrome", "Cryptocurrency Believer"}
		return types[rand.Intn(len(types))]
	case "uncommon":
		types := []string{"Caffeinated", "Meme Energy", "Discord Moderator", "User-Generated", "Community Spirit"}
		return types[rand.Intn(len(types))]
	default: // common
		types := []string{"A+", "B+", "AB+", "O+", "A-", "B-", "AB-", "O-", "Nacho Cheese", "Diet Coke"}
		return types[rand.Intn(len(types))]
	}
}

func generateHoroscope() string {
	rarity := generateRarity()

	switch rarity {
	case "legendary":
		signs := []string{"Quantum Entanglement", "The Singularity", "Heat Death", "Big Bang Redux"}
		return signs[rand.Intn(len(signs))]
	case "rare":
		signs := []string{"Reply Guy", "Oversharer", "LinkedIn Influencer", "Discord Admin", "Reddit Moderator"}
		return signs[rand.Intn(len(signs))]
	case "uncommon":
		signs := []string{"Algorithm", "Notification", "Blue Checkmark", "WiFi Signal", "Battery Low"}
		return signs[rand.Intn(len(signs))]
	default: // common
		signs := []string{"Aries", "Taurus", "Gemini", "Cancer", "Leo", "Virgo", "Libra", "Scorpio", "Sagittarius", "Capricorn", "Aquarius", "Pisces"}
		return signs[rand.Intn(len(signs))]
	}
}

func generateMolecularDensity() float64 {
	rarity := generateRarity()

	switch rarity {
	case "legendary":
		// Extreme values
		if rand.Float64() < 0.5 {
			return 0.1
		} else {
			return 99.9
		}
	case "rare":
		// Very low or very high
		if rand.Float64() < 0.5 {
			return rand.Float64() * 10 // 0-10
		} else {
			return 90 + rand.Float64()*9.9 // 90-99.9
		}
	case "uncommon":
		// Somewhat extreme
		if rand.Float64() < 0.5 {
			return 10 + rand.Float64()*20 // 10-30
		} else {
			return 70 + rand.Float64()*20 // 70-90
		}
	default: // common
		return 10 + rand.Float64()*80 // 10-90
	}
}

func generateFingers() int {
	rarity := generateRarity()

	switch rarity {
	case "legendary":
		// Impossible finger counts
		extremes := []int{0, 1, 25, 30, 50, 100}
		return extremes[rand.Intn(len(extremes))]
	case "rare":
		// Very weird but not impossible
		return rand.Intn(21) // 0-20
	case "uncommon":
		// Slightly off normal
		if rand.Float64() < 0.5 {
			return rand.Intn(2) + 6 // 6-7
		} else {
			return rand.Intn(3) + 13 // 13-15
		}
	default: // common
		// Mostly normal with slight variation
		return rand.Intn(5) + 8 // 8-12
	}
}

func generateToes() int {
	rarity := generateRarity()

	switch rarity {
	case "legendary":
		extremes := []int{0, 1, 25, 30, 50, 100}
		return extremes[rand.Intn(len(extremes))]
	case "rare":
		return rand.Intn(21) // 0-20
	case "uncommon":
		if rand.Float64() < 0.5 {
			return rand.Intn(2) + 6 // 6-7
		} else {
			return rand.Intn(3) + 13 // 13-15
		}
	default: // common
		return rand.Intn(5) + 8 // 8-12
	}
}

func generateFighterClass() string {
	rarity := generateRarity()

	switch rarity {
	case "legendary":
		classes := []string{"Reality Bender", "Concept Destroyer", "Existence Negator", "Universe Ender"}
		return classes[rand.Intn(len(classes))]
	case "rare":
		classes := []string{"Meme Lord", "Chaos Agent", "Discord Mod", "Reply Guy", "Karen"}
		return classes[rand.Intn(len(classes))]
	case "uncommon":
		classes := []string{"Community-Forged", "User-Defined", "Bespoke Violence", "Artisanal Combat"}
		return classes[rand.Intn(len(classes))]
	default: // common
		classes := []string{"Crowdsourced Chaos", "Democratic Destruction", "Collaborative Carnage", "Vanilla Fighter", "Basic Brawler"}
		return classes[rand.Intn(len(classes))]
	}
}

func generateRarity() string {
	roll := rand.Float64()

	if roll < 0.70 {
		return "common" // 70%
	} else if roll < 0.90 {
		return "uncommon" // 20%
	} else if roll < 0.98 {
		return "rare" // 8%
	} else {
		return "legendary" // 2%
	}
}

// handleCasino serves the main casino lobby page
func (s *Server) handleCasino(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth", http.StatusSeeOther)
		return
	}

	// Hard gate: require High Roller Card (ItemType: "high_roller")
	hasHighRoller := false
	inv, err := s.repo.GetUserInventory(user.ID)
	if err == nil {
		for _, it := range inv {
			if it.ItemType == "high_roller" && it.Quantity > 0 {
				hasHighRoller = true
				break
			}
		}
	}
	if !hasHighRoller {
		// Render a cryptic guard page; no mention of any casino
		data := PageData{
			User:        user,
			Title:       "Private Area - Authorized Personnel Only",
			RequiredCSS: []string{"closed.css"},
		}
		// Use standard renderer with base layout
		s.renderTemplate(w, "gate.html", data)
		return
	}

	// Check if it's Sunday and assign VIP role if they don't have it
	centralTime, _ := time.LoadLocation("America/Chicago")
	now := time.Now().In(centralTime)
	if now.Weekday() == time.Sunday {
		// Get the role manager from the scheduler's engine
		engine := s.scheduler.GetEngine()
		if engine != nil && engine.GetRoleManager() != nil {
			go func() {
				err := engine.GetRoleManager().AssignVIPRole(user)
				if err != nil {
					log.Printf("Failed to assign VIP role to %s: %v", user.Username, err)
				}
			}()
		}
	}

	// Determine casino bet cap (100M unless user has >=1000 sacrifices)
	casinoCap := 100000000
	inv2, _ := s.repo.GetUserInventory(user.ID)
	for _, it := range inv2 {
		if it.ItemType == "sacrifice" && it.Quantity >= 1000 {
			casinoCap = 0 // 0 means unlimited for our client-side logic; backend will skip cap if 0
			break
		}
	}

	data := PageData{
		User:         user,
		Title:        "Underground Casino",
		RequiredCSS:  []string{"casino.css"},
		CasinoBetMax: casinoCap,
	}

	// Render casino template directly (not through base template)
	tmpl, err := template.ParseFiles("templates/casino.html")
	if err != nil {
		log.Printf("Casino template parsing error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = tmpl.Execute(w, data)
	if err != nil {
		log.Printf("Casino template execution error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleMoonFlip processes moon phase coin flip bets
func (s *Server) handleMoonFlip(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 1% random extortion event
	if roll := rand.Intn(100); roll == 0 {
		log.Printf("[Extortion] Triggered for user %d (moonflip) roll=%d", user.ID, roll)
		s.respondWithExtortion(w, user)
		return
	}

	// Parse JSON request
	var req struct {
		Amount int    `json:"amount"`
		Choice string `json:"choice"` // "full" or "new"
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	// Validate inputs
	if req.Amount <= 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid bet amount",
		})
		return
	}

	if req.Choice != "full" && req.Choice != "new" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid choice",
		})
		return
	}

	// Check if user has sufficient credits
	if req.Amount > user.Credits {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Insufficient credits",
		})
		return
	}

	// Enforce casino cap (100M unless user has >=1000 sacrifices)
	if !s.userHasSacrificeExemption(user.ID) && req.Amount > 100000000 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Max bet is 100,000,000",
		})
		return
	}

	// Generate random result (server-side)
	results := []string{"full", "new"}
	result := results[rand.Intn(len(results))]

	// Determine win/loss
	won := result == req.Choice
	var newBalance int
	var payout int

	if won {
		payout = req.Amount * 2 // 2x payout
		newBalance = user.Credits + payout - req.Amount
	} else {
		newBalance = user.Credits - req.Amount
	}

	// Update user credits
	err = s.repo.UpdateUserCredits(user.ID, newBalance)
	if err != nil {
		log.Printf("Failed to update user credits: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to process bet",
		})
		return
	}

	// Return result
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"result":      result,
		"won":         won,
		"payout":      payout,
		"new_balance": newBalance,
		"choice":      req.Choice,
		"amount":      req.Amount,
	})
}

// handleHiLowStep1 processes the first step of the Hi-Low game (betting)
func (s *Server) handleHiLowStep1(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if roll := rand.Intn(100); roll == 0 {
		log.Printf("[Extortion] Triggered for user %d (hilow step1) roll=%d", user.ID, roll)
		s.respondWithExtortion(w, user)
		return
	}

	// Parse JSON request
	var req struct {
		Amount int `json:"amount"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Printf("Failed to decode Hi-Low step 1 request: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	// Validate amount
	if req.Amount <= 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid bet amount",
		})
		return
	}

	// Check if user has sufficient credits
	if req.Amount > user.Credits {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Insufficient credits",
		})
		return
	}

	// Enforce casino cap
	if !s.userHasSacrificeExemption(user.ID) && req.Amount > 100000000 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Max bet is 100,000,000",
		})
		return
	}

	// SECURITY: Charge the user immediately and generate first card
	newBalance := user.Credits - req.Amount
	err = s.repo.UpdateUserCredits(user.ID, newBalance)
	if err != nil {
		log.Printf("Failed to deduct bet amount for Hi-Low step 1: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to process bet",
		})
		return
	}

	// Generate first card server-side AFTER payment
	suits := []string{"♠️", "♥️", "♦️", "♣️"}
	values := []string{"A", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"}

	suit := suits[rand.Intn(len(suits))]
	value := values[rand.Intn(len(values))]
	firstCard := value + suit

	// Store the bet info in user session for step 2
	// For simplicity, we'll use a simple approach with database or session
	// For now, we'll return the first card and amount

	// Return the first card and amount
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"first_card":  firstCard,
		"amount":      req.Amount,
		"new_balance": newBalance,
	})
}

// handleHiLowStep2 processes the second step of the Hi-Low game (guessing)
func (s *Server) handleHiLowStep2(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse JSON request
	var req struct {
		Guess     string `json:"guess"`      // "hi" or "low"
		FirstCard string `json:"first_card"` // The first card from step 1
		Amount    int    `json:"amount"`     // The bet amount from step 1
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Printf("Failed to decode Hi-Low step 2 request: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	// Validate guess
	if req.Guess != "hi" && req.Guess != "low" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid guess",
		})
		return
	}

	// Validate inputs
	if req.FirstCard == "" || req.Amount <= 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Missing bet information",
		})
		return
	}

	// Generate second card server-side
	suits := []string{"♠️", "♥️", "♦️", "♣️"}
	values := []string{"A", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"}

	suit := suits[rand.Intn(len(suits))]
	value := values[rand.Intn(len(values))]
	secondCard := value + suit

	// Convert card values to numbers for comparison
	getCardValue := func(cardStr string) int {
		// Extract value part (everything before the suit emoji)
		valueStr := ""
		for _, char := range cardStr {
			if char != '♠' && char != '♥' && char != '♦' && char != '♣' && char != '️' {
				valueStr += string(char)
			} else {
				break
			}
		}

		switch valueStr {
		case "A":
			return 1
		case "J":
			return 11
		case "Q":
			return 12
		case "K":
			return 13
		default:
			val, _ := strconv.Atoi(valueStr)
			return val
		}
	}

	firstValue := getCardValue(req.FirstCard)
	secondValue := getCardValue(secondCard)

	// Determine win/loss
	var won bool
	if req.Guess == "hi" {
		won = secondValue > firstValue
	} else {
		won = secondValue < firstValue
	}

	// Handle ties (second card same value as first)
	if secondValue == firstValue {
		// Ties are losses for the player
		won = false
	}

	var newBalance int
	var payout int

	if won {
		payout = req.Amount * 2 // 2x payout
		newBalance = user.Credits + payout
	} else {
		newBalance = user.Credits // No additional charge, already paid in step 1
	}

	// Update user credits (only if they won)
	if won {
		err = s.repo.UpdateUserCredits(user.ID, newBalance)
		if err != nil {
			log.Printf("Failed to update user credits for Hi-Low step 2: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Failed to process winnings",
			})
			return
		}
	}

	// Return result
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"first_card":   req.FirstCard,
		"second_card":  secondCard,
		"won":          won,
		"payout":       payout,
		"new_balance":  newBalance,
		"guess":        req.Guess,
		"amount":       req.Amount,
		"first_value":  firstValue,
		"second_value": secondValue,
	})
}

// handleSlots processes emoji slot machine spins with server-side sequence generation
func (s *Server) handleSlots(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if roll := rand.Intn(100); roll == 0 {
		log.Printf("[Extortion] Triggered for user %d (slots) roll=%d", user.ID, roll)
		s.respondWithExtortion(w, user)
		return
	}

	// Parse JSON request
	var req struct {
		Amount int `json:"amount"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Printf("Failed to decode slots request: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	// Validate amount
	if req.Amount <= 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid bet amount",
		})
		return
	}

	// Check if user has sufficient credits
	if req.Amount > user.Credits {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Insufficient credits",
		})
		return
	}

	// Enforce casino cap
	if !s.userHasSacrificeExemption(user.ID) && req.Amount > 100000000 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Max bet is 100,000,000",
		})
		return
	}

	// Define slot machine emojis (server-side only)
	emojis := []string{"🍎", "🍊", "🍋", "🍌", "🍇", "🍓", "🥝", "🍑"}

	// Generate 4-5 sequences for animation (server-side RNG)
	numSequences := 4 + rand.Intn(2) // 4 or 5 sequences
	sequences := make([][]string, numSequences)

	for i := 0; i < numSequences; i++ {
		sequence := make([]string, 9) // 3x3 grid = 9 positions
		for j := 0; j < 9; j++ {
			sequence[j] = emojis[rand.Intn(len(emojis))]
		}
		sequences[i] = sequence
	}

	// Generate final grid (the one we score on)
	finalGrid := make([]string, 9)
	for i := 0; i < 9; i++ {
		finalGrid[i] = emojis[rand.Intn(len(emojis))]
	}

	// Check for winning lines (horizontal rows only)
	winningLines := checkWinningLines(finalGrid)
	won := len(winningLines) > 0

	// Calculate payout based on winning lines
	var payout int
	var newBalance int

	if won {
		// Get current progressive jackpot
		jackpotAmount, err := s.getProgressiveJackpot()
		if err != nil {
			log.Printf("Failed to get progressive jackpot: %v", err)
			jackpotAmount = 0
		}

		if len(winningLines) == 3 {
			// JACKPOT! Pay out the progressive jackpot + base payout
			basePayout := req.Amount * 6 // 6x base for 3 lines
			payout = basePayout + jackpotAmount
			newBalance = user.Credits + payout - req.Amount

			// Reset the progressive jackpot to a base amount
			err = s.setProgressiveJackpot(1000) // Reset to 1000 credits
			if err != nil {
				log.Printf("Failed to reset progressive jackpot: %v", err)
			}
		} else {
			// Regular payouts
			if len(winningLines) == 1 {
				payout = req.Amount * 10 // 10x bet
			} else if len(winningLines) == 2 {
				payout = req.Amount * 50 // 50x bet
			}
			newBalance = user.Credits + payout - req.Amount
		}
	} else {
		// Player lost - add 90% of their bet to the progressive jackpot
		newBalance = user.Credits - req.Amount
		jackpotContribution := (req.Amount * 9) / 10 // 90% of bet goes to jackpot
		if jackpotContribution < 1 {
			jackpotContribution = 1 // Minimum 1 credit contribution
		}

		currentJackpot, err := s.getProgressiveJackpot()
		if err != nil {
			log.Printf("Failed to get current jackpot for contribution: %v", err)
			currentJackpot = 1000 // Default starting jackpot
		}

		err = s.setProgressiveJackpot(currentJackpot + jackpotContribution)
		if err != nil {
			log.Printf("Failed to update progressive jackpot: %v", err)
		}
	}

	// Update user credits
	err = s.repo.UpdateUserCredits(user.ID, newBalance)
	if err != nil {
		log.Printf("Failed to update user credits for slots: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to process bet",
		})
		return
	}

	// Ensure winning_lines is never null - always return an array
	if winningLines == nil {
		winningLines = []int{}
	}

	// Return result with sequences for animation
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       true,
		"sequences":     sequences,
		"final_grid":    finalGrid,
		"winning_lines": winningLines,
		"won":           won,
		"payout":        payout,
		"amount":        req.Amount,
		"new_balance":   newBalance,
	})
}

// checkWinningLines checks for horizontal lines of 3 consecutive identical emojis
func checkWinningLines(grid []string) []int {
	var winningLines []int

	// Check each horizontal row (3 rows total)
	for row := 0; row < 3; row++ {
		// Get the 3 emojis in this row
		emoji1 := grid[row*3+0]
		emoji2 := grid[row*3+1]
		emoji3 := grid[row*3+2]

		// Check if all 3 are the same
		if emoji1 == emoji2 && emoji2 == emoji3 {
			winningLines = append(winningLines, row)
		}
	}

	return winningLines
}

// handleUserProfile renders a public user profile page
func (s *Server) handleUserProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["username"]

	// Get the profile user
	profileUser, err := s.repo.GetUserByUsername(username)
	if err != nil {
		log.Printf("User not found: %v", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Get the viewing user (for navigation, not required)
	viewingUser := GetUserFromContext(r.Context())

	// Get profile user's betting history (last 50 bets)
	userBets, err := s.repo.GetUserBets(profileUser.ID)
	if err != nil {
		log.Printf("Error fetching user bets: %v", err)
		userBets = nil
	}

	// Get profile user's inventory
	userInventory, err := s.repo.GetUserInventory(profileUser.ID)
	if err != nil {
		log.Printf("Error fetching user inventory: %v", err)
		userInventory = nil
	}

	// Get profile user's betting stats
	bettingStats, err := s.repo.GetUserBettingStats(profileUser.ID)
	if err != nil {
		log.Printf("Error fetching betting stats: %v", err)
		bettingStats = nil
	}

	// Get profile user's MVP setting
	var currentMVP *database.UserSetting
	var mvpFighter *database.Fighter
	mvpSetting, err := s.repo.GetUserSetting(profileUser.ID, "mvp_player")
	if err == nil {
		currentMVP = mvpSetting
		// Get the fighter details for the MVP
		if currentMVP != nil && currentMVP.SettingValue != "" {
			fighterID, err := strconv.Atoi(currentMVP.SettingValue)
			if err == nil {
				mvpFighter, err = s.repo.GetFighter(fighterID)
				if err != nil {
					log.Printf("Error getting MVP fighter: %v", err)
					mvpFighter = nil
				}
			}
		}
	}

	// Generate colors for the profile user
	primaryColor, secondaryColor := utils.GenerateUserColors(profileUser.DiscordID)

	displayName := profileUser.CustomUsername
	if displayName == "" {
		displayName = profileUser.Username
	}

	data := PageData{
		User:            viewingUser, // Viewing user for navigation
		Title:           fmt.Sprintf("%s's Profile", displayName),
		Users:           []database.User{*profileUser}, // Profile user in Users[0]
		UserBets:        userBets,
		UserInventory:   userInventory,
		BettingStats:    bettingStats,
		CurrentMVP:      currentMVP,
		Fighter:         mvpFighter, // MVP fighter details
		PrimaryColor:    primaryColor,
		SecondaryColor:  secondaryColor,
		MetaDescription: fmt.Sprintf("🎮 %s's Violence Profile 🎮 View their betting history, inventory, and chaos statistics in the Department of Recreational Violence.", displayName),
		MetaType:        "profile",
		RequiredCSS:     []string{"profile.css"},
	}

	s.renderTemplate(w, "profile.html", data)
}

// renderTemplate parses templates fresh on each request for hot-reloading
func (s *Server) renderTemplate(w http.ResponseWriter, templateName string, data interface{}) {
	// Define template functions
	funcMap := template.FuncMap{
		"add": func(a, b interface{}) int64 {
			var aVal, bVal int64

			switch v := a.(type) {
			case int:
				aVal = int64(v)
			case int64:
				aVal = v
			default:
				aVal = 0
			}

			switch v := b.(type) {
			case int:
				bVal = int64(v)
			case int64:
				bVal = v
			default:
				bVal = 0
			}

			return aVal + bVal
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"mul": func(a, b int) int {
			return a * b
		},
		"mod": func(a, b int) int {
			return a % b
		},
		"generateUserColors": func(discordID string) []string {
			primary, secondary := utils.GenerateUserColors(discordID)
			return []string{primary, secondary}
		},
		"int64": func(i int64) int {
			return int(i)
		},
		"getDisplayName": func(username, customUsername string) string {
			if customUsername != "" {
				return customUsername
			}
			return username
		},
		"min": func(a, b int) int {
			if a < b {
				return a
			}
			return b
		},
	}

	// Parse base template and the specific template with functions
	tmpl, err := template.New("").Funcs(funcMap).ParseFiles("templates/base.html", "templates/"+templateName)
	if err != nil {
		log.Printf("Template parsing error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = tmpl.ExecuteTemplate(w, "base.html", data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (s *Server) Start(port string) error {
	// Enable CORS
	corsHandler := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"*"}),
	)

	// Remove LoggingHandler - we don't want nginx-style access logs in our application logs
	finalHandler := corsHandler(s.router)

	log.Printf("Starting web server on port %s", port)
	return http.ListenAndServe(":"+port, finalHandler)
}

// getProgressiveJackpot retrieves the current progressive jackpot amount
func (s *Server) getProgressiveJackpot() (int, error) {
	// Use a simple file-based storage for the progressive jackpot
	// In production, this could be stored in the database
	data, err := os.ReadFile("progressive_jackpot.txt")
	if err != nil {
		// File doesn't exist, return default starting jackpot
		return 1000, nil
	}

	jackpot, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		// Invalid data, return default
		return 1000, nil
	}

	return jackpot, nil
}

// setProgressiveJackpot sets the current progressive jackpot amount
func (s *Server) setProgressiveJackpot(amount int) error {
	return os.WriteFile("progressive_jackpot.txt", []byte(strconv.Itoa(amount)), 0644)
}

// handleGetJackpot returns the current progressive jackpot amount via API
func (s *Server) handleGetJackpot(w http.ResponseWriter, r *http.Request) {
	jackpot, err := s.getProgressiveJackpot()
	if err != nil {
		log.Printf("Failed to get progressive jackpot: %v", err)
		jackpot = 1000 // Default fallback
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"jackpot": jackpot,
	})
}

// -------------------- Extortion Event --------------------
// respondWithExtortion immediately charges 70% of the user's credits and returns
// a payload instructing the client to show an extortion modal. The client must
// call /user/casino/extortion with the user's choice to settle.
func (s *Server) respondWithExtortion(w http.ResponseWriter, user *database.User) {
	// If the user is sacrifice-exempt, show a blessing message instead of charging
	if s.userHasSacrificeExemption(user.ID) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":           false,
			"extortion_blessed": true,
			"message":           "The gods were on your side. A quiet hand steers you past two waiting hooligans.",
		})
		return
	}

	// Deduct 70% as a hold
	original := user.Credits
	hold := (original * 70) / 100
	newBalance := original - hold
	if newBalance < 0 {
		newBalance = 0
	}
	_ = s.repo.UpdateUserCredits(user.ID, newBalance)
	// Persist authoritative extortion context for secure settlement
	_ = s.repo.SetUserSetting(user.ID, "extortion_original", fmt.Sprintf("%d", original), nil)
	_ = s.repo.SetUserSetting(user.ID, "extortion_hold", fmt.Sprintf("%d", hold), nil)
	_ = s.repo.SetUserSetting(user.ID, "extortion_active", "1", nil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":          false,
		"extortion":        true,
		"message":          "A pair of goons corner you by the bathroom. They want 20% to 'keep the peace'. Pay up or run?",
		"hold":             hold,
		"new_balance":      newBalance,
		"original_balance": original,
		"fee_amount":       (original * 20) / 100,
	})
}

// handleExtortionResolve applies the user's choice after the 70% hold.
// Body: { choice: "pay"|"run" }
func (s *Server) handleExtortionResolve(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Choice string `json:"choice"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Invalid request"})
		return
	}

	// Compute amounts using authoritative original/hold stored at trigger time
	// If they choose pay: final = original - 20% → refund = (original - 20%) - (original - 70%) = 50%
	// If they run and fail: final = 50% → refund = 20%
	// If they run and succeed: final = 100% → refund = 70%
	origSetting, _ := s.repo.GetUserSetting(user.ID, "extortion_original")
	holdSetting, _ := s.repo.GetUserSetting(user.ID, "extortion_hold")
	activeSetting, _ := s.repo.GetUserSetting(user.ID, "extortion_active")
	if activeSetting == nil || activeSetting.SettingValue != "1" || origSetting == nil || holdSetting == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Extortion state missing or expired"})
		return
	}
	originalCredits, _ := strconv.Atoi(origSetting.SettingValue)
	heldCredits, _ := strconv.Atoi(holdSetting.SettingValue)
	var refund int
	var outcome string
	var message string
	switch req.Choice {
	case "pay":
		refund = (originalCredits * 50) / 100
		outcome = "paid"
		message = "You hand over the envelope. The room relaxes. Net loss: 20%."
	case "run":
		// coin flip
		if rand.Intn(2) == 0 {
			// fail → net 50% loss
			refund = (originalCredits * 20) / 100
			outcome = "run_fail"
			message = "You bolt. A meaty hand catches your collar. They keep half."
		} else {
			// succeed → refund all 70%
			refund = heldCredits
			outcome = "run_success"
			message = "You slip the grasp and vanish into the crowd. They get nothing."
		}
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Invalid choice"})
		return
	}

	finalBalance := user.Credits + refund
	if err := s.repo.UpdateUserCredits(user.ID, finalBalance); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Failed to settle"})
		return
	}

	// Clear extortion state
	_ = s.repo.SetUserSetting(user.ID, "extortion_active", "0", nil)
	_ = s.repo.SetUserSetting(user.ID, "extortion_original", "", nil)
	_ = s.repo.SetUserSetting(user.ID, "extortion_hold", "", nil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"new_balance": finalBalance,
		"refund":      refund,
		"outcome":     outcome,
		"message":     message,
	})
}

// -------------------- BLACKJACK (stateless, server-side RNG) --------------------

// handleBlackjackStart charges the bet and deals initial cards
func (s *Server) handleBlackjackStart(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if roll := rand.Intn(100); roll == 0 {
		log.Printf("[Extortion] Triggered for user %d (blackjack start) roll=%d", user.ID, roll)
		s.respondWithExtortion(w, user)
		return
	}

	var req struct {
		Amount int `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	if req.Amount <= 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid bet amount",
		})
		return
	}

	if req.Amount > user.Credits {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Insufficient credits",
		})
		return
	}

	// Enforce casino cap
	if !s.userHasSacrificeExemption(user.ID) && req.Amount > 100000000 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Max bet is 100,000,000",
		})
		return
	}

	// Charge immediately (like Hi-Low step 1)
	newBalance := user.Credits - req.Amount
	if err := s.repo.UpdateUserCredits(user.ID, newBalance); err != nil {
		log.Printf("Failed to deduct bet amount for Blackjack start: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to process bet",
		})
		return
	}

	// Deal initial hands (server-side RNG)
	suits := []string{"♠️", "♥️", "♦️", "♣️"}
	values := []string{"A", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"}
	dealCard := func() string {
		return values[rand.Intn(len(values))] + suits[rand.Intn(len(suits))]
	}

	player := []string{dealCard(), dealCard()}
	dealerUp := dealCard()

	// Build signed state token (bind to user ID)
	type bjState struct {
		UID        int      `json:"uid"`
		Amount     int      `json:"amount"`
		DealerUp   string   `json:"dealer_up"`
		PlayerHand []string `json:"player_hand"`
		Step       string   `json:"step"`
		TS         int64    `json:"ts"`
	}
	st := bjState{UID: user.ID, Amount: req.Amount, DealerUp: dealerUp, PlayerHand: player, Step: "inplay", TS: time.Now().Unix()}
	payload, _ := json.Marshal(st)
	dataB64 := base64.StdEncoding.EncodeToString(payload)
	secret := []byte(os.Getenv("SESSION_SECRET"))
	sig := s.signBytes(secret, payload)
	state := signedState{Data: dataB64, Sig: sig}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       true,
		"player_hand":   player,
		"dealer_upcard": dealerUp,
		"amount":        req.Amount,
		"new_balance":   newBalance,
		"state":         state,
	})
}

// handleBlackjackHit deals one more card to the player and reports bust or continue
func (s *Server) handleBlackjackHit(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// extortion can also trigger mid-hand
	if roll := rand.Intn(100); roll == 0 {
		log.Printf("[Extortion] Triggered for user %d (blackjack hit) roll=%d", user.ID, roll)
		s.respondWithExtortion(w, user)
		return
	}

	var req struct {
		State signedState `json:"state"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	// Verify token
	secret := []byte(os.Getenv("SESSION_SECRET"))
	payloadBytes, err := base64.StdEncoding.DecodeString(req.State.Data)
	if err != nil || !s.verifyBytes(secret, payloadBytes, req.State.Sig) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Bad state token"})
		return
	}
	var st struct {
		UID        int      `json:"uid"`
		Amount     int      `json:"amount"`
		DealerUp   string   `json:"dealer_up"`
		PlayerHand []string `json:"player_hand"`
		Step       string   `json:"step"`
		TS         int64    `json:"ts"`
	}
	if err := json.Unmarshal(payloadBytes, &st); err != nil || st.UID != user.ID || st.Amount <= 0 || st.Step != "inplay" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Invalid state"})
		return
	}

	// Server deals a random card
	suits := []string{"♠️", "♥️", "♦️", "♣️"}
	values := []string{"A", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"}
	newCard := values[rand.Intn(len(values))] + suits[rand.Intn(len(suits))]
	player := append(append([]string{}, st.PlayerHand...), newCard)

	// Compute value with Aces as 11/1
	getCardVal := func(card string) (int, bool) {
		valueStr := ""
		for _, ch := range card {
			if ch != '♠' && ch != '♥' && ch != '♦' && ch != '♣' && ch != '️' {
				valueStr += string(ch)
			} else {
				break
			}
		}
		switch valueStr {
		case "A":
			return 11, true
		case "K", "Q", "J":
			return 10, false
		default:
			val, _ := strconv.Atoi(valueStr)
			return val, false
		}
	}

	calc := func(hand []string) (int, bool) {
		sum := 0
		aces := 0
		for _, c := range hand {
			v, isAce := getCardVal(c)
			sum += v
			if isAce {
				aces++
			}
		}
		for sum > 21 && aces > 0 {
			sum -= 10
			aces--
		}
		return sum, sum <= 21
	}

	playerTotal, ok := calc(player)
	bust := !ok

	// Build new signed state if not bust
	var nextState *signedState
	if !bust {
		st.PlayerHand = player
		st.TS = time.Now().Unix()
		payload, _ := json.Marshal(st)
		dataB64 := base64.StdEncoding.EncodeToString(payload)
		sig := s.signBytes(secret, payload)
		ns := signedState{Data: dataB64, Sig: sig}
		nextState = &ns
	}

	w.Header().Set("Content-Type", "application/json")
	resp := map[string]interface{}{
		"success":      true,
		"player_hand":  player,
		"new_card":     newCard,
		"player_total": playerTotal,
		"bust":         bust,
	}
	if nextState != nil {
		resp["state"] = nextState
	}
	json.NewEncoder(w).Encode(resp)
}

// handleBlackjackStand resolves the round by drawing dealer cards and paying out
func (s *Server) handleBlackjackStand(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if roll := rand.Intn(100); roll == 0 {
		log.Printf("[Extortion] Triggered for user %d (blackjack stand) roll=%d", user.ID, roll)
		s.respondWithExtortion(w, user)
		return
	}

	var req struct {
		State signedState `json:"state"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}
	// Verify token
	secret := []byte(os.Getenv("SESSION_SECRET"))
	payloadBytes, err := base64.StdEncoding.DecodeString(req.State.Data)
	if err != nil || !s.verifyBytes(secret, payloadBytes, req.State.Sig) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Bad state token"})
		return
	}
	var st struct {
		UID        int      `json:"uid"`
		Amount     int      `json:"amount"`
		DealerUp   string   `json:"dealer_up"`
		PlayerHand []string `json:"player_hand"`
		Step       string   `json:"step"`
		TS         int64    `json:"ts"`
	}
	if err := json.Unmarshal(payloadBytes, &st); err != nil || st.UID != user.ID || st.Amount <= 0 || st.Step != "inplay" || len(st.PlayerHand) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Invalid state"})
		return
	}

	// Helpers
	getCardVal := func(card string) (int, bool) {
		valueStr := ""
		for _, ch := range card {
			if ch != '♠' && ch != '♥' && ch != '♦' && ch != '♣' && ch != '️' {
				valueStr += string(ch)
			} else {
				break
			}
		}
		switch valueStr {
		case "A":
			return 11, true
		case "K", "Q", "J":
			return 10, false
		default:
			val, _ := strconv.Atoi(valueStr)
			return val, false
		}
	}

	calc := func(hand []string) (int, bool) {
		sum := 0
		aces := 0
		for _, c := range hand {
			v, isAce := getCardVal(c)
			sum += v
			if isAce {
				aces++
			}
		}
		for sum > 21 && aces > 0 {
			sum -= 10
			aces--
		}
		return sum, sum <= 21
	}

	// Build dealer hand starting from upcard
	suits := []string{"♠️", "♥️", "♦️", "♣️"}
	values := []string{"A", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"}
	dealCard := func() string {
		return values[rand.Intn(len(values))] + suits[rand.Intn(len(suits))]
	}

	dealer := []string{st.DealerUp, dealCard()}
	dealerTotal, _ := calc(dealer)
	for dealerTotal < 17 {
		dealer = append(dealer, dealCard())
		dealerTotal, _ = calc(dealer)
	}

	playerTotal, playerOk := calc(st.PlayerHand)
	if !playerOk {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":      true,
			"dealer_hand":  dealer,
			"dealer_total": dealerTotal,
			"player_total": playerTotal,
			"won":          false,
			"push":         false,
			"payout":       0,
			"new_balance":  user.Credits,
			"amount":       st.Amount,
		})
		return
	}

	var won, push bool
	if dealerTotal > 21 || playerTotal > dealerTotal {
		won = true
		push = false
	} else if playerTotal == dealerTotal {
		won = false
		push = true
	} else {
		won = false
		push = false
	}

	newBalance := user.Credits
	payout := 0
	if won {
		payout = st.Amount * 2
		newBalance = user.Credits + payout
		if err := s.repo.UpdateUserCredits(user.ID, newBalance); err != nil {
			log.Printf("Failed to pay Blackjack winnings: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Failed to process winnings",
			})
			return
		}
	} else if push {
		payout = st.Amount
		newBalance = user.Credits + payout
		if err := s.repo.UpdateUserCredits(user.ID, newBalance); err != nil {
			log.Printf("Failed to refund Blackjack push: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Failed to process refund",
			})
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"dealer_hand":  dealer,
		"dealer_total": dealerTotal,
		"player_total": playerTotal,
		"won":          won,
		"push":         push,
		"payout":       payout,
		"new_balance":  newBalance,
		"amount":       st.Amount,
	})
}
