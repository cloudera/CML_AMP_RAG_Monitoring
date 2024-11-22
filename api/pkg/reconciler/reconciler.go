package reconciler

import "context"

type Reconciler[T Key] interface {
	Name() string
	Reboot(ctx context.Context)
	Resync(ctx context.Context, queue *ReconcileQueue[T])
	Reconcile(ctx context.Context, items []ReconcileItem[T])
}
