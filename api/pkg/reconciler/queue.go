package reconciler

import (
	ltime "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/time"
	"sync"
	"time"
)

type Key interface {
	int64 | uint64 | string
}

type ReconcileQueue[T Key] struct {
	Pending        map[T]struct{}
	running        map[T]struct{}
	toRetry        map[T]time.Time
	wakeup         chan bool
	shutdown       chan bool
	isShuttingDown bool
	lock           sync.Mutex
}

type ReconcileItemCallback func(error)

type ReconcileItem[T Key] struct {
	ID       T
	Callback ReconcileItemCallback
}

func NewReconcileQueue[T Key]() *ReconcileQueue[T] {
	q := &ReconcileQueue[T]{
		Pending:  make(map[T]struct{}),
		running:  make(map[T]struct{}),
		toRetry:  make(map[T]time.Time),
		wakeup:   make(chan bool, 1),
		shutdown: make(chan bool, 1),
	}

	go q.runRetry()

	return q
}

func (q *ReconcileQueue[T]) runRetry() {
	// Moves toRetry -> Pending
	for {
		select {
		case <-q.shutdown:
			q.isShuttingDown = true

			// Wake up the queue to see the shutdown
			q.wakeup <- true
			return
		default:
		}

		ltime.Sleep(1 * time.Second)

		q.lock.Lock()
		for id, retryTime := range q.toRetry {
			if time.Now().After(retryTime) {
				q.Pending[id] = struct{}{}
				delete(q.toRetry, id)

				// Wake up the queue
				select {
				case q.wakeup <- true:
				default:
				}
			}
		}
		q.lock.Unlock()
	}
}

func (q *ReconcileQueue[T]) Add(id T) {
	// Moves nil -> Pending

	// Check that the id is not already in the queue
	q.lock.Lock()
	defer q.lock.Unlock()
	if _, ok := q.Pending[id]; ok {
		return
	}
	if _, ok := q.running[id]; ok {
		return
	}
	if _, ok := q.toRetry[id]; ok {
		return
	}
	// Add to the queue
	q.Pending[id] = struct{}{}

	// Wake up the queue
	select {
	case q.wakeup <- true:
	default:
	}
}

func (q *ReconcileQueue[T]) Pop(max int) []ReconcileItem[T] {
	// Moves Pending -> running
	ret := make([]ReconcileItem[T], 0)

	// Get the lock
	q.lock.Lock()

	// Get elements from the Pending queue
	for id := range q.Pending {
		ret = append(ret, ReconcileItem[T]{
			ID:       id,
			Callback: q.getCallback(id),
		})

		if len(ret) == max {
			break
		}
	}

	// If we got no elements, park and try again later
	if len(ret) == 0 {
		q.lock.Unlock()
		<-q.wakeup
		if q.isShuttingDown {
			return ret
		}
		return q.Pop(max)
	}

	defer q.lock.Unlock()
	// Remove the elements we got from the Pending queue
	for _, item := range ret {
		delete(q.Pending, item.ID)
		q.running[item.ID] = struct{}{}
	}

	return ret
}

func (q *ReconcileQueue[T]) getCallback(id T) ReconcileItemCallback {
	// Moves running -> nil|toRetry
	callback := func(err error) {
		q.lock.Lock()
		defer q.lock.Unlock()
		// Remove from the list of running
		delete(q.running, id)
		// Add to the list to retry
		if err != nil {
			q.toRetry[id] = time.Now().Add(ltime.JitteredDuration(5000 * time.Millisecond))
		}
	}

	return callback
}
