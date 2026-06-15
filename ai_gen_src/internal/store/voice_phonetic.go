package store

import (
	"context"
	"strings"
	"sync"
	"time"
)

// PhoneticEntry maps an STT-heard pattern to its canonical form.
type PhoneticEntry struct {
	HeardPattern string
	Canonical    string
	Category     string
}

// phoneticCache caches DB-backed phonetic entries with 10-minute TTL.
var phoneticCache = struct {
	mu       sync.RWMutex
	entries  []PhoneticEntry
	loadedAt time.Time
	ttl      time.Duration
}{ttl: 10 * time.Minute}

// domainPhoneticStatic is a compile-time static map for critical domain terms.
// These supplement DB entries and apply regardless of DB state.
var domainPhoneticStatic = [][2]string{
	// Providers
	{"i media", "IMEDIA"}, {"ai media", "IMEDIA"}, {"i me dia", "IMEDIA"},
	{"e sale", "ESALE"}, {"shop pay", "SHOPPAY"},
	// Products (short forms)
	{"the zing", "ZING"}, {"the garena", "GARENA"},
	{"the vinaphone", "VINAPHONE"}, {"the mobifone", "MOBIFONE"}, {"the viettel", "VIETTEL"},
	{"nap vina", "TOPUP_VINA"}, {"nap vinaphone", "TOPUP_VINA"},
	{"nap mobi", "TOPUP_MOBI"}, {"nap mobifone", "TOPUP_MOBI"},
	{"nap viettel", "TOPUP_VIETTEL"},
	// Amounts
	{"muoi nghin", "10000"}, {"muoi nghìn", "10000"}, {"10 nghin", "10000"},
	{"hai muoi nghin", "20000"}, {"nam muoi nghin", "50000"}, {"mot tram nghin", "100000"},
}

// ApplyDomainPhonetic applies the static phonetic map to a normalized message,
// replacing heard patterns with their canonical forms (lowercased).
func ApplyDomainPhonetic(norm string) string {
	result := norm
	for _, pair := range domainPhoneticStatic {
		heard, canonical := pair[0], strings.ToLower(pair[1])
		if strings.Contains(result, heard) {
			result = strings.ReplaceAll(result, heard, canonical)
		}
	}
	return result
}

// ApplyPhoneticEntries applies a slice of DB-backed entries to a normalized message.
func ApplyPhoneticEntries(norm string, entries []PhoneticEntry) string {
	result := norm
	for _, e := range entries {
		heard := strings.ToLower(e.HeardPattern)
		canonical := strings.ToLower(e.Canonical)
		if strings.Contains(result, heard) {
			result = strings.ReplaceAll(result, heard, canonical)
		}
	}
	return result
}

// GetPhoneticEntries returns cached phonetic entries, refreshing if stale.
func (db *DB) GetPhoneticEntries(ctx context.Context) []PhoneticEntry {
	phoneticCache.mu.RLock()
	if time.Since(phoneticCache.loadedAt) < phoneticCache.ttl {
		out := phoneticCache.entries
		phoneticCache.mu.RUnlock()
		return out
	}
	phoneticCache.mu.RUnlock()

	phoneticCache.mu.Lock()
	defer phoneticCache.mu.Unlock()
	if time.Since(phoneticCache.loadedAt) < phoneticCache.ttl {
		return phoneticCache.entries
	}
	if entries, err := db.listPhoneticEntries(ctx); err == nil {
		phoneticCache.entries = entries
		phoneticCache.loadedAt = time.Now()
	}
	return phoneticCache.entries
}

func (db *DB) listPhoneticEntries(ctx context.Context) ([]PhoneticEntry, error) {
	const q = `SELECT heard_pattern, canonical, category FROM voice_phonetic_map
               WHERE enabled = 1 ORDER BY priority DESC, id`
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PhoneticEntry
	for rows.Next() {
		var e PhoneticEntry
		if err := rows.Scan(&e.HeardPattern, &e.Canonical, &e.Category); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// UpsertPhoneticEntry inserts or updates a phonetic map entry.
func (db *DB) UpsertPhoneticEntry(ctx context.Context, heard, canonical, category string, priority int) error {
	const q = `INSERT INTO voice_phonetic_map (heard_pattern, canonical, category, priority)
               VALUES (?, ?, ?, ?)
               ON DUPLICATE KEY UPDATE canonical=VALUES(canonical), category=VALUES(category),
                 priority=VALUES(priority), enabled=1`
	_, err := db.ExecContext(ctx, q, heard, canonical, category, priority)
	if err == nil {
		// Invalidate cache
		phoneticCache.mu.Lock()
		phoneticCache.loadedAt = time.Time{}
		phoneticCache.mu.Unlock()
	}
	return err
}
