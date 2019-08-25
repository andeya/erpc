package pool

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"time"

	"github.com/henrylee2cn/goutil/coarsetime"
)

type (
	// GoPool executes concurrently incoming function via a pool of goroutines
	// in FILO order, i.e. the most recently stopped goroutine will execute the next
	// incoming function.
	//
	// Such a scheme keeps CPU caches hot (in theory).
	GoPool struct {
		maxGoroutinesAmount      int
		maxGoroutineIdleDuration time.Duration

		lock              sync.Mutex
		goroutinesCount   int
		mustStop          bool
		ready             []*goroutineChan
		stopCh            chan struct{}
		goroutineChanPool sync.Pool
	}

	goroutineChan struct {
		lastUseTime time.Time
		ch          chan func()
	}
)

const (
	// DefaultMaxGoroutinesAmount is the default maximum amount of goroutines.
	DefaultMaxGoroutinesAmount = 256 * 1024
	// DefaultMaxGoroutineIdleDuration is the default maximum idle duration of a goroutine.
	DefaultMaxGoroutineIdleDuration = 10 * time.Second
)

// NewGoPool creates a new *GoPool.
// If maxGoroutinesAmount<=0, will use default value.
// If maxGoroutineIdleDuration<=0, will use default value.
func NewGoPool(maxGoroutinesAmount int, maxGoroutineIdleDuration time.Duration) *GoPool {
	gp := new(GoPool)
	if maxGoroutinesAmount <= 0 {
		gp.maxGoroutinesAmount = DefaultMaxGoroutinesAmount
	} else {
		gp.maxGoroutinesAmount = maxGoroutinesAmount
	}
	if maxGoroutineIdleDuration <= 0 {
		gp.maxGoroutineIdleDuration = DefaultMaxGoroutineIdleDuration
	} else {
		gp.maxGoroutineIdleDuration = maxGoroutineIdleDuration
	}
	gp.start()
	return gp
}

// MaxGoroutinesAmount returns the max amount of goroutines.
func (gp *GoPool) MaxGoroutinesAmount() int {
	return gp.maxGoroutinesAmount
}

// MaxGoroutineIdle returns the max idle duration of the goroutine.
func (gp *GoPool) MaxGoroutineIdle() time.Duration {
	return gp.maxGoroutineIdleDuration
}

// start starts GoPool.
func (gp *GoPool) start() {
	if gp.stopCh != nil {
		panic("BUG: GoPool already started")
	}
	gp.stopCh = make(chan struct{})
	stopCh := gp.stopCh
	go func() {
		var scratch []*goroutineChan
		for {
			gp.clean(&scratch)
			select {
			case <-stopCh:
				return
			default:
				time.Sleep(gp.maxGoroutineIdleDuration)
			}
		}
	}()
}

// Stop starts GoPool.
// If calling 'Go' after calling 'Stop', will no longer reuse goroutine.
func (gp *GoPool) Stop() {
	if gp.stopCh == nil {
		panic("BUG: GoPool wasn't started")
	}
	close(gp.stopCh)
	gp.stopCh = nil

	// Stop all the goroutines waiting for incoming materiels.
	// Handle not wait for busy goroutines - they will stop after
	// serving the materiel and noticing gp.mustStop = true.
	gp.lock.Lock()
	ready := gp.ready
	for i, ch := range ready {
		ch.ch <- nil
		ready[i] = nil
	}
	gp.ready = ready[:0]
	gp.mustStop = true
	gp.lock.Unlock()
}

var ErrLack = errors.New("lack of goroutines, because exceeded maxGoroutinesAmount limit.")

// Go executes the function via a goroutine.
// If returns non-nil, the function cannot be executed because exceeded maxGoroutinesAmount limit.
func (gp *GoPool) Go(fn func()) error {
	ch := gp.getCh()
	if ch == nil {
		return ErrLack
	}
	ch.ch <- fn
	return nil
}

