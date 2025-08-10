package web

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"spoodblort/database"
	"time"

	"github.com/gorilla/sessions"
)

type ContextKey string

const UserContextKey ContextKey = "user"

type AuthMiddleware struct {
	repo  *database.Repository
	store *sessions.CookieStore
}

func NewAuthMiddleware(repo *database.Repository, sessionSecret string) *AuthMiddleware {
	store := sessions.NewCookieStore([]byte(sessionSecret))

	// Determine if we're in production based on environment
	isProduction := os.Getenv("ENVIRONMENT") == "production"

	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   9999 * 24 * 60 * 60, // 9999 days
		HttpOnly: true,
		Secure:   isProduction, // Only secure cookies in production (HTTPS)
		SameSite: http.SameSiteLaxMode,
	}

	return &AuthMiddleware{
		repo:  repo,
		store: store,
	}
}

// LoadUser middleware checks for session token and loads user into context
func (am *AuthMiddleware) LoadUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := am.store.Get(r, "spoodblort-session")
		if err != nil {
			// Invalid session, continue without user
			next.ServeHTTP(w, r)
			return
		}

		token, ok := session.Values["token"].(string)
		if !ok || token == "" {
			// No token in session
			next.ServeHTTP(w, r)
			return
		}

		// Check token in database
		user, err := am.repo.GetUserBySessionToken(token)
		if err != nil {
			// Invalid or expired token, clean up session
			delete(session.Values, "token")
			session.Save(r, w)
			next.ServeHTTP(w, r)
			return
		}

		// Add user to request context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAuth middleware redirects to login if user not authenticated
func (am *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUserFromContext(r.Context())
		if user == nil {
			http.Redirect(w, r, "/auth", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// CreateSession creates a new session for a user
func (am *AuthMiddleware) CreateSession(w http.ResponseWriter, r *http.Request, userID int) error {
	// Generate random session token
	token, err := generateSessionToken()
	if err != nil {
		return err
	}

	// Store token in database with expiration
	expiresAt := time.Now().Add(30 * 24 * time.Hour) // 30 days
	err = am.repo.CreateSession(token, userID, expiresAt)
	if err != nil {
		return err
	}

	// Store token in cookie session
	session, err := am.store.Get(r, "spoodblort-session")
	if err != nil {
		return err
	}

	session.Values["token"] = token
	return session.Save(r, w)
}

// DestroySession logs out a user
func (am *AuthMiddleware) DestroySession(w http.ResponseWriter, r *http.Request) error {
	session, err := am.store.Get(r, "spoodblort-session")
	if err != nil {
		return err
	}

	// Remove token from database if it exists
	if token, ok := session.Values["token"].(string); ok && token != "" {
		am.repo.DeleteSession(token)
	}

	// Clear session
	session.Values = make(map[interface{}]interface{})
	session.Options.MaxAge = -1
	return session.Save(r, w)
}

// GetUserFromContext extracts user from request context
func GetUserFromContext(ctx context.Context) *database.User {
	user, ok := ctx.Value(UserContextKey).(*database.User)
	if !ok {
		return nil
	}
	return user
}

// generateSessionToken creates a cryptographically secure random token
func generateSessionToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
