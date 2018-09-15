package utils

import (
	"github.com/henrylee2cn/goutil"
)

var countStringSet = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")

// CountString self-increasing string counter
type CountString struct {
	indexes []int
	str     string
}

// NewCountString creates a self-increasing string counter.
// NOTE: panic if maxLen<=0
func NewCountString(maxLen int) *CountString {
	if maxLen > len(countStringSet) {
		maxLen = len(countStringSet)
	} else if maxLen <= 0 {
		panic("NewCountString: maxLen must be greater than 0!")
	}
	return &CountString{
		indexes: make([]int, 1, maxLen),
	}
}

// Incr increases 1.
// NOTE: non-concurrent security.
func (c *CountString) Incr() *CountString {
	c.str = ""
	var ok bool
	for i, j := range c.indexes {
		j++
		if j < len(countStringSet) {
			c.indexes[i] = j
			ok = true
			break
		}
		c.indexes[i] = 0
	}
	if !ok {
		if len(c.indexes) < cap(c.indexes) {
			c.indexes = append(c.indexes, 0)
		} else {
			// to zero
			c.indexes = make([]int, 1, cap(c.indexes))
		}
	}
	return c
}

// String returns the string.
// NOTE: non-concurrent security
func (c *CountString) String() string {
	if c.str == "" {
		var b = make([]byte, 0, len(c.indexes))
		for i := len(c.indexes) - 1; i >= 0; i-- {
			b = append(b, countStringSet[c.indexes[i]])
		}
		c.str = goutil.BytesToString(b)
	}
	return c.str
}
