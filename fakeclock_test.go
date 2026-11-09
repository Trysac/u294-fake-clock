package fakeclock_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	fakeclock "github.com/Trysac/u294-fake-clock"
)

func TestAfterAndAdvance(t *testing.T) {
	t0 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	f := fakeclock.New(t0)

	ch := f.After(time.Second)

	// Advance 999ms: should not fire.
	f.Advance(999 * time.Millisecond)
	select {
	case <-ch:
		t.Fatal("timer fired too early")
	default:
	}

	// Advance remaining 1ms: should fire with t0+1s.
	f.Advance(time.Millisecond)
	want := t0.Add(time.Second)
	select {
	case got := <-ch:
		if !got.Equal(want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	default:
		t.Fatal("timer did not fire")
	}

	if got := f.Now(); !got.Equal(want) {
		t.Fatalf("Now() = %v, want %v", got, want)
	}
}

func TestTimerOrdering(t *testing.T) {
	t0 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	f := fakeclock.New(t0)

	// Create timers: 2s, 1s, 1s.
	ch2 := f.After(2 * time.Second)
	ch1a := f.After(time.Second)
	ch1b := f.After(time.Second)

	// Advance 3s.
	f.Advance(3 * time.Second)

	// Expect order: 1a, 1b, 2.
	expect := []<-chan time.Time{ch1a, ch1b, ch2}
	deadlines := []time.Time{
		t0.Add(time.Second),
		t0.Add(time.Second),
		t0.Add(2 * time.Second),
	}
	for i, ch := range expect {
		select {
		case got := <-ch:
			if !got.Equal(deadlines[i]) {
				t.Fatalf("timer %d: got %v, want %v", i, got, deadlines[i])
			}
		default:
			t.Fatalf("timer %d did not fire", i)
		}
	}

	if got := f.Now(); !got.Equal(t0.Add(3 * time.Second)) {
		t.Fatalf("Now() = %v, want %v", got, t0.Add(3*time.Second))
	}
}

func TestTimerStopAndReset(t *testing.T) {
	t0 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	f := fakeclock.New(t0)

	timer := f.NewTimer(time.Second)
	if !timer.Stop() {
		t.Fatal("Stop on pending timer should return true")
	}

	// Advance 2s: timer should not fire.
	f.Advance(2 * time.Second)
	select {
	case <-timer.C():
		t.Fatal("stopped timer fired")
	default:
	}

	// Stop again: should return false.
	if timer.Stop() {
		t.Fatal("Stop on already stopped timer should return false")
	}

	// Reset should rearm relative to current Now (t0+2s).
	if !timer.Reset(time.Second) {
		t.Fatal("Reset should return true")
	}
	f.Advance(time.Second)
	select {
	case got := <-timer.C():
		want := t0.Add(3 * time.Second)
		if !got.Equal(want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	default:
		t.Fatal("reset timer did not fire")
	}

	// Stop after fired: false.
	if timer.Stop() {
		t.Fatal("Stop after fired should return false")
	}
}

func TestTickerDropAndStopReset(t *testing.T) {
	t0 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	f := fakeclock.New(t0)

	tk := f.NewTicker(time.Second)
	defer tk.Stop()

	// Advance 3.5s: should have only one tick at t0+1s (buffer full).
	f.Advance(3500 * time.Millisecond)
	select {
	case got := <-tk.C():
		want := t0.Add(time.Second)
		if !got.Equal(want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	default:
		t.Fatal("no tick received")
	}
	// Channel should be empty (other ticks dropped).
	select {
	case <-tk.C():
		t.Fatal("unexpected extra tick")
	default:
	}

	// Advance 500ms: should deliver t0+4s.
	f.Advance(500 * time.Millisecond)
	select {
	case got := <-tk.C():
		want := t0.Add(4 * time.Second)
		if !got.Equal(want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	default:
		t.Fatal("expected tick at 4s")
	}

	// Stop ticker: no more ticks.
	tk.Stop()
	f.Advance(2 * time.Second)
	select {
	case <-tk.C():
		t.Fatal("ticker fired after Stop")
	default:
	}

	// Reset ticker after Stop? Not specified, but we can test it rearms.
	tk.Reset(time.Second)
	f.Advance(time.Second)
	select {
	case <-tk.C():
		// good
	default:
		t.Fatal("reset ticker did not fire")
	}
}

func TestNewTickerPanicsOnNonPositive(t *testing.T) {
	f := fakeclock.New(time.Now())
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		} else if r != "fakeclock: non-positive ticker interval" {
			t.Fatalf("got panic %q, want %q", r, "fakeclock: non-positive ticker interval")
		}
	}()
	f.NewTicker(0)
}

func TestSleepAndBlockUntilContext(t *testing.T) {
	t0 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	f := fakeclock.New(t0)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		f.Sleep(time.Minute)
		close(done)
	}()

	// Wait until the goroutine is blocked in Sleep.
	if err := f.BlockUntilContext(ctx, 1); err != nil {
		t.Fatalf("BlockUntilContext error: %v", err)
	}
	if got := f.Waiters(); got != 1 {
		t.Fatalf("Waiters() = %d, want 1", got)
	}

	// Advance the clock to wake the sleeper.
	f.Advance(time.Minute)
	select {
	case <-done:
		// success
	case <-time.After(5 * time.Second):
		t.Fatal("sleeper did not wake")
	}
	if got := f.Waiters(); got != 0 {
		t.Fatalf("Waiters() = %d, want 0", got)
	}

	// BlockUntilContext with already cancelled context.
	cancelledCtx, cancel2 := context.WithCancel(context.Background())
	cancel2()
	err := f.BlockUntilContext(cancelledCtx, 1)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestAdvancePanicsOnNegative(t *testing.T) {
	f := fakeclock.New(time.Now())
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		} else if r != "fakeclock: negative advance" {
			t.Fatalf("got panic %q, want %q", r, "fakeclock: negative advance")
		}
	}()
	f.Advance(-1)
}

