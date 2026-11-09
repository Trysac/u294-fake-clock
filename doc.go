// Package fakeclock provides a Clock interface for Go with two
// implementations: a real clock that wraps the standard library time
// package, and a manual fake clock for deterministic testing.
//
// The fake clock advances time only when the test calls Advance or Set.
// Timers, tickers, and sleeps fire in a defined order (by deadline, then
// creation sequence), and channels are buffered so that sends never block.
package fakeclock
