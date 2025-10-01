package wiki

import (
	"fmt"
	"regexp"
	"strings"

	"spoodblort/database"
)

var nonTitleChars = regexp.MustCompile(`[^A-Za-z0-9 _-]`)

func fighterDisplayTitle(f database.Fighter) string {
	return fmt.Sprintf("Roster #%03d %s", f.ID, f.Name)
}

func sanitizeName(name string) string {
	name = nonTitleChars.ReplaceAllString(name, "_")
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, " ", "_")
	return name
}

// FighterPageTitle returns the canonical page title used by the legacy scripts
// e.g. Roster_005_Lady_Chaos
func FighterPageTitle(f database.Fighter) string {
	return fmt.Sprintf("Roster_%03d_%s", f.ID, sanitizeName(f.Name))
}

// BuildFighterPageText renders the full wikitext for a fighter page
// aligned with the legacy bash scripts.
func BuildFighterPageText(f database.Fighter) string {
	team := f.Team
	if strings.TrimSpace(team) == "" {
		team = "Unaffiliated"
	}
	className := f.FighterClass
	if strings.TrimSpace(className) == "" {
		className = "Unclassified"
	}

	// Prevent template pipe conflicts inside lore
	lore := strings.ReplaceAll(f.Lore, "|", "{{!}}")
	loreMarkdown := strings.TrimSpace(lore)
	if loreMarkdown == "" {
		loreMarkdown = "This fighter's lore file is missing. Please consult the Department of Narrative Risk."
	}

	display := fighterDisplayTitle(f)

	// Compose page text (kept close to create_fighters.sh)
	var b strings.Builder
	fmt.Fprintf(&b, "{{DISPLAYTITLE:%s}}\n", display)
	fmt.Fprintf(&b, "{{Fighter\n|team=%s\n|class=%s\n|strength=%d\n|speed=%d\n|endurance=%d\n|technique=%d\n|lore=%s\n}}\n\n",
		team, className, f.Strength, f.Speed, f.Endurance, f.Technique, f.Lore)

	fmt.Fprintf(&b, "'''%s''' is a registered combatant in the '''Spoodblort''' violence league. As '''%s''', this fighter represents '''%s''' and competes as a '''%s''' archetype.\n\n",
		f.Name, display, team, className)

	b.WriteString("== Lore ==\n")
	b.WriteString(loreMarkdown + "\n\n")

	b.WriteString("== Combat Profile ==\n")
	b.WriteString("{| class=\"wikitable\"\n")
	b.WriteString("! Attribute !! Value\n|-\n")
	fmt.Fprintf(&b, "| Strength || %d\n|-%s", f.Strength, "\n")
	fmt.Fprintf(&b, "| Speed || %d\n|-%s", f.Speed, "\n")
	fmt.Fprintf(&b, "| Endurance || %d\n|-%s", f.Endurance, "\n")
	fmt.Fprintf(&b, "| Technique || %d\n", f.Technique)
	b.WriteString("|}\n\n")

	b.WriteString("== Chaos Metrics ==\n")
	b.WriteString("{| class=\"wikitable\"\n")
	b.WriteString("! Metric !! Reading\n|-\n")
	fmt.Fprintf(&b, "| Blood Type || %s\n|-%s", fallbackString(f.BloodType, "Unknown"), "\n")
	fmt.Fprintf(&b, "| Horoscope || %s\n|-%s", fallbackString(f.Horoscope, "Classified"), "\n")
	fmt.Fprintf(&b, "| Molecular Density || %v\n|-%s", f.MolecularDensity, "\n")
	fmt.Fprintf(&b, "| Existential Dread || %d\n|-%s", f.ExistentialDread, "\n")
	fmt.Fprintf(&b, "| Fingers || %d\n|-%s", f.Fingers, "\n")
	fmt.Fprintf(&b, "| Toes || %d\n|-%s", f.Toes, "\n")
	fmt.Fprintf(&b, "| Recorded Ancestors || %d\n", f.Ancestors)
	b.WriteString("|}\n\n")

	b.WriteString("== Fight Record ==\n")
	fmt.Fprintf(&b, "* Wins: %d\n", f.Wins)
	fmt.Fprintf(&b, "* Losses: %d\n", f.Losses)
	fmt.Fprintf(&b, "* Draws: %d\n\n", f.Draws)

	b.WriteString("== Data Provenance ==\n")
	b.WriteString("* Imported from Spoodblort game database on {{CURRENTDAY}} {{CURRENTMONTHNAME}} {{CURRENTYEAR}}.\n")
	b.WriteString("* Lore managed by the Department of Narrative Risk.\n\n")

	b.WriteString("[[Category:Fighters]]\n")
	return b.String()
}