func TestSetPanicsOnBackwards(t *testing.T) {
	t0 := time.Now()
	f := fakeclock.New(t0)
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		} else if r != "fakeclock: time moved backwards" {
			t.Fatalf("got panic %q, want %q", r, "fakeclock: time moved backwards")
		}
	}()
	f.Set(t0.Add(-time.Second))
}

func TestSetFiresTimers(t *testing.T) {
	t0 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	f := fakeclock.New(t0)
	timer := f.NewTimer(time.Second)

	// Set to t0+1s: timer should fire.
	f.Set(t0.Add(time.Second))
	select {
	case got := <-timer.C():
		if !got.Equal(t0.Add(time.Second)) {
			t.Fatalf("got %v, want %v", got, t0.Add(time.Second))
		}
	default:
		t.Fatal("timer did not fire on Set")
	}
	if got := f.Now(); !got.Equal(t0.Add(time.Second)) {
		t.Fatalf("Now() = %v, want %v", got, t0.Add(time.Second))
	}
}

func TestRealClockNow(t *testing.T) {
	r := fakeclock.Real()
	before := time.Now()
	got := r.Now()
	after := time.Now()
	if got.Before(before.Add(-time.Second)) || got.After(after.Add(time.Second)) {
		t.Fatalf("Real().Now() = %v, expected near %v", got, time.Now())
	}
}

func TestManyTimersSameInstant(t *testing.T) {
	t0 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	f := fakeclock.New(t0)
	const n = 1000
	chs := make([]<-chan time.Time, n)
	for i := 0; i < n; i++ {
		chs[i] = f.After(time.Second)
	}

	f.Advance(time.Second)

	// All should fire, and in creation order (same deadline).
	for i, ch := range chs {
		select {
		case got := <-ch:
			if !got.Equal(t0.Add(time.Second)) {
				t.Fatalf("timer %d: got %v, want %v", i, got, t0.Add(time.Second))
			}
		default:
			t.Fatalf("timer %d did not fire", i)
		}
	}
}

