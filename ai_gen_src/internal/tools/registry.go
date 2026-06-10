package tools

import (
	"context"

	"opsone/internal/store"
)

// Registry holds tool implementations (§6).
type Registry struct {
	DB *store.DB
}

// NewRegistry creates a tool registry backed by MySQL store.
func NewRegistry(db *store.DB) *Registry {
	return &Registry{DB: db}
}

func (r *Registry) dataSource(ctx context.Context) (string, error) {
	s, err := r.DB.GetAgentSettings(ctx)
	if err != nil {
		return "", err
	}
	return s.DataSource, nil
}
