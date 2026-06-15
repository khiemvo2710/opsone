package store

import (
	"context"
	"database/sql"
)

// ChatSessionContext holds the current product/provider focus of a chat session.
type ChatSessionContext struct {
	Product  string
	Provider string
	SKU      string
}

// GetChatSessionContext returns the saved context for a session UUID.
func (db *DB) GetChatSessionContext(ctx context.Context, sessionUUID string) (ChatSessionContext, error) {
	if sessionUUID == "" {
		return ChatSessionContext{}, nil
	}
	const q = `SELECT COALESCE(ctx_product,''), COALESCE(ctx_provider,''), COALESCE(ctx_sku,'')
               FROM chat_sessions WHERE session_uuid = ?`
	var c ChatSessionContext
	err := db.QueryRowContext(ctx, q, sessionUUID).Scan(&c.Product, &c.Provider, &c.SKU)
	if err == sql.ErrNoRows {
		return ChatSessionContext{}, nil
	}
	return c, err
}

// UpdateChatSessionContext saves the current context for a session UUID.
func (db *DB) UpdateChatSessionContext(ctx context.Context, sessionUUID, product, provider, sku string) error {
	if sessionUUID == "" {
		return nil
	}
	const q = `UPDATE chat_sessions
               SET ctx_product=?, ctx_provider=?, ctx_sku=?, ctx_updated_at=NOW()
               WHERE session_uuid=?`
	_, err := db.ExecContext(ctx, q,
		sqlNullStr(product), sqlNullStr(provider), sqlNullStr(sku), sessionUUID)
	return err
}

func sqlNullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
