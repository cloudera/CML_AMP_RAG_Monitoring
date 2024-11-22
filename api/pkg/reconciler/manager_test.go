package reconciler

import (
	"context"
	"testing"
	"time"
)

type testReconciler struct {
	rebootTimes       []time.Time
	resyncTimes       []time.Time
	reconcileTimes    []time.Time
	resyncSignalAfter int
	resyncSignal      chan bool
}

func (t *testReconciler) Name() string {
	return "test"
}

func (t *testReconciler) Reboot(_ context.Context) {
	t.rebootTimes = append(t.rebootTimes, time.Now())
}

func (t *testReconciler) Resync(_ context.Context, queue *ReconcileQueue[int64]) {
	t.resyncTimes = append(t.resyncTimes, time.Now())
	if t.resyncSignalAfter == len(t.resyncTimes) {
		t.resyncSignal <- true
	}
	// TODO implement queue
}

func (t *testReconciler) Reconcile(_ context.Context, items []ReconcileItem[int64]) {
	//TODO implement items
	t.reconcileTimes = append(t.reconcileTimes, time.Now())
}

var _ Reconciler[int64] = &testReconciler{}

func TestManagerStartFinish(t *testing.T) {
	config, err := NewConfig(100*time.Millisecond, 1, 1)
	if err != nil {
		t.Fatalf("failed to create config: %v", err)
	}

	r := &testReconciler{
		resyncSignal:      make(chan bool),
		resyncSignalAfter: 10,
	}
	manager := NewManager(context.Background(), config, r)
	manager.Start()
	<-r.resyncSignal
	manager.Finish()
}
