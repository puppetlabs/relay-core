package secretauth

import (
	"log"
	"time"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
)

const defaultMaxRetries = 10

// worker is responsible for managing the kubernetes object workqueue for
// a resource and dispatching objects derived from change events to handler.
type worker struct {
	workqueue  workqueue.RateLimitingInterface
	maxRetries int
	handler    func(string) error
}

func (w *worker) add(key string) {
	w.workqueue.Add(key)
}

func (w *worker) processNextWorkItem() bool {
	key, shutdown := w.workqueue.Get()

	if shutdown {
		return false
	}

	defer w.workqueue.Done(key)

	err := w.handler(key.(string))
	w.handleError(err, key)

	return true
}

func (w *worker) handleError(err error, key interface{}) {
	if err == nil {
		w.workqueue.Forget(key)

		return
	}

	log.Println("handling error", err)

	if w.workqueue.NumRequeues(key) < w.maxRetries {
		w.workqueue.AddRateLimited(key)

		return
	}

	utilruntime.HandleError(err)

	w.workqueue.Forget(key)
}

func (w *worker) work() {
	for w.processNextWorkItem() {
	}
}

func (w *worker) run(threads int, stopCh chan struct{}) {
	for i := 0; i < threads; i++ {
		go wait.Until(w.work, time.Second, stopCh)
	}
}

func (w *worker) shutdown() {
	w.workqueue.ShutDown()
}

func newWorker(resource string, handler func(string) error, maxRetries int) *worker {
	mr := defaultMaxRetries
	if maxRetries > 0 {
		mr = maxRetries
	}

	q := workqueue.NewNamedRateLimitingQueue(
		workqueue.DefaultControllerRateLimiter(), resource)

	return &worker{
		maxRetries: mr,
		workqueue:  q,
		handler:    handler,
	}
}
