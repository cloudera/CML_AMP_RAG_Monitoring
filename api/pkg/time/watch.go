package ltime

import "time"

type Watch interface {
	Now() time.Time
}

type WallWatch struct{}

func (WallWatch) Now() time.Time {
	return time.Now()
}

func NewWallWatch() WallWatch { return WallWatch{} }

type TestingWatch struct {
	Current time.Time
}

func (f *TestingWatch) Now() time.Time {
	return f.Current
}
