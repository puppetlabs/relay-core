package prometheus

import (
	"sync"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/puppetlabs/horsehead/v2/instrumentation/metrics/collectors"
)

type Timer struct {
	vector prom.ObserverVec
	timers map[*collectors.TimerHandle]*prom.Timer
	labels []collectors.Label

	sync.RWMutex
}

func (t *Timer) WithLabels(labels ...collectors.Label) collectors.Timer {
	return &Timer{
		vector: t.vector,
		labels: labels,
		timers: make(map[*collectors.TimerHandle]*prom.Timer),
	}
}

func (t *Timer) Start() *collectors.TimerHandle {
	t.Lock()
	defer t.Unlock()

	h := &collectors.TimerHandle{}

	promt := prom.NewTimer(prom.ObserverFunc(func(v float64) {
		// we can change the label values while a timer is in flight. this allows us to capture
		// context about what happened inside a callback.
		t.vector.With(convertLabels(t.labels)).Observe(v)
	}))

	t.timers[h] = promt

	return h
}

func (t *Timer) ObserveDuration(h *collectors.TimerHandle, labels ...collectors.Label) {
	t.RLock()
	defer t.RUnlock()

	t.labels = labels

	if promt, ok := t.timers[h]; ok {
		promt.ObserveDuration()
	}
}

func NewTimer(vector prom.ObserverVec) *Timer {
	return &Timer{
		vector: vector,
		timers: make(map[*collectors.TimerHandle]*prom.Timer),
	}
}
