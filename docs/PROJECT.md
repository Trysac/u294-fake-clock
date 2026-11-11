# Project record

## Objective

`fake-clock` provides a `Clock` interface for Go with `Real()`, a thin wrapper over the `time` package, and `New(start)`, a manual fake clock whose timers, tickers and sleeps fire only when a test advances time via `Advance` or `Set`.

## Scope

In scope:

- The public API `Clock`, `Timer`, `Ticker`, `Real`, `New`, and `*Fake` methods `Advance`, `Set`, `Waiters`, `BlockUntilContext`.
- Deterministic ordering: due events fire by deadline, with ties broken by creation sequence; `Now` is set to each deadline before firing.
- Buffer-1 channels; ticker drop behavior; zero and negative duration behavior; documented panic messages.
- Tests for timers, tickers, stop/reset, sleepers, concurrent advance, and the race detector.

Out of scope:

- Time zones, monotonic clock readings, and `AfterFunc`.
- Real-time sleeps or scheduler-dependent behavior in the fake.
- Dependencies outside the Go standard library.

## Timeline

| Date | Event |
|---|---|
| 2026 | Repository created from the factory utility template. |

## Effort metrics

Measured values only; a value that has not been measured stays "pending measurement".

| Metric | Value |
|---|---|
| Tokens (input / output) | 599456 / 64907 |
| Hours (human) | pending measurement |
| Cost (USD) | 0.0792 |

## Factory run u294 (measured from API usage)

| Phase | Model | Batch | Input | Output | Cache write | Cache read | Cost (USD) |
|---|---|---|---|---|---|---|---|
| generation | deepseek-v4-pro | yes | 3337 | 20079 | 0 | 1280 | 0.0420 |
| fix-1 | deepseek-flash | no | 16900 | 7807 | 0 | 155904 | 0.0077 |
| audit | deepseek-flash | no | 25551 | 21225 | 0 | 253312 | 0.0173 |
| fix-2 | deepseek-flash | no | 12102 | 6861 | 0 | 127488 | 0.0063 |
| docs | deepseek-flash | no | 3326 | 8935 | 0 | 256 | 0.0059 |

Total: $0.0792. Prices: deepseek line, as measured by its adapter.
