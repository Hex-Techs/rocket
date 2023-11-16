package storage

import (
	"sync"
)

type Store interface {
	Add(key uint, obj interface{}) error
	Update(key uint, obj interface{}) error
	Delete(key uint, obj interface{}) error
	List() []interface{}
	Get(key uint, obj interface{}) (item interface{}, exists bool, err error)
	Resync() error
}

type threadSafeMap struct {
	m map[uint]interface{}
	l sync.RWMutex
}

func (t *threadSafeMap) Add(key uint, obj interface{}) error {
	t.l.Lock()
	defer t.l.Unlock()
	t.m[key] = obj
	return nil
}

func (t *threadSafeMap) Update(key uint, obj interface{}) error {
	t.l.Lock()
	defer t.l.Unlock()
	t.m[key] = obj
	return nil
}

func (t *threadSafeMap) Delete(key uint, obj interface{}) error {
	t.l.Lock()
	defer t.l.Unlock()
	delete(t.m, key)
	return nil
}

func (t *threadSafeMap) List() []interface{} {
	t.l.RLock()
	defer t.l.RUnlock()
	var list []interface{}
	for _, v := range t.m {
		list = append(list, v)
	}
	return list
}

func (t *threadSafeMap) Get(key uint, obj interface{}) (item interface{}, exists bool, err error) {
	t.l.RLock()
	defer t.l.RUnlock()
	item, exists = t.m[key]
	return item, exists, nil
}

func (t *threadSafeMap) Resync() error {
	return nil
}
