package chatresolve

import (
	"strings"

	"opsone/internal/catalog"
)

// UIActionSlots holds parsed scope for a UI action.
type UIActionSlots struct {
	Product     string
	SKU         string
	Provider    string
	DurationMin int
	PlanID      uint64
	AutoAction  string
	Reject      bool
}

var extendMaintTokens = []string{
	"gia han bao tri", "gia han bt", "keo dai bao tri", "extend maintenance", "extend maint",
}

var restoreBaselineTokens = []string{
	"tra lai routing", "tra lai baseline", "khoi phuc routing", "routing goc", "routing baseline",
	"restore baseline", "baseline routing",
}

var uiHelpTokens = []string{
	"co the lam gi", "lam duoc gi", "thao tac ui", "nut nao", "bam nut", "thuc hien tren ui",
	"thao tac dashboard", "help ui",
}

// IsExtendMaintenanceCommand extends active maintenance window.
func IsExtendMaintenanceCommand(msg string) bool {
	key := trimMsgKey(msg)
	if containsAny(key, extendMaintTokens) {
		return true
	}
	return strings.Contains(key, "gia han") &&
		(strings.Contains(key, "bao tri") || strings.Contains(key, "bt"))
}

// IsRestoreBaselineCommand restores routing baseline without cancelling maintenance.
func IsRestoreBaselineCommand(msg string) bool {
	if IsReopenServiceCommand(msg) {
		return false
	}
	key := trimMsgKey(msg)
	if containsAny(key, restoreBaselineTokens) {
		return true
	}
	return (strings.Contains(key, "tra lai") || strings.Contains(key, "khoi phuc")) &&
		strings.Contains(key, "routing")
}

// IsUIActionsHelpCommand asks what dashboard actions are available.
func IsUIActionsHelpCommand(msg string) bool {
	key := trimMsgKey(msg)
	return containsAny(key, uiHelpTokens)
}

func extractUIActionSlots(msg string) UIActionSlots {
	return UIActionSlots{
		Product:     ExtractProductFromText(msg),
		SKU:         NormalizeSKU(ExtractSKUFromText(msg)),
		Provider:    ExtractProviderFromText(msg),
		DurationMin: ExtractDurationMinutes(msg),
		AutoAction:  ParseScopeAutoMode(msg),
	}
}

// DetectUIAction maps user text to a Dashboard action (Tầng B §7.6.5.4).
func DetectUIAction(msg string) (catalog.UIActionKey, UIActionSlots) {
	slots := extractUIActionSlots(msg)

	if IsUIActionsHelpCommand(msg) {
		return catalog.UIActionHelp, slots
	}
	if IsListPendingCommand(msg) {
		return catalog.UIActionListPending, slots
	}
	if IsExtendMaintenanceCommand(msg) {
		return catalog.UIActionExtendMaint, slots
	}
	if IsReopenServiceCommand(msg) {
		return catalog.UIActionReopenService, slots
	}
	if IsRestoreBaselineCommand(msg) {
		return catalog.UIActionRestoreBaseline, slots
	}
	if IsSetMaintenanceCommand(msg) {
		return catalog.UIActionSetMaintenance, slots
	}
	if IsSetScopeAutoCommand(msg) {
		return catalog.UIActionSetScopeAuto, slots
	}
	if IsApproveShortCommand(msg) || containsScopedApproval(msg) {
		slots.Reject = false
		return catalog.UIActionApproveReject, slots
	}
	if IsRejectShortCommand(msg) {
		slots.Reject = true
		return catalog.UIActionApproveReject, slots
	}
	return "", slots
}

func containsScopedApproval(msg string) bool {
	key := CommandKey(msg)
	if !strings.Contains(key, "duyet") && !strings.Contains(key, "dong y") &&
		!strings.Contains(key, "tu choi") && !strings.Contains(key, "approve") &&
		!strings.Contains(key, "reject") {
		return false
	}
	return ExtractProductFromText(msg) != ""
}
