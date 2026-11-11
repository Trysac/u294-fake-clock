# Architecture

## Fake Clock Design

The fake clock maintains a sorted list of pending events (timers and tickers)
and a list of sleepers. The events are ordered by their fire time and, for
equal times, by creation sequence to ensure deterministic ordering. A mutex
protects the state, and a condition variable is used for efficient waiting
and broadcasting when waiter counts change.

`Advance` and `Set` step through the due times in order, setting the clock to
each deadline before firing events or waking sleepers. This ensures that
timers created during the same advance (e.g., rescheduled tickers) with
deadlines within the window are also fired.

`BlockUntilContext` waits without polling until the required number of
waiters (blocked sleeps or pending timer channels) is reached or the context
is cancelled.
