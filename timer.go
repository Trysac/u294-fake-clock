package fakeclock

import "time"

// fakeTimer implements Timer for the fake clock. The channel is stored
// immutably on the wrapper so C() never races with Reset, which swaps
// the pending event.
type fakeTimer struct {
	fake  *Fake
	ch    chan time.Time
	event *event
}

func (t *fakeTimer) C() <-chan time.Time {
	return t.ch
}

// Stop prevents the timer from firing if it has not already fired.
// It returns true if the timer was still pending, false otherwise.
func (t *fakeTimer) Stop() bool {
	t.fake.mu.Lock()
	defer t.fake.mu.Unlock()
	for i, e := range t.fake.events {
		if e == t.event {
			t.fake.events = append(t.fake.events[:i], t.fake.events[i+1:]...)
			t.fake.cond.Broadcast()
			return true
		}
	}
	return false
}

// Reset changes the timer to expire after duration d. It returns true
// if the timer was successfully rearmed (always the case). If d <= 0,
// the timer fires on the next Advance(0).
func (t *fakeTimer) Reset(d time.Duration) bool {
	t.fake.mu.Lock()
	defer t.fake.mu.Unlock()

	// Remove the existing event if it is still pending.
	for i, e := range t.fake.events {
		if e == t.event {
			t.fake.events = append(t.fake.events[:i], t.fake.events[i+1:]...)
			break
		}
	}

	// Discard a value left in the channel by a previous firing so the
	// rearmed timer delivers its own deadline.
	select {
	case <-t.ch:
	default:
	}

	fireTime := t.fake.now
	if d > 0 {
		fireTime = t.fake.now.Add(d)
	}
	newEvent := &event{
		fireTime: fireTime,
		seq:      t.fake.seq,
		ch:       t.ch,
	}
	t.fake.seq++
	t.fake.insertEvent(newEvent)
	t.event = newEvent
	t.fake.cond.Broadcast()
	return true
}

// fakeTicker implements Ticker for the fake clock. The channel is stored
// immutably on the wrapper so C() never races with Reset.
type fakeTicker struct {
	fake  *Fake
	ch    chan time.Time
	event *event
}

func (t *fakeTicker) C() <-chan time.Time {
	return t.ch
}

// Stop turns off the ticker. After Stop, no more ticks will be sent.
func (t *fakeTicker) Stop() {
	t.fake.mu.Lock()
	defer t.fake.mu.Unlock()
	for i, e := range t.fake.events {
		if e == t.event {
			t.fake.events = append(t.fake.events[:i], t.fake.events[i+1:]...)
			break
		}
	}
	t.fake.cond.Broadcast()
}

// Reset stops a ticker and resets its period to the specified duration.
// The next tick will arrive after d. It panics if d <= 0 with
// "fakeclock: non-positive ticker interval".
func (t *fakeTicker) Reset(d time.Duration) {
	if d <= 0 {
		panic("fakeclock: non-positive ticker interval")
	}
	t.fake.mu.Lock()
	defer t.fake.mu.Unlock()

	// Remove existing event if still pending.
	for i, e := range t.fake.events {
		if e == t.event {
			t.fake.events = append(t.fake.events[:i], t.fake.events[i+1:]...)
			break
		}
	}

	newEvent := &event{
		fireTime: t.fake.now.Add(d),
		seq:      t.fake.seq,
		ch:       t.ch,
		interval: d,
	}
	t.fake.seq++
	t.fake.insertEvent(newEvent)
	t.event = newEvent
	t.fake.cond.Broadcast()
}

// realTimer adapts *time.Timer to the Timer interface.
type realTimer struct {
	*time.Timer
}

func (rt *realTimer) C() <-chan time.Time {
	return rt.Timer.C
}

func (rt *realTimer) Stop() bool {
	return rt.Timer.Stop()
}

func (rt *realTimer) Reset(d time.Duration) bool {
	return rt.Timer.Reset(d)
}

// realTicker adapts *time.Ticker to the Ticker interface.
type realTicker struct {
	*time.Ticker
}

func (rt *realTicker) C() <-chan time.Time {
	return rt.Ticker.C
}

func (rt *realTicker) Stop() {
	rt.Ticker.Stop()
}

func (rt *realTicker) Reset(d time.Duration) {
	rt.Ticker.Reset(d)
}
