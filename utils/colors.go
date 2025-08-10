package utils

import (
	"crypto/md5"
	"fmt"
	"strconv"
)

// GenerateUserColors creates two hex colors from a Discord ID
// Returns primary color and secondary color, both guaranteed to be visible on black
func GenerateUserColors(discordID string) (string, string) {
	// Create MD5 hash of Discord ID
	hash := md5.Sum([]byte(discordID))
	hashHex := fmt.Sprintf("%x", hash)

	// Take first 12 chars and split into two 6-char colors
	color1Raw := hashHex[:6]
	color2Raw := hashHex[6:12]

	// Ensure colors are bright enough for black background
	color1 := ensureBrightness(color1Raw)
	color2 := ensureBrightness(color2Raw)

	return color1, color2
}

// ensureBrightness modifies a hex color to ensure it's visible on black background
func ensureBrightness(hexColor string) string {
	// Convert hex to RGB
	r, _ := strconv.ParseInt(hexColor[0:2], 16, 64)
	g, _ := strconv.ParseInt(hexColor[2:4], 16, 64)
	b, _ := strconv.ParseInt(hexColor[4:6], 16, 64)

	// Calculate brightness (perceived luminance)
	brightness := (r*299 + g*587 + b*114) / 1000

	// If too dark, brighten it by adding to each channel
	if brightness < 128 {
		// Add enough to make it bright
		boost := int64(128 - brightness + 50) // Extra 50 for safety
		r = min(255, r+boost)
		g = min(255, g+boost)
		b = min(255, b+boost)
	}

	return fmt.Sprintf("%02x%02x%02x", r, g, b)
}

// GetDisplayName returns custom username if set, otherwise Discord username
func GetDisplayName(user *User) string {
	if user.CustomUsername != "" {
		return user.CustomUsername
	}
	return user.Username
}

// User struct reference for the function above
type User struct {
	Username       string
	CustomUsername string
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
