package loop

import (
	"context"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type node struct {
	data Task
	next *node
}

// Queue implements a job queue for the event loop
type Queue struct {
	name     string
	loopName string
	head     *node
	tail     *node
	count    int
	lock     *sync.Mutex
	waitCh   chan struct{}
}

// NewQueue creates a new queue
func NewQueue(loopName, name string) *Queue {
	return &Queue{
		lock:     &sync.Mutex{},
		waitCh:   make(chan struct{}),
		name:     name,
		loopName: loopName,
	}
}

// Len returns the number of jobs queued
func (q *Queue) Len() int {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.count
}

// Push pushes a new job onto the queue
func (q *Queue) Push(item Task) {
	q.lock.Lock()
	defer q.lock.Unlock()

	n := &node{data: item}
	if q.tail == nil {
		q.tail = n
		q.head = n
	} else {
		q.tail.next = n
		q.tail = n
	}
	q.count++
	totalJobs.With(prometheus.Labels{"loop": q.loopName, "queue": q.name}).Inc()
	queuedJobs.With(prometheus.Labels{"loop": q.loopName, "queue": q.name}).Inc()

	// if there's someone waiting on PopWait(),
	// try to notify it
	select {
	case q.waitCh <- struct{}{}:
	default:
	}
}

// Pop returns the next task to execute from the queue or nil
// if the queue is empty
func (q *Queue) Pop() Task {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.head == nil {
		return nil
	}

	n := q.head
	q.head = n.next

	if q.head == nil {
		q.tail = nil
	}
	q.count--
	queuedJobs.With(prometheus.Labels{"loop": q.loopName, "queue": q.name}).Dec()

	return n.data
}

// PopWait returns the next job from the queue and will
// block until either the context is cancelled or a job
// becomes available
func (q *Queue) PopWait(ctx context.Context) Task {
	next := q.Pop()
	if next == nil {
		select {
		case <-q.waitCh:
			return q.PopWait(ctx)
		case <-ctx.Done():
			return nil
		}
	}
	return next
}
