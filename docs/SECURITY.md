# Security notes

## Threat model

`github.com/Trysac/u294-fake-clock` is a pure Go library: no network, filesystem, `os/exec`, or third-party dependencies. It does not handle untrusted data across a trust boundary; it is not a sandbox and cannot protect against malicious code in the same process.

### Defended

- **Supply chain**: standard library only; no runtime dependencies; releases are git tags served through the Go module proxy and checksum database.
- **Secrets**: the repository stores no credentials or registry tokens.
- **Programmer error**: invalid time movement and non-positive ticker intervals panic with documented messages rather than corrupting clock state.

### Not defended

- **Caller misuse**: `Advance` with a negative duration, `Set` to an earlier time, and `NewTicker` with a non-positive interval panic. Callers must not recover and continue with invalid assumptions.
- **Resource exhaustion**: each timer, ticker, or sleeper retains state until stopped or fired; the library does not cap the number of pending events.
- **Time model gaps**: time zones, monotonic clock readings, and `AfterFunc` are not modeled.
- **Blocking**: `BlockUntilContext` waits until `Waiters() >= n` or the context is done; a context without a deadline can block indefinitely.

## Publishing

Releases are git tags served through the Go module proxy and checksum database. No registry token is stored in the repository.

To report a vulnerability, use the repository's private security advisory channel.
