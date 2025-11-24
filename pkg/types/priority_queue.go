package types

import (
	"container/heap"
	"sync"
)

// JobPriority defines job priority levels
type JobPriority int

const (
	PriorityLow    JobPriority = 0
	PriorityNormal JobPriority = 1
	PriorityHigh   JobPriority = 2
)

// PriorityJob wraps a job with priority
type PriorityJob struct {
	Job      *Job
	Priority JobPriority
	Index    int // Index in the heap
}

// PriorityQueue implements a priority queue for jobs
type PriorityQueue struct {
	items []*PriorityJob
	mu    sync.RWMutex
	cond  *sync.Cond
}

// NewPriorityQueue creates a new priority queue
func NewPriorityQueue() *PriorityQueue {
	pq := &PriorityQueue{
		items: make([]*PriorityJob, 0),
	}
	pq.cond = sync.NewCond(&pq.mu)
	heap.Init(pq)
	return pq
}

// Enqueue adds a job to the queue
func (pq *PriorityQueue) Enqueue(job *Job, priority JobPriority) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	item := &PriorityJob{
		Job:      job,
		Priority: priority,
	}

	heap.Push(pq, item)
	pq.cond.Signal() // Wake up waiting goroutines
}

// Dequeue removes and returns the highest priority job
func (pq *PriorityQueue) Dequeue() *Job {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	// Wait if queue is empty
	for pq.Len() == 0 {
		pq.cond.Wait()
	}

	item := heap.Pop(pq).(*PriorityJob)
	return item.Job
}

// TryDequeue attempts to dequeue without blocking
func (pq *PriorityQueue) TryDequeue() (*Job, bool) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if pq.Len() == 0 {
		return nil, false
	}

	item := heap.Pop(pq).(*PriorityJob)
	return item.Job, true
}

// Len returns the number of items in the queue
func (pq *PriorityQueue) Len() int {
	return len(pq.items)
}

// Size returns the current queue size (thread-safe)
func (pq *PriorityQueue) Size() int {
	pq.mu.RLock()
	defer pq.mu.RUnlock()
	return len(pq.items)
}

// Less compares two items (higher priority first)
func (pq *PriorityQueue) Less(i, j int) bool {
	return pq.items[i].Priority > pq.items[j].Priority
}

// Swap swaps two items
func (pq *PriorityQueue) Swap(i, j int) {
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
	pq.items[i].Index = i
	pq.items[j].Index = j
}

// Push implements heap.Interface
func (pq *PriorityQueue) Push(x interface{}) {
	n := len(pq.items)
	item := x.(*PriorityJob)
	item.Index = n
	pq.items = append(pq.items, item)
}

// Pop implements heap.Interface
func (pq *PriorityQueue) Pop() interface{} {
	old := pq.items
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.Index = -1 // for safety
	pq.items = old[0 : n-1]
	return item
}
