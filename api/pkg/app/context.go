package app

import (
	"context"
	"time"
)

const BACKGROUND_TIMEOUT_DURATION = time.Minute

func BackgroundTimeoutContext() (context.Context, context.CancelFunc) {
	return BackgroundTimeoutContextDuration(BACKGROUND_TIMEOUT_DURATION)
}

func BackgroundTimeoutContextDuration(duration time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), duration)
}
