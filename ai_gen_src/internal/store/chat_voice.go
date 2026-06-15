package store

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// VoiceCorrection is one row from chat_voice_corrections.
type VoiceCorrection struct {
	ID            int64
	HeardNorm     string
	CorrectedNorm string
	HitCount      int
	LastSeenAt    time.Time
}

// UpsertVoiceCorrection records a STT heard→corrected pair (§7.6.5.5 P5).
// heard = stt_raw normalized, corrected = actual message normalized.
func (db *DB) UpsertVoiceCorrection(ctx context.Context, heard, corrected string) error {
	heard = normalizeChatPattern(heard)
	corrected = normalizeChatPattern(corrected)
	if heard == "" || corrected == "" || heard == corrected {
		return nil
	}
	// Truncate to column limit
	if len(heard) > 512 {
		heard = heard[:512]
	}
	if len(corrected) > 512 {
		corrected = corrected[:512]
	}
	const query = `
		INSERT INTO chat_voice_corrections (heard_norm, corrected_norm, hit_count, last_seen_at)
		VALUES (?, ?, 1, NOW())
		ON DUPLICATE KEY UPDATE
			hit_count = hit_count + 1,
			last_seen_at = NOW()`
	_, err := db.ExecContext(ctx, query, heard, corrected)
	return err
}

// ListVoiceCorrections returns top corrections ordered by hit_count.
func (db *DB) ListVoiceCorrections(ctx context.Context, limit int) ([]VoiceCorrection, error) {
	if limit <= 0 {
		limit = 50
	}
	const query = `SELECT id, heard_norm, corrected_norm, hit_count, last_seen_at
	               FROM chat_voice_corrections
	               ORDER BY hit_count DESC, last_seen_at DESC
	               LIMIT ?`
	rows, err := db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []VoiceCorrection
	for rows.Next() {
		var c VoiceCorrection
		if err := rows.Scan(&c.ID, &c.HeardNorm, &c.CorrectedNorm, &c.HitCount, &c.LastSeenAt); err != nil {
			continue
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetVoiceCorrections returns all corrections for runtime application (cached by caller).
func (db *DB) GetVoiceCorrections(ctx context.Context) (map[string]string, error) {
	const query = `SELECT heard_norm, corrected_norm FROM chat_voice_corrections ORDER BY hit_count DESC`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]string)
	for rows.Next() {
		var heard, corrected string
		if err := rows.Scan(&heard, &corrected); err != nil {
			continue
		}
		out[heard] = corrected
	}
	return out, rows.Err()
}

// ApplyVoiceCorrections applies known STT corrections to a normalized message.
// Returns the corrected message (or original if no correction found).
func ApplyVoiceCorrections(norm string, corrections map[string]string) string {
	if len(corrections) == 0 {
		return norm
	}
	// Exact match first
	if corrected, ok := corrections[norm]; ok {
		return corrected
	}
	// Substring replacement: replace heard fragments
	result := norm
	for heard, corrected := range corrections {
		if strings.Contains(result, heard) {
			result = strings.ReplaceAll(result, heard, corrected)
		}
	}
	return result
}

// MarkInteractionWrongRoute updates action_result='wrong_route' for a log entry.
func (db *DB) MarkInteractionWrongRoute(ctx context.Context, interactionID int64) error {
	const query = `UPDATE chat_interaction_log
	               SET action_result = 'wrong_route'
	               WHERE id = ? AND action_result != 'wrong_route'`
	_, err := db.ExecContext(ctx, query, interactionID)
	return err
}

// FindRecentInteraction looks for the latest interaction from the same session in the last N seconds.
// Returns (id, route, found).
func (db *DB) FindRecentInteraction(ctx context.Context, sessionUUID string, withinSeconds int) (int64, string, bool) {
	if withinSeconds <= 0 {
		withinSeconds = 60
	}
	const query = `SELECT id, route FROM chat_interaction_log
	               WHERE session_uuid = ?
	                 AND created_at >= DATE_SUB(NOW(), INTERVAL ? SECOND)
	               ORDER BY id DESC LIMIT 1`
	var id int64
	var route string
	err := db.QueryRowContext(ctx, query, sessionUUID, withinSeconds).Scan(&id, &route)
	if err == sql.ErrNoRows || err != nil {
		return 0, "", false
	}
	return id, route, true
}

// DetectRetrySignal checks if the current message is a retry of a recent interaction.
// Returns (prevInteractionID, isRetry). A retry is when:
//   - same session had an interaction within 60s
//   - current message is semantically similar (keyword overlap >= 60%)
func DetectRetrySignal(currentNorm string, prevNorm string) bool {
	if currentNorm == "" || prevNorm == "" {
		return false
	}
	// Same message = retry
	if currentNorm == prevNorm {
		return true
	}
	// Keyword overlap >= 60%
	curWords := keywordSet(currentNorm)
	prevWords := keywordSet(prevNorm)
	if len(curWords) == 0 || len(prevWords) == 0 {
		return false
	}
	overlap := 0
	for w := range curWords {
		if prevWords[w] {
			overlap++
		}
	}
	// Overlap ratio relative to smaller set
	minLen := len(curWords)
	if len(prevWords) < minLen {
		minLen = len(prevWords)
	}
	return float64(overlap)/float64(minLen) >= 0.6
}

func keywordSet(norm string) map[string]bool {
	words := strings.Fields(norm)
	set := make(map[string]bool, len(words))
	for _, w := range words {
		if len([]rune(w)) >= 2 {
			set[w] = true
		}
	}
	return set
}
