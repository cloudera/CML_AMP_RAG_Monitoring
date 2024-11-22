package ltime

import "time"

func ToIntTimeRange(timeRange []time.Time) []int64 {
	intTimeRange := make([]int64, len(timeRange))
	for i, t := range timeRange {
		intTimeRange[i] = t.Unix()
	}
	return intTimeRange
}

func FromIntTimeRange(intTimeRange []int64) []time.Time {
	timeRange := make([]time.Time, len(intTimeRange))
	for i, t := range intTimeRange {
		timeRange[i] = time.Unix(t, 0)
	}
	return timeRange
}
