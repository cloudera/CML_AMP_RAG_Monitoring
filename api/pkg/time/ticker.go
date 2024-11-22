package ltime

import "time"

type Ticker interface {
	Channel() <-chan time.Time
	Close()
}

type WallTicker struct {
	ticker *time.Ticker
}

func (w *WallTicker) Channel() <-chan time.Time {
	return w.ticker.C
}

func (w *WallTicker) Close() {
	w.ticker.Stop()
}

func NewWallTicker(duration time.Duration) *WallTicker {
	return &WallTicker{time.NewTicker(duration)}
}

var _ Ticker = &WallTicker{}

type TestingTicker struct {
	c      chan time.Time
	closed bool
}

func NewTestingTicker() *TestingTicker {
	ret := &TestingTicker{
		c: make(chan time.Time),
	}

	go func() {
		for !ret.closed {
			ret.c <- time.Now()
		}
	}()

	return ret
}

func (t *TestingTicker) Channel() <-chan time.Time {
	return t.c
}

func (t *TestingTicker) Close() {
	t.closed = true
	// Drain the goroutine
	select {
	case <-t.c:
	default:
	}
}

var _ Ticker = &TestingTicker{}
