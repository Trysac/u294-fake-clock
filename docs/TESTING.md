# Testing

## Running

```sh
go test ./...
go test -race ./...
go test -cover ./...
```

## What exists

The Go tests are in `*_test.go` files in the module root and use package `fakeclock_test`. They exercise the exported API only: `New`, `Real`, `Advance`, `Set`, `Waiters`, `BlockUntilContext`, `After`, `Sleep`, `NewTimer`, `NewTicker`, and the `Timer`/`Ticker` methods.

Covered behavior includes:

- `After` before and after the exact deadline; ordering of 2s, 1s, and another 1s timer by deadline then creation sequence.
- `Stop` on pending and fired timers; `Reset` on a stopped timer; ticker buffer-1 drop and a later tick.
- `NewTicker(0)` panic; `Sleep` plus `BlockUntilContext` and `Waiters`; cancelled context returns `ctx.Err()`.
- `Advance(-1)` and `Set` backwards panic; `Set` forward fires due timers.
- `Real().Now()` within 1 s of `time.Now()`.
- `Reset` while pending, `Stop` twice, a ticker stopped during `Advance`, and 1000 timers at the same instant fired in creation order.
- A race-detector run where two goroutines call `Advance` concurrently while a third creates timers.

Runnable `Example` functions are in `fakeclock_test` and are verified by `go test` through their `// Output:` comments.

## Determinism

Tests do not sleep in real time except the `Real()` check. Goroutine tests use `BlockUntilContext` with a `context.WithTimeout` of 5 s only as a safety net. There is no network, no wall-clock dependency, and no dependence on test order.

There is no conformance tier: all tests use the standard library and run locally; there are no build-tagged external-tool tests.
