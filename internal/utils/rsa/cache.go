package rsa

import (
	"container/list"
	"sync"
)

// LRU cache
type cache struct {
	cap int

	mu  sync.Mutex
	ll  *list.List
	tab map[[32]byte]*entry
}

type entry struct {
	id   [32]byte
	pk   *PubKey
	ref  int
	elem *list.Element
}

func newCache(c int) *cache {
	if c < 1 {
		panic("new lru cache capacity could not be less than 1")
	}

	return &cache{
		cap: c,
		ll:  list.New(),
		tab: make(map[[32]byte]*entry),
	}
}

func (m *cache) close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for e := m.ll.Front(); e != nil; e = e.Next() {
		en := e.Value.(*entry)
		if en.pk != nil {
			en.pk.Close()
		}
	}

	m.ll.Init()
	m.tab = make(map[[32]byte]*entry)
}

func (m *cache) getElement(id [32]byte) *entry {
	m.mu.Lock()
	defer m.mu.Unlock()

	// cache hit
	if en := m.tab[id]; en != nil {
		en.ref++
		m.ll.MoveToFront(en.elem)
		return en
	}

	return nil
}

func (m *cache) newElement(id [32]byte, pk *PubKey) *entry {
	en := &entry{
		id:  id,
		pk:  pk,
		ref: 1,
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// cache miss
	en.elem = m.ll.PushFront(en)
	m.tab[id] = en

	// remove elements if cache overflowing
	m.evict()

	return en
}

func (m *cache) evict() {
	if m.ll.Len() > m.cap {
		var tail *list.Element
		if tail = m.ll.Back(); tail == nil {
			return
		}

		en := tail.Value.(*entry)
		if en.ref > 0 { // if last element in the list is busy now...
			return // todo : add some warnings here
		}

		m.ll.Remove(tail)
		delete(m.tab, en.id)

		if en.pk != nil {
			en.pk.Close()
		}
	}
}

func (m *cache) release(en *entry) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if en.ref > 0 {
		en.ref--
	}

	if m.ll.Len() > m.cap {
		m.evict()
	}
}