func fallbackString(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

// UpsertFighterPage writes the full fighter page content to the wiki.
func (c *Client) UpsertFighterPage(f database.Fighter) error {
	title := FighterPageTitle(f)
	text := BuildFighterPageText(f)
	return c.SetText(title, text, "Sync fighter page from game DB")
}

// --- Fight pages ---

func fightDisplayTitle(f database.Fight) string {
	return fmt.Sprintf("Fight %d %s vs %s", f.ID, f.Fighter1Name, f.Fighter2Name)
}

// FightPageTitle mirrors legacy naming: Fight_123_Name1_vs_Name2
func FightPageTitle(f database.Fight) string {
	return fmt.Sprintf("Fight_%d_%s_vs_%s", f.ID, sanitizeName(f.Fighter1Name), sanitizeName(f.Fighter2Name))
}

// BuildFightPageText builds a rich fight page with avatars, stats, and result details.
func BuildFightPageText(f database.Fight, f1 database.Fighter, f2 database.Fighter, tournamentName string) string {
	display := fightDisplayTitle(f)

	// Winner and score formatting
	winner := ""
	if f.WinnerID.Valid {
		if int(f.WinnerID.Int64) == f.Fighter1ID {
			winner = f1.Name
		} else if int(f.WinnerID.Int64) == f.Fighter2ID {
			winner = f2.Name
		}
	}

	resultSection := "To be determined."
	if winner != "" {
		if f.FinalScore1.Valid && f.FinalScore2.Valid {
			resultSection = fmt.Sprintf("Winner: %s — Final Score %d-%d.", winner, f.FinalScore1.Int64, f.FinalScore2.Int64)
		} else {
			resultSection = fmt.Sprintf("Winner: %s.", winner)
		}
	}

	// External avatars (rendered via HTML to support external URLs)
	avatar1 := strings.TrimSpace(f1.AvatarURL)
	if avatar1 == "" {
		avatar1 = database.DefaultFighterAvatarPath
	}
	avatar2 := strings.TrimSpace(f2.AvatarURL)
	if avatar2 == "" {
		avatar2 = database.DefaultFighterAvatarPath
	}

	var b strings.Builder
	fmt.Fprintf(&b, "{{DISPLAYTITLE:%s}}\n", display)
	fmt.Fprintf(&b, "{{Fight\n|tournament=%s\n|scheduled=%s\n|fighter1=%s\n|fighter2=%s\n|status=%s\n|winner=%s\n}}\n\n",
		tournamentName, f.ScheduledTime.Format("2006-01-02 15:04:05-07:00"), f.Fighter1Name, f.Fighter2Name, f.Status, winner)

	fmt.Fprintf(&b, "'''%s'''\n\n", display)

	b.WriteString("== Overview ==\n")
	fmt.Fprintf(&b, "This bout is scheduled under the '''%s''' card.\n\n", tournamentName)

	b.WriteString("== Tale of the Tape ==\n")
	b.WriteString("{| class=\"wikitable\" style=\"width:100%;; text-align:center;\"\n")
	b.WriteString("! colspan=2 | ")
	b.WriteString(f1.Name)
	b.WriteString(" ||  || colspan=2 | ")
	b.WriteString(f2.Name)
	b.WriteString("\n| -\n")
	fmt.Fprintf(&b, "| <html><img src=\"%s\" style=\"max-width:140px; border-radius:6px;\"></html> || || <html><img src=\"%s\" style=\"max-width:140px; border-radius:6px;\"></html>\n", avatar1, avatar2)
	b.WriteString("|-\n")
	b.WriteString("! Attribute !! Value || || ! Attribute !! Value\n")
	b.WriteString("|-\n")
	fmt.Fprintf(&b, "| Strength || %d || || Strength || %d\n", f1.Strength, f2.Strength)
	b.WriteString("|-\n")
	fmt.Fprintf(&b, "| Speed || %d || || Speed || %d\n", f1.Speed, f2.Speed)
	b.WriteString("|-\n")
	fmt.Fprintf(&b, "| Endurance || %d || || Endurance || %d\n", f1.Endurance, f2.Endurance)
	b.WriteString("|-\n")
	fmt.Fprintf(&b, "| Technique || %d || || Technique || %d\n", f1.Technique, f2.Technique)
	b.WriteString("|}\n\n")

	b.WriteString("== Result ==\n")
	b.WriteString(resultSection + "\n\n")

	if f.FinalScore1.Valid && f.FinalScore2.Valid {
		b.WriteString("=== Final Health ===\n")
		b.WriteString("{| class=\"wikitable\"\n")
		b.WriteString("! Fighter !! Health\n|-\n")
		fmt.Fprintf(&b, "| %s || %d\n|-\n", f1.Name, f.FinalScore1.Int64)
		fmt.Fprintf(&b, "| %s || %d\n", f2.Name, f.FinalScore2.Int64)
		b.WriteString("|}\n\n")
	}

	b.WriteString("[[Category:Fights]]\n")
	return b.String()
}

// UpsertFightPage writes the full fight page content to the wiki.
func (c *Client) UpsertFightPage(f database.Fight, f1 database.Fighter, f2 database.Fighter, tournamentName string) error {
	title := FightPageTitle(f)
	text := BuildFightPageText(f, f1, f2, tournamentName)
	return c.SetText(title, text, "Sync fight page from game DB")
}
