package utils

import "testing"

func TestCountString(t *testing.T) {
	c1 := NewCountString(2)
	for i := 0; i < len(countStringSet)+5; i++ {
		t.Logf("c1: %d:%s", i, c1.String())
		c1.Incr()
	}
	c2 := NewCountString(len(countStringSet) + 5)
	t.Log("c2: 0:", c2.String())
	t.Log("c2: 1:", c2.Incr().String())
	for i := 1<<32 - 1; i >= 0; i-- {
		c2.Incr()
	}
	t.Logf("c2: %d:%s", 1<<32-1, c2.String())
	t.Logf("c2: %d:%s", 1<<32, c2.Incr().String())
}
