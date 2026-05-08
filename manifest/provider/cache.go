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
