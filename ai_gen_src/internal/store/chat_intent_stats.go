package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// BumpChatIntentStat records or increments a FAQ pattern hit (indexed by intent + hash).
func (db *DB) BumpChatIntentStat(ctx context.Context, intentKey, sampleMessage string) error {
	intentKey = strings.TrimSpace(intentKey)
	if intentKey == "" || intentKey == "unknown" {
		return nil
	}
	norm := normalizeChatPattern(sampleMessage)
	if norm == "" {
		return nil
	}
	hash := chatPatternHash(intentKey, norm)
	sample := sampleMessage
	if len(sample) > 480 {
		sample = sample[:480]
	}
	const query = `
		INSERT INTO chat_intent_stats (intent_key, pattern_hash, sample_message, hit_count, last_seen_at)
		VALUES (?, ?, ?, 1, NOW())
		ON DUPLICATE KEY UPDATE
			hit_count = hit_count + 1,
			last_seen_at = NOW(),
			sample_message = IF(CHAR_LENGTH(?) > CHAR_LENGTH(sample_message), ?, sample_message)`
	_, err := db.ExecContext(ctx, query, intentKey, hash, sample, sample, sample)
	return err
}

func normalizeChatPattern(msg string) string {
	s := strings.ToLower(strings.TrimSpace(msg))
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' {
			if !prevSpace {
				b.WriteRune(' ')
				prevSpace = true
			}
			continue
		}
		prevSpace = false
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

func chatPatternHash(intentKey, norm string) string {
	h := sha256.Sum256([]byte(intentKey + "|" + norm))
	return hex.EncodeToString(h[:12])
}
