package utils

import (
	"math/rand"
	"time"
)

func NewSeededRNG(seed int64) *rand.Rand {
	return rand.New(rand.NewSource(seed))
}

func DailyFighterSeed(date time.Time) int64 {
	return int64(date.Year()*10000 + int(date.YearDay()))
}

func VoidReasonSeed(now time.Time) int64 {
	return now.Unix()
}

func FightTickSeed(fightID int, tickNumber int) int64 {
	return int64(fightID*1000000 + tickNumber)
}