func TestResetPendingTimer(t *testing.T) {
	t0 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	f := fakeclock.New(t0)
	timer := f.NewTimer(2 * time.Second)

	// Reset to 1s before firing.
	if !timer.Reset(time.Second) {
		t.Fatal("Reset on pending timer should return true")
	}

	// Advance 1s: timer should fire.
	f.Advance(time.Second)
	select {
	case got := <-timer.C():
		if !got.Equal(t0.Add(time.Second)) {
			t.Fatalf("got %v, want %v", got, t0.Add(time.Second))
		}
	default:
		t.Fatal("timer did not fire after reset")
	}
}

func TestStopTwice(t *testing.T) {
	f := fakeclock.New(time.Now())
	timer := f.NewTimer(time.Second)
	if !timer.Stop() {
		t.Fatal("first Stop should return true")
	}
	if timer.Stop() {
		t.Fatal("second Stop should return false")
	}
}

func TestTickerStoppedBeforeAdvance(t *testing.T) {
	t0 := time.Now()
	f := fakeclock.New(t0)
	tk := f.NewTicker(time.Second)
	tk.Stop()

	f.Advance(2 * time.Second)
	select {
	case <-tk.C():
		t.Fatal("stopped ticker fired")
	default:
	}
}

func TestConcurrentAdvanceAndTimers(t *testing.T) {
	t0 := time.Now()
	f := fakeclock.New(t0)

	var wg sync.WaitGroup
	// Two goroutines calling Advance concurrently.
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f.Advance(time.Second)
		}()
	}
	// One goroutine creating timers.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			f.After(time.Second)
		}
	}()

	wg.Wait()
	// No panic or race; just ensure it's still functional.
	_ = f.Now()
}

// A fired timer whose channel was never drained must not deadlock the
// Fake when it is reset and advanced again.
func TestTimerFiredThenResetDoesNotHang(t *testing.T) {
	t0 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	f := fakeclock.New(t0)

	timer := f.NewTimer(time.Second)
	f.Advance(time.Second) // fires, value stays buffered
	if timer.Stop() {
		t.Fatal("Stop on a fired timer should return false")
	}
	if !timer.Reset(time.Second) {
		t.Fatal("Reset should return true")
	}

	f.Advance(time.Second) // must not block
	select {
	case got := <-timer.C():
		want := t0.Add(2 * time.Second)
		if !got.Equal(want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	default:
		t.Fatal("reset timer did not fire")
	}
	if got := f.Now(); !got.Equal(t0.Add(2 * time.Second)) {
		t.Fatalf("Now() = %v, want %v", got, t0.Add(2*time.Second))
	}
}

// Waiters counts goroutines blocked in Sleep plus pending timers, never
// live tickers, so BlockUntilContext cannot be satisfied by a ticker alone.
func TestWaitersExcludesTickers(t *testing.T) {
	t0 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	f := fakeclock.New(t0)

	tk := f.NewTicker(time.Second)
	defer tk.Stop()

	if got := f.Waiters(); got != 0 {
		t.Fatalf("Waiters() = %d with only a ticker, want 0", got)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		f.Sleep(time.Minute)
		close(done)
	}()

	if err := f.BlockUntilContext(ctx, 1); err != nil {
		t.Fatalf("BlockUntilContext error: %v", err)
	}
	if got := f.Waiters(); got != 1 {
		t.Fatalf("Waiters() = %d, want 1 (ticker must not count)", got)
	}

	f.Advance(time.Minute)
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("sleeper did not wake")
	}
	if got := f.Waiters(); got != 0 {
		t.Fatalf("Waiters() = %d after wake, want 0", got)
	}
}

// C must be safe to call concurrently with Reset and Stop.
func TestConcurrentChannelAndReset(t *testing.T) {
	t0 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	f := fakeclock.New(t0)

	timer := f.NewTimer(time.Second)
	tk := f.NewTicker(time.Second)
	defer tk.Stop()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			_ = timer.C()
			_ = tk.C()
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			timer.Reset(time.Second)
			tk.Reset(time.Second)
			f.Advance(time.Second)
			timer.Stop()
		}
	}()
	wg.Wait()
}

func Example() {
	f := fakeclock.New(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
	ch := f.After(time.Second)
	f.Advance(time.Second)
	fmt.Println(<-ch)
	// Output:
	// 2020-01-01 00:00:01 +0000 UTC
}
