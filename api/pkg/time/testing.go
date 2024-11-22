package ltime

import (
	"pgregory.net/rapid"
	"time"
)

var times = []string{
	"2020-06-01T00:00:00Z",
	"2020-06-01T06:00:00Z",
	"2020-06-01T12:00:00Z",
	"2020-06-01T18:00:00Z",
}

var durations = []string{
	"1m",
	"5m",
	"10m",
	"30m",
}

var timeSampler *rapid.Generator[time.Time]
var durationSampler *rapid.Generator[time.Duration]

func init() {
	timeGenerators := make([]*rapid.Generator[time.Time], 0)
	for _, time_ := range times {
		parsed, err := time.Parse(time.RFC3339, time_)
		if err != nil {
			panic(err)
		}
		timeGenerators = append(timeGenerators, rapid.Just(parsed))
	}
	timeSampler = rapid.OneOf(timeGenerators...)

	durationGenerators := make([]*rapid.Generator[time.Duration], 0)
	for _, dd := range durations {
		parsed, err := time.ParseDuration(dd)
		if err != nil {
			panic(err)
		}
		durationGenerators = append(durationGenerators, rapid.Just(parsed))
	}
	durationSampler = rapid.OneOf(durationGenerators...)
}

func TestingTimeGenerator() *rapid.Generator[time.Time] {
	return timeSampler
}

func TestingDurationGenerator() *rapid.Generator[time.Duration] {
	return durationSampler
}
