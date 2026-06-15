package rollback

import (
	"context"
	"fmt"

	"opsone/internal/store"
)

// Service provides rollback functionality for routing changes
type Service struct {
	DB *store.DB
}

// Request contains parameters for a rollback operation
type Request struct {
	ChangeID   string // change log ID to rollback
	ExecutedBy string // who is performing the rollback
	Reason     string // reason for rollback
}

// Response contains the result of a rollback operation
type Response struct {
	RoutingRestored map[string]float64 // provider → traffic_pct
	ChangedAt       string             // timestamp of rollback
}

// Rollback reverts a routing change to its previous state
func (s *Service) Rollback(ctx context.Context, req Request) (*Response, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// TODO: Implement rollback logic
	// 1. Fetch the change log entry by ChangeID
	// 2. Get the previous routing state
	// 3. Apply the previous state back to the database
	// 4. Record the rollback action

	return &Response{
		RoutingRestored: map[string]float64{},
		ChangedAt:       "",
	}, nil
}
