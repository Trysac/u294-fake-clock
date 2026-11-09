# fake-clock

[![Go Reference](https://pkg.go.dev/badge/github.com/Trysac/u294-fake-clock.svg)](https://pkg.go.dev/github.com/Trysac/u294-fake-clock)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/Trysac/u294-fake-clock/actions/workflows/ci.yml/badge.svg)](https://github.com/Trysac/u294-fake-clock/actions/workflows/ci.yml)

A Clock interface for Go with a real implementation and a manual fake whose timers, tickers and sleeps fire only when a test advances time

## Problem

Describe in two or three sentences the concrete problem this utility solves, who hits it, and what
they do today without it. Replace this paragraph when the first real feature lands.

## Install

```sh
go get github.com/Trysac/u294-fake-clock
```

Requires Go 1.23 or newer. No dependencies outside the standard library.

## Usage

```go
package main

import (
	"errors"
	"fmt"

	"github.com/Trysac/u294-fake-clock"
)

func main() {
	// Normalize trims surrounding whitespace and lower-cases the text.
	out, _ := fakeclock.Normalize("  Hello World ")
	fmt.Println(out) // -> "hello world"

	// Blank input is rejected explicitly instead of returning an empty string.
	if _, err := fakeclock.Normalize("   "); errors.Is(err, fakeclock.ErrBlank) {
		fmt.Println("rejected:", err)
	}
}
```

## Documentation

- [Architecture](docs/ARCHITECTURE.md) and [decisions](docs/decisions/)
- [Testing](docs/TESTING.md) and [security](docs/SECURITY.md)
- [Project record](docs/PROJECT.md) and [stack](docs/STACK.md)
- [Changelog](CHANGELOG.md) and [contributing guide](CONTRIBUTING.md)

## License

[MIT](LICENSE) (c) 2026 Miguel Gomez
