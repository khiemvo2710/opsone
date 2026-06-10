package agent

// Recovery timeline after routing apply (§3.1, §8.6.3): +1 cycle → yellow, +1 more if OK → green.
const (
	recoveryYellowAfterCycles = 1
	recoveryGreenAfterCycles  = 2
)

// overlayRecoveryState adjusts scope state during post-routing recovery window.
func overlayRecoveryState(baseState string, breached bool, cyclesSinceApply int) (state string, clearRecovery bool) {
	if cyclesSinceApply <= 0 {
		return baseState, false
	}
	if cyclesSinceApply < recoveryGreenAfterCycles {
		return "RECOVERING", false
	}
	if !breached {
		return "NORMAL", true
	}
	return "INCIDENT", false
}