// TryGo tries to execute the function via goroutine.
// If there are no concurrent resources, execute it synchronously.
func (gp *GoPool) TryGo(fn func()) {
	if gp.Go(fn) != nil {
		fn()
	}
}

// MustGo always try to use goroutine callbacks
// until execution is complete or the context is canceled.
func (gp *GoPool) MustGo(fn func(), ctx ...context.Context) error {
	if len(ctx) == 0 {
		for gp.Go(fn) != nil {
			runtime.Gosched()
		}
		return nil
	}
	c := ctx[0]
	for {
		select {
		case <-c.Done():
			return c.Err()
		default:
			if gp.Go(fn) == nil {
				return nil
			}
			runtime.Gosched()
		}
	}
}

func (gp *GoPool) clean(scratch *[]*goroutineChan) {
	maxGoroutineIdleDuration := gp.maxGoroutineIdleDuration

	// Clean least recently used goroutines if they didn't serve materiels
	// for more than maxGoroutineIdleDuration.
	currentTime := time.Now()

	gp.lock.Lock()
	ready := gp.ready
	n := len(ready)
	i := 0
	for i < n && currentTime.Sub(ready[i].lastUseTime) > maxGoroutineIdleDuration {
		i++
	}
	*scratch = append((*scratch)[:0], ready[:i]...)
	if i > 0 {
		m := copy(ready, ready[i:])
		for i = m; i < n; i++ {
			ready[i] = nil
		}
		gp.ready = ready[:m]
	}
	gp.lock.Unlock()

	// Notify obsolete goroutines to stop.
	// This notification must be outside the gp.lock, since ch.ch
	// may be blocking and may consume a lot of time if many goroutines
	// are located on non-local CPUs.
	tmp := *scratch
	for i, ch := range tmp {
		ch.ch <- nil
		tmp[i] = nil
	}
}

var goroutineChanCap = func() int {
	// Use blocking goroutineChan if GOMAXPROCS=1.
	// This immediately switches Go to GoroutineFunc, which results
	// in higher performance (under go1.5 at least).
	if runtime.GOMAXPROCS(0) == 1 {
		return 0
	}

	// Use non-blocking goroutineChan if GOMAXPROCS>1,
	// since otherwise the Go caller (Acceptor) may lag accepting
	// new materiels if GoroutineFunc is CPU-bound.
	return 1
}()

func (gp *GoPool) getCh() *goroutineChan {
	var ch *goroutineChan
	createGoroutine := false

	gp.lock.Lock()
	ready := gp.ready
	n := len(ready) - 1
	if n < 0 {
		if gp.goroutinesCount < gp.maxGoroutinesAmount {
			createGoroutine = true
			gp.goroutinesCount++
		}
	} else {
		ch = ready[n]
		ready[n] = nil
		gp.ready = ready[:n]
	}
	gp.lock.Unlock()

	if ch == nil {
		if !createGoroutine {
			return nil
		}
		vch := gp.goroutineChanPool.Get()
		if vch == nil {
			vch = &goroutineChan{
				ch: make(chan func(), goroutineChanCap),
			}
		}
		ch = vch.(*goroutineChan)
		go func() {
			gp.goroutineFunc(ch)
			gp.goroutineChanPool.Put(vch)
		}()
	}
	return ch
}

func (gp *GoPool) release(ch *goroutineChan) bool {
	ch.lastUseTime = coarsetime.FloorTimeNow()
	gp.lock.Lock()
	if gp.mustStop {
		gp.lock.Unlock()
		return false
	}
	gp.ready = append(gp.ready, ch)
	gp.lock.Unlock()
	return true
}

func (gp *GoPool) goroutineFunc(ch *goroutineChan) {
	for fn := range ch.ch {
		if fn == nil {
			break
		}
		fn()
		if !gp.release(ch) {
			break
		}
	}

	gp.lock.Lock()
	gp.goroutinesCount--
	gp.lock.Unlock()
}
