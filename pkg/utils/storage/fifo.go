package storage

import (
	"sync"
	"time"
)

type EventType string

const (
	CreateEvent EventType = "create"
	UpdateEvent EventType = "update"
	DeleteEvent EventType = "delete"
)

type ResourceEvent struct {
	ID           uint
	ResourceName string
	CreateAt     time.Time
	UpdateAt     time.Time
	DeleteAt     time.Time
	Event        EventType
}

type StoreQueue interface {
	Add(item interface{})
	Pop() interface{}
	Size() int
	IsEmpty() bool
}

type fifo struct {
	lock        sync.RWMutex
	cond        sync.Cond
	items       []interface{}
	damaged     bool
	damagedTime time.Time
}

// NewQueue creates a new queue
func NewQueue(size int) StoreQueue {
	if size <= 0 {
		size = 30
	}
	return &fifo{
		items: make([]interface{}, 0, size),
	}
}

// Add adds an item to the end of the queue
func (q *fifo) Add(item interface{}) {
	q.lock.Lock()
	q.items = append(q.items, item)
	if len(q.items) > cap(q.items) {
		q.damaged = true
		q.damagedTime = time.Now()
	}
	q.lock.Unlock()
}

// Pop removes an item from the start of the queue
func (q *fifo) Pop() interface{} {
	q.lock.Lock()
	defer q.lock.Unlock()
	if len(q.items) == 0 {
		return nil
	}
	item, items := q.items[0], q.items[1:]
	q.items = items
	return item
}

// IsEmpty checks if the queue is empty
func (q *fifo) IsEmpty() bool {
	q.lock.RLock()
	defer q.lock.RUnlock()
	return len(q.items) == 0
}

// Size returns the number of items in the queue
func (q *fifo) Size() int {
	q.lock.RLock()
	defer q.lock.RUnlock()
	return len(q.items)
}
