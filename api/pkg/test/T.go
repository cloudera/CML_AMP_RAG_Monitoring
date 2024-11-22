package ltest

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

type T interface {
	Helper()
	Fatalf(format string, args ...interface{})
	Cleanup(func())
	assert.TestingT
}

func NewRapidT(t *rapid.T) *RapidT {
	return &RapidT{
		T: t,
	}
}

type RapidT struct {
	*rapid.T
	cleanups []func()
}

func (r *RapidT) Helper() {
}

func (r *RapidT) Fatalf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args...))
}

func (r *RapidT) Errorf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

func (r *RapidT) Cleanup(f func()) {
	r.cleanups = append(r.cleanups, f)
}

func (r *RapidT) RunCleanup() {
	for _, f := range r.cleanups {
		f()
	}
}

var _ T = &RapidT{}

func NewMainT() *MainT {
	return &MainT{}
}

type MainT struct {
	cleanups []func()
}

func (m *MainT) Errorf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

func (m *MainT) Helper() {
}

func (m *MainT) Fatalf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args...))
}

func (m *MainT) Cleanup(f func()) {
	m.cleanups = append(m.cleanups, f)
}

func (m *MainT) RunCleanup() {
	for _, f := range m.cleanups {
		f()
	}
}

var _ T = &MainT{}
