# Manual fake clock with deterministic ordering

Status: accepted (generated, pending human review)

## Context

Testing code that uses time.Now, time.After, or time.Sleep directly leads to slow and flaky tests. An injectable clock abstraction is needed, but the fake must fire timers in a predictable order and allow tests to synchronize with goroutines blocked on the clock.

## Decision

Implement a Fake clock that stores pending events (timers and tickers) in a slice sorted by fire time and creation sequence. Advance and Set step through due deadlines, updating the current time before firing. A condition variable is used to broadcast wait state changes, and BlockUntilContext waits efficiently for a minimum number of waiters.

## Consequences

Tests become deterministic and fast because no real time passes unless the test advances the clock. The mutex held during Advance serializes all operations, which may limit concurrency but is acceptable for a testing utility. The ordered firing guarantees that ties are broken by creation order.
