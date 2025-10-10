package scheduler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"spoodblort/database"
	"time"
)

// fbm1 produces smooth pseudo-noise by summing sines at different frequencies
func fbm1(t float64, phases [4]float64) float64 {
	weights := [4]float64{1.0, 0.5, 0.25, 0.125}
	freqs := [4]float64{0.8, 1.7, 3.5, 7.1}
	sum := 0.0
	wsum := 0.0
	for i := 0; i < 4; i++ {
		sum += weights[i] * math.Sin(t*freqs[i]+phases[i])
		wsum += weights[i]
	}
	return sum / wsum
}

func phaseFrom(seed string, salt string) [4]float64 {
	h := sha256.Sum256([]byte(seed + "|" + salt))
	var p [4]float64
	for i := 0; i < 4; i++ {
		p[i] = float64(h[i]) / 255.0 * 2 * math.Pi
	}
	return p
}

func mapRange(v, min, max float64) int { // clamp already implied by sin
	return int(min + (v*0.5+0.5)*(max-min))
}

// categorical helpers
func pickFromNoise(t float64, seed, salt string, pool []string) string {
	// Smooth index from FBM, consistent for given t/seed/salt
	idxF := (fbm1(t, phaseFrom(seed, salt))*0.5 + 0.5) * float64(len(pool))
	idx := int(math.Floor(idxF))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(pool) {
		idx = len(pool) - 1
	}
	return pool[idx]
}

var (
	poolBiomes = []string{
		"Arena Suburbia", "Industrial Coast", "Tundra Arcade", "Mega‑Mall", "Salt Flats", "Crater Basin", "Glitter Dunes", "Marble Plaza", "Canal Quarter", "Rooftop Gardens", "Subway Catacombs", "Data Center Tundra", "Parking Megalith", "Stadium Archipelago", "Amphitheater Sinkhole", "Industrial Coast", "Tundra Arcade", "Mega‑Mall", "Salt Flats", "Crater Basin", "Glitter Dunes", "Marble Plaza", "Canal Quarter", "Rooftop Gardens", "Subway Catacombs", "Data Center Tundra", "Parking Megalith", "Stadium Archipelago", "Amphitheater Sinkhole",
	}
	poolPizzas    = []string{"very bad", "bad", "okay", "meh", "good", "oh yeah", "great", "very great", "amazing"}
	poolOfficials = []string{"bribed", "resisting bribe", "complicit", "difficult", "expensive", "merciful", "audit‑happy", "sleepy", "on strike", "training", "confused"}
	poolCheese    = []string{"light", "aromatic", "pungent", "overpowering", "rich", "salty", "inviting", "nutty", "barnyard", "blue‑echo", "smoky", "buttery"}
	poolRegimes   = []string{"Calm", "Viscous", "Storm", "Bureaucratic Fog", "Chaos Squall", "Static Hiss", "Lucky Drizzle"}
)

// EnsureWeeklyWeather computes weekly card if missing
func (s *Scheduler) EnsureWeeklyWeather(t *database.Tournament, weekStart time.Time) error {
	seed := weeklySeed(t, weekStart)
	tDays := float64(weekStart.Unix()) / 86400.0
	pick := func(pool []string, salt string) string { return pickFromNoise(tDays/7.0, seed, salt, pool) }
	ww := &database.WeatherWeekly{
		TournamentID:         t.ID,
		TournamentWeek:       t.WeekNumber,
		WeekStart:            weekStart,
		SeedHash:             seed,
		AlgoVersion:          "Ω-7.2",
		Biome:                pick(poolBiomes, "biome"),
		PizzaSelection:       pick(poolPizzas, "pizza"),
		CasinoOfficials:      pick(poolOfficials, "officials"),
		WeeklyTraitsJSON:     "{}",
		TransitionMatrixJSON: "{}",
	}
	return s.repo.UpsertWeeklyWeather(ww)
}

func weeklySeed(t *database.Tournament, start time.Time) string {
	s := fmt.Sprintf("%d|%d|%s|%s|%s", t.ID, t.WeekNumber, t.Name, t.Sponsor, start.UTC().Format("2006-01-02"))
	sum := sha256.Sum256([]byte(s))
	sum2 := sha256.Sum256(sum[:])
	return hex.EncodeToString(append(sum[:], sum2[:]...))
}

// EnsureDailyWeather computes daily record for the given date
func (s *Scheduler) EnsureDailyWeather(t *database.Tournament, day time.Time) error {
	// tDays: days since epoch for smoothness
	tDays := float64(day.Unix()) / 86400.0
	seed := fmt.Sprintf("%d|%d|%s", t.ID, t.WeekNumber, day.UTC().Format("2006-01-02"))
	phasesV := phaseFrom(seed, "visc")
	phasesT := phaseFrom(seed, "temp")
	phasesW := phaseFrom(seed, "wind")
	phasesR := phaseFrom(seed, "rain")

	visc := mapRange(fbm1(tDays/1.6, phasesV), 2000, 80000)
	temp := mapRange(fbm1(tDays/1.7, phasesT), 0, 100)
	windS := mapRange(fbm1(tDays/1.4, phasesW), 2, 35)
	windD := mapRange(fbm1(tDays/1.3, phasesW), 0, 359)
	rain := mapRange(fbm1(tDays/1.8, phasesR), 0, 120)
	if rain < 40 {
		rain = 0
	} else {
		rain -= 40
	}

	cheese := pickFromNoise(tDays/2.5, seed, "cheese", poolCheese)
	regime := pickFromNoise(tDays/3.0, seed, "regime", poolRegimes)
	// indices (example small set)
	indices := map[string]int{
		"barometric_mood":    mapRange(fbm1(tDays/1.9, phaseFrom(seed, "mood")), 0, 100),
		"audience_voltage":   mapRange(fbm1(tDays/1.5, phaseFrom(seed, "volt")), 0, 100),
		"fog_of_bureaucracy": mapRange(fbm1(tDays/2.3, phaseFrom(seed, "fog")), 0, 100),
	}
	indJSON, _ := json.Marshal(indices)
	counts := map[string]int{
		"hornet_sightings":  mapRange(fbm1(tDays/2.1, phaseFrom(seed, "hornet")), 0, 6),
		"lightning_strikes": mapRange(fbm1(tDays/1.8, phaseFrom(seed, "zap")), 0, 9),
		"toe_audits":        mapRange(fbm1(tDays/2.7, phaseFrom(seed, "toe")), 0, 5),
	}
	cntJSON, _ := json.Marshal(counts)

	dd := &database.WeatherDaily{
		Date:            day,
		TournamentID:    t.ID,
		TournamentWeek:  t.WeekNumber,
		SeedHash:        seed,
		AlgoVersion:     "Ω-7.2",
		Regime:          regime,
		Viscosity:       visc,
		TemperatureF:    temp,
		Temporality:     mapRange(fbm1(tDays/2.0, phaseFrom(seed, "tempo")), 0, 100),
		CheeseSmell:     cheese,
		TimeMode:        pickFromNoise(tDays/4.0, seed, "timemode", []string{"A", "B", "C", "D", "E", "F"}),
		WindSpeedMPH:    windS,
		WindDirDeg:      windD,
		PrecipitationMM: rain,
		DrizzleMinutes:  mapRange(fbm1(tDays/2.2, phaseFrom(seed, "drizzle")), 0, 120),
		IndicesJSON:     string(indJSON),
		CountsJSON:      string(cntJSON),
		EventsJSON:      "[]",
		MetaJSON:        "{}",
		IsFinal:         false,
	}
	return s.repo.UpsertDailyWeather(dd)
}
