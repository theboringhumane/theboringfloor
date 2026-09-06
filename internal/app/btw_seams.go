package app

// btwSwapSafetyBackend reports whether moving the primary while its boss turn
// is still in flight preserves that turn. Backends that do not expose this
// additive seam are conservatively treated as unsafe.
type btwSwapSafetyBackend interface {
	SwapSafeMidTurn() bool
}

// btwReconcileBackend restores a completion that arrived while its session was
// unseated. It is deliberately additive so demo and older backends continue to
// work without a reconciliation implementation.
type btwReconcileBackend interface {
	ReconcileBoss(sessionID string) error
}
