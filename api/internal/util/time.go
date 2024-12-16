package util

import "time"

func TimeStamp(millis int64) time.Time {
	seconds := millis / 1000
	nanoseconds := (millis % 1000) * 1e6
	return time.Unix(seconds, nanoseconds)
}
