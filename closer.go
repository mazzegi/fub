package fub

import "sync"

type Closer interface {
	Close()
}

type CloserCache struct {
	sync.RWMutex
	nextID  uint64
	closers map[uint64]Closer
}

func NewCloserCache() *CloserCache {
	return &CloserCache{
		closers: map[uint64]Closer{},
		nextID:  1,
	}
}

func (cs *CloserCache) Add(c Closer) uint64 {
	cs.Lock()
	defer cs.Unlock()
	id := cs.nextID
	cs.closers[id] = c
	cs.nextID++
	return id
}

func (cs *CloserCache) Remove(id uint64) {
	cs.Lock()
	defer cs.Unlock()
	delete(cs.closers, id)
}

func (cs *CloserCache) CloseAll() {
	cs.Lock()
	defer cs.Unlock()
	for _, c := range cs.closers {
		c.Close()
	}
	cs.closers = map[uint64]Closer{}
}
