# fake-clock

`fake-clock` provides `fakeclock.Clock`, an interface for the time operations Go tests commonly need, with two implementations: `Real()`, a thin wrapper over the `time` package, and `New(start)`, a manual fake clock. The fake fires timers, tickers and sleeps only when the test calls `Advance` or `Set`; it does not sleep in wall-clock time.

## Installation

```sh
go get github.com/Trysac/u294-fake-clock
```

Import the module as `fakeclock`:

```go
import fakeclock "github.com/Trysac/u294-fake-clock"
```

## Usage

```go
package main

import (
    "time"
    fakeclock "github.com/Trysac/u294-fake-clock"
)
func main() {
    f := fakeclock.New(time.Unix(0, 0)); ch := f.After(time.Second)
    f.Advance(time.Second); <-ch
}
```

## Public API

- `type Clock interface { Now() time.Time; Since(t time.Time) time.Duration; After(d time.Duration) <-chan time.Time; Sleep(d time.Duration); NewTimer(d time.Duration) Timer; NewTicker(d time.Duration) Ticker }`
- `type Timer interface { C() <-chan time.Time; Stop() bool; Reset(d time.Duration) bool }`
- `type Ticker interface { C() <-chan time.Time; Stop(); Reset(d time.Duration) }`
- `func Real() Clock`
- `type Fake struct` with `func New(start time.Time) *Fake` and methods `Now`, `Since`, `After`, `Sleep`, `NewTimer`, `NewTicker`, `Advance(d time.Duration)`, `Set(t time.Time)`, `Waiters() int`, `BlockUntilContext(ctx context.Context, n int) error`.

### Errors and panics

`BlockUntilContext` is the only method that returns an error: it returns `ctx.Err()` if the context is already cancelled or becomes done before enough waiters are reached. The package has no sentinel errors and no configuration options.

Programmer errors panic with these exact messages:

- `Advance(d)` with a negative duration: `fakeclock: negative advance`
- `Set(t)` to an earlier time: `fakeclock: time moved backwards`
- `NewTicker(d)` with a non-positive interval: `fakeclock: non-positive ticker interval`

### Limits and behavior

Every timer, ticker and `After` channel has a buffer of 1 and a send never blocks. A full ticker channel drops the tick, like `time.Ticker`.

`Advance` moves time to each due deadline in order; ties are broken by creation sequence. `Now` is set to each deadline before firing, so timers created during `Advance` with a deadline inside the window fire in the same call. `Set` to a later time fires due timers the same way.

`After(d <= 0)` and `NewTimer(d <= 0)` fire on the next `Advance(0)`. `Sleep(d <= 0)` returns immediately. `Waiters` counts goroutines blocked in `Sleep` plus pending `After`/`NewTimer` channels not yet fired. `BlockUntilContext` waits on a condition variable until `Waiters() >= n` or the context is done; it does not poll.

Out of scope: time zones, monotonic clock readings, and `AfterFunc`. The module uses the Go standard library only.

## Why not benbjohnson/clock

The specification names `benbjohnson/clock`, which is archived. This package is maintained, exposes a smaller `Clock`/`Timer`/`Ticker` interface, and documents the firing order: deadline first, then creation sequence.

## Security

See [docs/SECURITY.md](docs/SECURITY.md).
