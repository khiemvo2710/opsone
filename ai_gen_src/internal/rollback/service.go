package rollback

import (
	"context"
	"fmt"

	"opsone/internal/store"
)

// ConflictError when routing_after no longer matches current config (§8.7).
type ConflictError struct {
	Message string
}

func (e *ConflictError) Error() string { return e.Message }

// Service performs routing rollback (§8.7).
type Service struct {
	DB *store.DB
}

// Request for rollback.
type Request struct {
	ChangeID   uint64
	ExecutedBy string
	Reason     string
	Force      bool
}

// Response after rollback.
type Response struct {
	RollbackChangeID uint64
	OriginalChangeID uint64
	ProductCode      string
	RoutingRestored  map[string]float64
}

// Rollback restores routing_config from routing_before (§10.9).
func (s *Service) Rollback(ctx context.Context, req Request) (Response, error) {
	rec, err := s.DB.GetAgentChangeByID(ctx, req.ChangeID)
	if err != nil {
		return Response{}, err
	}
	if rec.ChangeStatus != "applied" {
		return Response{}, fmt.Errorf("change %d không ở trạng thái applied", req.ChangeID)
	}

	latest, err := s.DB.IsLatestAppliedChange(ctx, req.ChangeID, rec.ProductCode, rec.Scope, rec.SKUCode)
	if err != nil {
		return Response{}, err
	}
	if !latest {
		return Response{}, fmt.Errorf("change %d không phải bản mới nhất của scope — rollback LIFO", req.ChangeID)
	}

	matches, err := s.DB.RoutingMatchesCurrent(ctx, rec.ProductCode, rec.Scope, rec.SKUCode, rec.RoutingAfter)
	if err != nil {
		return Response{}, err
	}
	if !matches && !req.Force {
		return Response{}, &ConflictError{Message: "routing_config hiện tại không khớp routing_after — cần xác nhận force"}
	}

	by := req.ExecutedBy
	if by == "" {
		by = "ops.admin"
	}

	if err := s.DB.ApplyRoutingSnapshot(ctx, rec.RoutingBefore, by, false); err != nil {
		return Response{}, err
	}

	rollbackID, err := s.DB.InsertRollbackChange(ctx, rec, rec.RoutingBefore, by, req.Reason)
	if err != nil {
		return Response{}, err
	}
	if err := s.DB.MarkChangeRolledBack(ctx, req.ChangeID, by); err != nil {
		return Response{}, err
	}

	return Response{
		RollbackChangeID: rollbackID,
		OriginalChangeID: req.ChangeID,
		ProductCode:      rec.ProductCode,
		RoutingRestored:  rec.RoutingBefore.Providers,
	}, nil
}
