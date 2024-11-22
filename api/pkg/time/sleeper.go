package ltime

import (
	"math/rand"
	"time"
)

type Sleeper interface {
	Sleep(duration time.Duration)
}

type WallSleeper struct{}

func (WallSleeper) Sleep(duration time.Duration) {
	time.Sleep(JitteredDuration(duration))
}

var _ Sleeper = WallSleeper{}

func NewWallSleeper() WallSleeper {
	return WallSleeper{}
}

type TestingSleeper struct{}

func (TestingSleeper) Sleep(duration time.Duration) {
}

var _ Sleeper = TestingSleeper{}

func NewTestingSleeper() TestingSleeper {
	return TestingSleeper{}
}

func JitteredDuration(duration time.Duration) time.Duration {
	// Add some jitter to make duration 20% smaller or longer
	return time.Duration(float64(duration) * (0.8 + 0.4*rand.Float64()))
}

func Sleep(duration time.Duration) {
	time.Sleep(JitteredDuration(duration))
}
