package overloader

import (
	"sync/atomic"
	"time"
)

type qpsLimiter struct {
	limit    int32
	tokens   int32
	interval time.Duration
	once     int32
	ticker   *time.Ticker
}

func newQPSLimiter(maxQPS int32, qpsInterval time.Duration) *qpsLimiter {
	once := maxQPS / int32(time.Second/qpsInterval)
	if once == 0 {
		once = 1
	}
	q := &qpsLimiter{
		limit:    maxQPS,
		tokens:   maxQPS,
		interval: qpsInterval,
		once:     once,
		ticker:   time.NewTicker(qpsInterval),
	}
	go q.startTicker()
	return q
}

func (q *qpsLimiter) getLimit() int32 {
	return atomic.LoadInt32(&q.limit)
}

func (q *qpsLimiter) getInterval() time.Duration {
	return q.interval
}

func (q *qpsLimiter) update(maxQPS int32, qpsInterval time.Duration) {
	if maxQPS == q.limit && qpsInterval == q.interval {
		return
	}
	q.limit = maxQPS
	once := maxQPS / int32(time.Second/qpsInterval)
	if once == 0 {
		once = 1
	}
	q.once = once
	if qpsInterval != q.interval {
		q.interval = qpsInterval
		q.stopTicker()
		q.ticker = time.NewTicker(qpsInterval)
		go q.startTicker()
	}
}

func (q *qpsLimiter) take() bool {
	if atomic.LoadInt32(&q.tokens) <= 0 {
		return false
	}
	return atomic.AddInt32(&q.tokens, -1) >= 0
}

func (q *qpsLimiter) startTicker() {
	ch := q.ticker.C
	for range ch {
		q.updateToken()
	}
}

func (q *qpsLimiter) stopTicker() {
	q.ticker.Stop()
}

func (q *qpsLimiter) updateToken() {
	var v int32
	v = atomic.LoadInt32(&q.tokens)
	if v < 0 {
		v = q.once
	} else if v+q.once > q.limit {
		v = q.limit
	} else {
		v = v + q.once
	}
	atomic.StoreInt32(&q.tokens, v)
}
