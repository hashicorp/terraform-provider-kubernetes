// Copyright (c) HashiCorp, Inc.
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

type keyedCache[K comparable, V any] struct {
	m sync.Map
}

func (c *keyedCache[K, V]) Get(k K, f func() (V, error)) (V, error) {
	ce, _ := c.m.LoadOrStore(k, &cache[V]{})
	return ce.(*cache[V]).Get(f)
}
