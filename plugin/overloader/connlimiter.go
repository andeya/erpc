package overloader

import "sync/atomic"

type connLimiter struct {
	lim int32
	now int32
	tmp int32
}

func newConnLimiter(maxConn int32) *connLimiter {
	l := &connLimiter{}
	l.update(maxConn)
	return l
}

func (c *connLimiter) update(maxConn int32) {
	atomic.StoreInt32(&c.lim, maxConn)
}

func (c *connLimiter) take() bool {
	x := atomic.AddInt32(&c.tmp, 1)
	if x <= atomic.LoadInt32(&c.lim) {
		atomic.AddInt32(&c.now, 1)
		return true
	}
	atomic.AddInt32(&c.tmp, -1)
	return false
}

func (c *connLimiter) release() {
	atomic.AddInt32(&c.now, -1)
	atomic.AddInt32(&c.tmp, -1)
}

func (c *connLimiter) getLimit() int32 {
	return atomic.LoadInt32(&c.lim)
}

func (c *connLimiter) getNow() int32 {
	return atomic.LoadInt32(&c.now)
}
