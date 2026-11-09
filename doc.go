// Package fakeclock is a small, dependency-free Go library: A Clock interface for Go with a real implementation and a manual fake whose timers, tickers and sleeps fire only when a test advances time
//
// The public API is every exported identifier of this root package. Functions are pure: they do
// no I/O, keep no global mutable state and report invalid input through returned errors.
package fakeclock
