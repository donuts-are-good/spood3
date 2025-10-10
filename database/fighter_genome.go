package database

import (
	"crypto/sha512"
	"encoding/binary"
	"fmt"
	"time"
)

// DeriveGenome computes a sha3-512 digest over all stable Fighter fields except the genome itself.
// The output is returned as a lowercase hex string.
func (f Fighter) DeriveGenome() string {
	h := sha512.New()

	writeString := func(s string) { _, _ = h.Write([]byte(s)); _, _ = h.Write([]byte{0x1f}) }
	writeInt := func(n int) { var b [8]byte; binary.BigEndian.PutUint64(b[:], uint64(n)); _, _ = h.Write(b[:]) }
	writeBool := func(bv bool) {
		if bv {
			_, _ = h.Write([]byte{1})
		} else {
			_, _ = h.Write([]byte{0})
		}
	}
	writeTime := func(t time.Time) { writeString(t.UTC().Format(time.RFC3339Nano)) }
	writeFloat := func(x float64) { writeString(fmt.Sprintf("%.9g", x)) }

	// Exclude any future Genome field; include the rest in a stable order.
	writeString(f.Name)
	writeString(f.Team)
	writeInt(f.Strength)
	writeInt(f.Speed)
	writeInt(f.Endurance)
	writeInt(f.Technique)
	writeString(f.BloodType)
	writeString(f.Horoscope)
	writeFloat(f.MolecularDensity)
	writeInt(f.ExistentialDread)
	writeInt(f.Fingers)
	writeInt(f.Toes)
	writeInt(f.Ancestors)
	writeString(f.FighterClass)
	writeInt(f.Wins)
	writeInt(f.Losses)
	writeInt(f.Draws)
	writeBool(f.IsDead)
	writeBool(f.IsUndead)
	if f.ReanimatedBy != nil {
		writeInt(*f.ReanimatedBy)
	} else {
		writeInt(0)
	}
	writeTime(f.CreatedAt)

	// Custom fighter fields
	if f.CreatedByUserID != nil {
		writeInt(*f.CreatedByUserID)
	} else {
		writeInt(0)
	}
	writeBool(f.IsCustom)
	if f.CreationDate != nil {
		writeTime(*f.CreationDate)
	} else {
		writeString("")
	}
	if f.CustomDescription != nil {
		writeString(*f.CustomDescription)
	} else {
		writeString("")
	}
	writeString(f.Lore)
	writeString(f.AvatarURL)

	// First digest
	sum0 := h.Sum(nil)
	// Second digest over the first to extend to 256 hex chars total
	h2 := sha512.New()
	_, _ = h2.Write(sum0)
	sum1 := h2.Sum(nil)
	return fmt.Sprintf("%x%x", sum0, sum1)
}
