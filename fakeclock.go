package fakeclock

import (
	"context"
	"sync"
	"time"
)

// Clock is the interface implemented by both [Real] and [Fake].
// It provides the minimal time-related operations needed for testing.
type Clock interface {
	Now() time.Time
	Since(t time.Time) time.Duration
	After(d time.Duration) <-chan time.Time
	Sleep(d time.Duration)
	NewTimer(d time.Duration) Timer
	NewTicker(d time.Duration) Ticker
}

// Timer is the interface returned by Clock.NewTimer.
type Timer interface {
	C() <-chan time.Time
	Stop() bool
	Reset(d time.Duration) bool
}

// Ticker is the interface returned by Clock.NewTicker.
type Ticker interface {
	C() <-chan time.Time
	Stop()
	Reset(d time.Duration)
}

// realClock is the Clock implementation that wraps the standard library time package.
type realClock struct{}

// Real returns a Clock backed by the real system clock.
func Real() Clock {
	return realClock{}
}

func (realClock) Now() time.Time {
	return time.Now()
}

func (realClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

func (realClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

func (realClock) Sleep(d time.Duration) {
	time.Sleep(d)
}

func (realClock) NewTimer(d time.Duration) Timer {
	return &realTimer{Timer: time.NewTimer(d)}
}

func (realClock) NewTicker(d time.Duration) Ticker {
	return &realTicker{Ticker: time.NewTicker(d)}
}

// eventKind distinguishes timer and ticker events.
type eventKind uint8

const (
	eventTimer eventKind = iota
	eventTicker
)

// event represents a pending timer or ticker fire.
type event struct {
	fireTime time.Time
	seq      uint64 // creation sequence for tie-breaking
	ch       chan time.Time
	interval time.Duration // zero for timers, positive for tickers
}

// sleeper represents a goroutine blocked in Fake.Sleep.
type sleeper struct {
	deadline time.Time
	done     bool
}

// Fake is a manual clock whose time only advances when the test calls
// Advance or Set. All methods are safe for concurrent use.
type Fake struct {
	mu       sync.Mutex
	cond     *sync.Cond
	now      time.Time
	seq      uint64
	events   []*event // sorted by (fireTime, seq)
	sleepers []*sleeper
}

// New creates a Fake clock starting at the given time.
func New(start time.Time) *Fake {
	f := &Fake{now: start}
	f.cond = sync.NewCond(&f.mu)
	return f
}

// Now returns the current time of the fake clock.
func (f *Fake) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

// Since returns the duration elapsed since t according to the fake clock.
func (f *Fake) Since(t time.Time) time.Duration {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now.Sub(t)
}

// After waits for the duration to elapse and then sends the current time
// on the returned channel. If d <= 0, the channel fires on the next
// Advance(0).
func (f *Fake) After(d time.Duration) <-chan time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	ch := make(chan time.Time, 1)
	// A non-positive duration fires on the next Advance(0).
	fireTime := f.now
	if d > 0 {
		fireTime = f.now.Add(d)
	}
	e := &event{
		fireTime: fireTime,
		seq:      f.seq,
		ch:       ch,
	}
	f.seq++
	f.insertEvent(e)
	f.cond.Broadcast()
	return ch
}

// Sleep blocks until the fake clock has advanced by d. If d <= 0,
// it returns immediately.
func (f *Fake) Sleep(d time.Duration) {
	if d <= 0 {
		return
	}
	f.mu.Lock()
	s := &sleeper{deadline: f.now.Add(d)}
	f.sleepers = append(f.sleepers, s)
	f.cond.Broadcast()
	for !s.done {
		f.cond.Wait()
	}
	f.mu.Unlock()
}

// NewTimer creates a new Timer that will send the current time on its
// channel after at least duration d. If d <= 0, it fires on the next
// Advance(0).
func (f *Fake) NewTimer(d time.Duration) Timer {
	f.mu.Lock()
	defer f.mu.Unlock()
	ch := make(chan time.Time, 1)
	fireTime := f.now
	if d > 0 {
		fireTime = f.now.Add(d)
	}
	e := &event{
		fireTime: fireTime,
		seq:      f.seq,
		ch:       ch,
	}
	f.seq++
	f.insertEvent(e)
	f.cond.Broadcast()
	return &fakeTimer{fake: f, ch: ch, event: e}
}

// NewTicker returns a new Ticker containing a channel that will send
// the current time on the channel after each tick. The period must be
// positive; NewTicker panics with "fakeclock: non-positive ticker interval"
// if d <= 0.
func (f *Fake) NewTicker(d time.Duration) Ticker {
	if d <= 0 {
		panic("fakeclock: non-positive ticker interval")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	ch := make(chan time.Time, 1)
	e := &event{
		fireTime: f.now.Add(d),
		seq:      f.seq,
		ch:       ch,
		interval: d,
	}
	f.seq++
	f.insertEvent(e)
	f.cond.Broadcast()
	return &fakeTicker{fake: f, ch: ch, event: e}
}

// Advance moves the fake clock forward by d, firing all timers, tickers,
// and sleepers whose deadlines fall within the interval. If d is negative,
// Advance panics with "fakeclock: negative advance".
func (f *Fake) Advance(d time.Duration) {
	if d < 0 {
		panic("fakeclock: negative advance")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.advanceLocked(f.now.Add(d))
}

// Set moves the fake clock to the given time, firing all due events.
// If t is before the current time, Set panics with
// "fakeclock: time moved backwards".
func (f *Fake) Set(t time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.advanceLocked(t)
}

// Waiters returns the number of goroutines blocked in Sleep plus the
// number of pending timers and After channels.
func (f *Fake) Waiters() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.waitersLocked()
}

// BlockUntilContext waits until at least n waiters are present or the
// context is done. It returns ctx.Err() if the context is cancelled.
func (f *Fake) BlockUntilContext(ctx context.Context, n int) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	stop := make(chan struct{})
	defer close(stop)
	go func() {
		select {
		case <-ctx.Done():
			f.mu.Lock()
			f.cond.Broadcast()
			f.mu.Unlock()
		case <-stop:
		}
	}()

	for f.waitersLocked() < n {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		f.cond.Wait()
	}
	return nil
}

// insertEvent inserts e into f.events maintaining sort order by fireTime
// then seq.
func (f *Fake) insertEvent(e *event) {
	i := 0
	for i < len(f.events) {
		existing := f.events[i]
		if existing.fireTime.After(e.fireTime) ||
			(existing.fireTime.Equal(e.fireTime) && existing.seq > e.seq) {
			break
		}
		i++
	}
	f.events = append(f.events, nil)
	copy(f.events[i+1:], f.events[i:])
	f.events[i] = e
}

// waitersLocked returns the waiter count assuming f.mu is held.
func (f *Fake) waitersLocked() int {
	count := 0
	for _, e := range f.events {
		if e.interval == 0 { // pending timers and After channels only
			count++
		}
	}
	for _, s := range f.sleepers {
		if !s.done {
			count++
		}
	}
	return count
}

// advanceLocked advances the clock to target, firing all due events.
// It assumes f.mu is held. If target is before the current time, it
// panics with "fakeclock: time moved backwards".
func (f *Fake) advanceLocked(target time.Time) {
	if target.Before(f.now) {
		panic("fakeclock: time moved backwards")
	}

	for {
		// Find the earliest due time among events and sleepers.
		var t time.Time
		hasDue := false

		if len(f.events) > 0 && !f.events[0].fireTime.After(target) {
			t = f.events[0].fireTime
			hasDue = true
		}

		// Check sleepers.
		var sleeperDeadline time.Time
		hasSleeperDue := false
		for _, s := range f.sleepers {
			if !s.done && !s.deadline.After(target) {
				if !hasSleeperDue || s.deadline.Before(sleeperDeadline) {
					sleeperDeadline = s.deadline
					hasSleeperDue = true
				}
			}
		}
		if hasSleeperDue && (!hasDue || sleeperDeadline.Before(t)) {
			t = sleeperDeadline
			hasDue = true
		}

		if !hasDue {
			break
		}

		f.now = t

		// Fire all events at this time.
		for len(f.events) > 0 && f.events[0].fireTime.Equal(t) {
			e := f.events[0]
			f.events = f.events[1:]
			if e.interval > 0 { // ticker
				select {
				case e.ch <- t:
				default: // buffer full, drop tick
				}
				// Reschedule next tick.
				e.fireTime = e.fireTime.Add(e.interval)
				e.seq = f.seq
				f.seq++
				f.insertEvent(e)
			} else { // timer
				select {
				case e.ch <- t:
				default: // buffer full, drop value rather than deadlock
				}
			}
		}

		// Wake sleepers whose deadlines have passed.
		for _, s := range f.sleepers {
			if !s.done && !s.deadline.After(t) {
				s.done = true
			}
		}

		f.cond.Broadcast()
	}

	if f.now.Before(target) {
		f.now = target
	}
}
