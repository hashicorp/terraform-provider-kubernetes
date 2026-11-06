// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import "sync"

type cache[T any] struct {
	once  sync.Once
	value T
	err   error
}

func (c *cache[T]) Get(f func() (T, error)) (T, error) {
	c.once.Do(func() {
		c.value, c.err = f()
	})
	return c.value, c.err
}

type keyedCache[T any] struct {
	mu sync.Mutex
	m  map[uint64]*cache[T]
}

func (kc *keyedCache[T]) Get(key uint64, f func() (T, error)) (T, error) {
	kc.mu.Lock()
	c, ok := kc.m[key]
	if !ok {
		c = &cache[T]{}
		if kc.m == nil {
			kc.m = make(map[uint64]*cache[T])
		}
		kc.m[key] = c
	}
	kc.mu.Unlock()
	return c.Get(f)
}
