package pool

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/henrylee2cn/goutil/coarsetime"
)

// ResPool is a high availability/high concurrent resource pool, which automatically manages the number of resources.
// So it is similar to database/sql's db pool.
//
// ResPool is a pool of zero or more underlying avatar(resource).
// It's safe for concurrent use by multiple goroutines.
// ResPool creates and frees resource automatically;
// it also maintains a free pool of idle avatar(resource).
type ResPool interface {
	// Name returns the name.
	Name() string
	// Get returns a object in Resource type.
	Get() (Resource, error)
	// GetContext returns a object in Resource type.
	// Support context cancellation.
	GetContext(context.Context) (Resource, error)
	// Put gives a resource back to the ResPool.
	// If error is not nil, close the avatar.
	Put(Resource, error)
	// Callback callbacks your handle function, returns the error of getting resource or handling.
	// Support recover panic.
	Callback(func(Resource) error) error
	// Callback callbacks your handle function, returns the error of getting resource or handling.
	// Support recover panic and context cancellation.
	CallbackContext(context.Context, func(Resource) error) error
	// SetMaxLifetime sets the maximum amount of time a resource may be reused.
	//
	// Expired resource may be closed lazily before reuse.
	//
	// If d <= 0, resource are reused forever.
	SetMaxLifetime(d time.Duration)
	// SetMaxIdle sets the maximum number of resources in the idle
	// resource pool.
	//
	// If SetMaxIdle is greater than 0 but less than the new MaxIdle
	// then the new MaxIdle will be reduced to match the SetMaxIdle limit
	//
	// If n <= 0, no idle resources are retained.
	SetMaxIdle(n int)
	// SetMaxOpen sets the maximum number of open resources.
	//
	// If MaxIdle is greater than 0 and the new MaxOpen is less than
	// MaxIdle, then MaxIdle will be reduced to match the new
	// MaxOpen limit
	//
	// If n <= 0, then there is no limit on the number of open resources.
	// The default is 0 (unlimited).
	SetMaxOpen(n int)
	// Close closes the ResPool, releasing any open resources.
	//
	// It is rare to close a ResPool, as the ResPool handle is meant to be
	// long-lived and shared between many goroutines.
	Close() error
	// Stats returns resource statistics.
	Stats() ResPoolStats
}

// Resource is a resource that can be stored in the ResPool.
type Resource interface {
	// SetAvatar stores the contact with resPool
	// Do not call it yourself, it is only called by (*ResPool).get, and will only be called once
	SetAvatar(*Avatar)
	// GetAvatar gets the contact with resPool
	// Do not call it yourself, it is only called by (*ResPool).Put
	GetAvatar() *Avatar
	// Close closes the original source
	// No need to call it yourself, it is only called by (*Avatar).close
	Close() error
}

// This is the size of the resourceOpener request chan (resPool.openerCh).
// This value should be larger than the maximum typical value
// used for resPool.maxOpen. If maxOpen is significantly larger than
// avatarRequestQueueSize then it is possible for ALL calls into the *ResPool
// to block until the resourceOpener can satisfy the backlog of requests.
const avatarRequestQueueSize = 1000000

// NewResPool creates ResPool.
func NewResPool(name string, newfunc func(context.Context) (Resource, error)) ResPool {
	p := &resPool{
		newfunc:        newfunc,
		name:           name,
		openerCh:       make(chan struct{}, avatarRequestQueueSize),
		lastPut:        make(map[*Avatar]string),
		avatarRequests: make(map[uint64]chan avatarRequest),
	}
	go p.resourceOpener()
	return p
}

type resPool struct {
	// newfunc may return a cached resource (one previously
	// closed), but doing so is unnecessary; the resPool package
	// maintains a resPool of idle resources for efficient re-use.
	//
	// The returned resource is only used by one goroutine at a
	// time.
	newfunc func(context.Context) (Resource, error)
	name    string
	// numClosed is an atomic counter which represents a total number of
	// closed resources.
	numClosed uint64

	mu             sync.Mutex // protects following fields
	freeAvatar     []*Avatar
	avatarRequests map[uint64]chan avatarRequest
	nextRequest    uint64 // Next key to use in avatarRequests.
	numOpen        int    // number of opened and pending open resources
	// Used to signal the need for new resources
	// a goroutine running resourceOpener() reads on this chan and
	// maybeOpenNewResources sends on the chan (one send per needed resource)
	// It is closed during p.Close(). The close tells the resourceOpener
	// goroutine to exit.
	openerCh    chan struct{}
	closed      bool
	dep         map[finalCloser]depSet
	lastPut     map[*Avatar]string // stacktrace of last getone's put; debug only
	maxIdle     int                // zero means defaultMaxIdle; negative means 0
	maxOpen     int                // <= 0 means unlimited
	maxLifetime time.Duration      // maximum amount of time a resource may be reused
	cleanerCh   chan struct{}
}

var _ ResPool = new(resPool)

// resourceReuseStrategy determines how (*resPool).getone returns resources.
type resourceReuseStrategy uint8

const (
	// alwaysNew forces a new avatar.
	alwaysNew resourceReuseStrategy = iota
	// cachedOrNew returns a cached resource, if available, else waits
	// for one to become available (if MaxOpen has been reached) or
	// creates a new avatar.
	cachedOrNew
)

// Avatar links a Resource with a mutex, to
// be held during all calls into the Avatar.
type Avatar struct {
	p         *resPool
	createdAt time.Time

	lock        sync.Mutex // guards following
	res         Resource
	closed      bool
	finalClosed bool // Avatar.Close has been called

	// guarded by resPool.mu
	inUse bool
}

// ResPool returns ResPool to which it belongs.
func (avatar *Avatar) ResPool() ResPool {
	return avatar.p
}

// Free releases self to the ResPool.
// If error is not nil, close it.
func (avatar *Avatar) Free(err error) {
	avatar.p.putAvatar(avatar, err)
}

func (avatar *Avatar) expired(timeout time.Duration) bool {
	if timeout <= 0 {
		return false
	}
	return avatar.createdAt.Add(timeout).Before(coarsetime.FloorTimeNow())
}

// the avatar.p's Mutex is held.
func (avatar *Avatar) closeResPoolLocked() func() error {
	avatar.lock.Lock()
	defer avatar.lock.Unlock()
	if avatar.closed {
		return func() error { return errors.New("resPool: duplicate *Avatar close") }
	}
	avatar.closed = true
	return avatar.p.removeDepLocked(avatar, avatar)
}

func (avatar *Avatar) close() error {
	avatar.lock.Lock()
	if avatar.closed {
		avatar.lock.Unlock()
		return errors.New("resPool: duplicate *Avatar close")
	}
	avatar.closed = true
	avatar.lock.Unlock() // not defer; removeDep finalClose calls may need to lock

	// And now updates that require holding avatar.mu.Lock.
	avatar.p.mu.Lock()
	fn := avatar.p.removeDepLocked(avatar, avatar)
	avatar.p.mu.Unlock()
	return fn()
}

func (avatar *Avatar) finalClose() error {
	var err error
	withLock(&avatar.lock, func() {
		avatar.finalClosed = true
		err = avatar.res.Close()
		avatar.res = nil
	})

	avatar.p.mu.Lock()
	avatar.p.numOpen--
	avatar.p.maybeOpenNewResources()
	avatar.p.mu.Unlock()

	atomic.AddUint64(&avatar.p.numClosed, 1)
	return err
}

// depSet is a finalCloser's outstanding dependencies
type depSet map[interface{}]bool // set of true bools

// The finalCloser interface is used by (*ResPool).addDep and related
// dependency reference counting.
type finalCloser interface {
	// finalClose is called when the reference count of an resource
	// goes to zero. (*ResPool).mu is not held while calling it.
	finalClose() error
}

func (p *resPool) addDepLocked(x finalCloser, dep interface{}) {
	if p.dep == nil {
		p.dep = make(map[finalCloser]depSet)
	}
	xdep := p.dep[x]
	if xdep == nil {
		xdep = make(depSet)
		p.dep[x] = xdep
	}
	xdep[dep] = true
}

func (p *resPool) removeDepLocked(x finalCloser, dep interface{}) func() error {
	//println(fmt.Sprintf("removeDep(%T %p, %T %p)", x, x, dep, dep))

	xdep, ok := p.dep[x]
	if !ok {
		panic(fmt.Sprintf("unpaired removeDep: no deps for %T", x))
	}

	l0 := len(xdep)
	delete(xdep, dep)

	switch len(xdep) {
	case l0:
		// Nothing removed. Shouldn't happen.
		panic(fmt.Sprintf("unpaired removeDep: no %T dep on %T", dep, x))
	case 0:
		// No more dependencies.
		delete(p.dep, x)
		return x.finalClose
	default:
		// Dependencies remain.
		return func() error { return nil }
	}
}

// Name returns the connResPool's name.
func (p *resPool) Name() string {
	return p.name
}

// Close closes the ResPool, releasing any open resources.
//
// It is rare to close a ResPool, as the ResPool handle is meant to be
// long-lived and shared between many goroutines.
func (p *resPool) Close() error {
	p.mu.Lock()
	if p.closed { // Make ResPool.Close idempotent
		p.mu.Unlock()
		return nil
	}
	close(p.openerCh)
	if p.cleanerCh != nil {
		close(p.cleanerCh)
	}
	var err error
	fns := make([]func() error, 0, len(p.freeAvatar))
	for _, avatar := range p.freeAvatar {
		fns = append(fns, avatar.closeResPoolLocked())
	}
	p.freeAvatar = nil
	p.closed = true
	for _, req := range p.avatarRequests {
		close(req)
	}
	p.mu.Unlock()
	for _, fn := range fns {
		err1 := fn()
		if err1 != nil {
			err = err1
		}
	}
	return err
}

const defaultMaxIdle = 2

func (p *resPool) maxIdleLocked() int {
	n := p.maxIdle
	switch {
	case n == 0:
		// TODO(bradfitz): ask newfunc, if supported, for its default preference
		return defaultMaxIdle
	case n < 0:
		return 0
	default:
		return n
	}
}

// SetMaxIdle sets the maximum number of resources in the idle
// resource pool.
//
// If SetMaxIdle is greater than 0 but less than the new MaxIdle
// then the new MaxIdle will be reduced to match the SetMaxIdle limit
//
// If n <= 0, no idle resources are retained.
func (p *resPool) SetMaxIdle(n int) {
	p.mu.Lock()
	if n > 0 {
		p.maxIdle = n
	} else {
		// No idle resources.
		p.maxIdle = -1
	}
	// Make sure maxIdle doesn't exceed maxOpen
	if p.maxOpen > 0 && p.maxIdleLocked() > p.maxOpen {
		p.maxIdle = p.maxOpen
	}
	var closing []*Avatar
	idleCount := len(p.freeAvatar)
	maxIdle := p.maxIdleLocked()
	if idleCount > maxIdle {
		closing = p.freeAvatar[maxIdle:]
		p.freeAvatar = p.freeAvatar[:maxIdle]
	}
	p.mu.Unlock()
	for _, c := range closing {
		c.close()
	}
}

// SetMaxOpen sets the maximum number of open resources.
//
// If MaxIdle is greater than 0 and the new MaxOpen is less than
// MaxIdle, then MaxIdle will be reduced to match the new
// MaxOpen limit
//
// If n <= 0, then there is no limit on the number of open resources.
// The default is 0 (unlimited).
func (p *resPool) SetMaxOpen(n int) {
	p.mu.Lock()
	p.maxOpen = n
	if n < 0 {
		p.maxOpen = 0
	}
	syncMaxIdle := p.maxOpen > 0 && p.maxIdleLocked() > p.maxOpen
	p.mu.Unlock()
	if syncMaxIdle {
		p.SetMaxIdle(n)
	}
}

// SetMaxLifetime sets the maximum amount of time a resource may be reused.
//
// Expired resource may be closed lazily before reuse.
//
// If d <= 0, resource are reused forever.
func (p *resPool) SetMaxLifetime(d time.Duration) {
	if d < 0 {
		d = 0
	}
	p.mu.Lock()
	// wake cleaner up when lifetime is shortened.
	if d > 0 && d < p.maxLifetime && p.cleanerCh != nil {
		select {
		case p.cleanerCh <- struct{}{}:
		default:
		}
	}
	p.maxLifetime = d
	p.startCleanerLocked()
	p.mu.Unlock()
}

// startCleanerLocked starts resourceCleaner if needed.
func (p *resPool) startCleanerLocked() {
	if p.maxLifetime > 0 && p.numOpen > 0 && p.cleanerCh == nil {
		p.cleanerCh = make(chan struct{}, 1)
		go p.resourceCleaner(p.maxLifetime)
	}
}

func (p *resPool) resourceCleaner(d time.Duration) {
	const minInterval = time.Second

	if d < minInterval {
		d = minInterval
	}
	t := time.NewTimer(d)

	for {
		select {
		case <-t.C:
		case <-p.cleanerCh: // maxLifetime was changed or connResPool was closed.
		}

		p.mu.Lock()
		d = p.maxLifetime
		if p.closed || p.numOpen == 0 || d <= 0 {
			p.cleanerCh = nil
			p.mu.Unlock()
			return
		}

		expiredSince := coarsetime.FloorTimeNow().Add(-d)
		var closing []*Avatar
		for i := 0; i < len(p.freeAvatar); i++ {
			c := p.freeAvatar[i]
			if c.createdAt.Before(expiredSince) {
				closing = append(closing, c)
				last := len(p.freeAvatar) - 1
				p.freeAvatar[i] = p.freeAvatar[last]
				p.freeAvatar[last] = nil
				p.freeAvatar = p.freeAvatar[:last]
				i--
			}
		}
		p.mu.Unlock()

		for _, c := range closing {
			c.close()
		}

		if d < minInterval {
			d = minInterval
		}
		t.Reset(d)
	}
}

// ResPoolStats contains resource statistics.
type ResPoolStats struct {
	// OpenResources is the number of open resources to the resource.
	OpenResources   int
	FreeResources   int
	ClosedResources uint64
}

// Stats returns resource statistics.
func (p *resPool) Stats() ResPoolStats {
	p.mu.Lock()
	stats := ResPoolStats{
		OpenResources:   p.numOpen,
		ClosedResources: p.numClosed,
		FreeResources:   len(p.freeAvatar),
	}
	p.mu.Unlock()
	return stats
}

// Assumes p.mu is locked.
// If there are avatarRequests and the resource limit hasn't been reached,
// then tell the resourceOpener to open new resources.
func (p *resPool) maybeOpenNewResources() {
	numRequests := len(p.avatarRequests)
	if p.maxOpen > 0 {
		numCanOpen := p.maxOpen - p.numOpen
		if numRequests > numCanOpen {
			numRequests = numCanOpen
		}
	}
	for numRequests > 0 {
		p.numOpen++ // optimistically
		numRequests--
		if p.closed {
			return
		}
		p.openerCh <- struct{}{}
	}
}

// Runs in a separate goroutine, opens new resources when requested.
func (p *resPool) resourceOpener() {
	ctx := context.TODO()
	for range p.openerCh {
		p.openNewResource(ctx)
	}
}

// Open one new resource
func (p *resPool) openNewResource(ctx context.Context) {
	// maybeOpenNewConnctions has already executed p.numOpen++ before it sent
	// on p.openerCh. This function must execute p.numOpen-- if the
	// resource fails or is closed before returning.
	res, err := p.newfunc(ctx)
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		if err == nil {
			res.Close()
		}
		p.numOpen--
		return
	}
	if err != nil {
		p.numOpen--
		// p.putResPoolLocked(nil, err)
		p.maybeOpenNewResources()
		return
	}
	avatar := &Avatar{
		p:         p,
		createdAt: coarsetime.FloorTimeNow(),
		res:       res,
	}
	res.SetAvatar(avatar)
	if p.putResPoolLocked(avatar, err) {
		p.addDepLocked(avatar, avatar)
	} else {
		p.numOpen--
		res.Close()
	}
}

// avatarRequest represents one request for a new resource
// When there are no idle resources available, ResPool.getone will create
// a new avatarRequest and put it on the p.avatarRequests list.
type avatarRequest struct {
	avatar *Avatar
	err    error
}

var errResPoolClosed = errors.New("resPool: resource is closed")

// nextRequestKeyLocked returns the next resource request key.
// It is assumed that nextRequest will not overflow.
func (p *resPool) nextRequestKeyLocked() uint64 {
	next := p.nextRequest
	p.nextRequest++
	return next
}

// maxBadGetoneRetries is the number of maximum retries if the newfunc returns
// ErrExpired to signal a broken resource before forcing a new
// resource to be opened.
const maxBadGetoneRetries = 2

// GetContext returns a object in Resource type, support context cancellation.
func (p *resPool) GetContext(ctx context.Context) (Resource, error) {
	var err error
	var res Resource
	for i := 0; i < maxBadGetoneRetries; i++ {
		res, err = p.getone(ctx, cachedOrNew)
		if err == nil {
			break
		}
	}
	if err != nil {
		return p.getone(ctx, alwaysNew)
	}
	return res, err
}

// Get returns a object in Resource type.
func (p *resPool) Get() (Resource, error) {
	return p.GetContext(context.Background())
}

// Callback callbacks your handle function, returns the error of getting resource or handling.
// Support recover panic.
func (p *resPool) Callback(fn func(Resource) error) error {
	return p.CallbackContext(context.Background(), fn)
}

// Callback callbacks your handle function, returns the error of getting resource or handling.
// Support recover panic and context cancellation.
func (p *resPool) CallbackContext(ctx context.Context, fn func(Resource) error) (err error) {
	res, err := p.GetContext(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
		p.Put(res, err)
	}()
	err = fn(res)
	return err
}

// ErrExpired error: getting expired resource.
var ErrExpired = errors.New("resPool: getting expired resource")

// getone returns a newly-opened or cached *Avatar.
func (p *resPool) getone(ctx context.Context, strategy resourceReuseStrategy) (Resource, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, errResPoolClosed
	}
	// Check if the context is expired.
	select {
	default:
	case <-ctx.Done():
		p.mu.Unlock()
		return nil, ctx.Err()
	}
	lifetime := p.maxLifetime

	// Prefer a free resource, if possible.
	numFree := len(p.freeAvatar)
	if strategy == cachedOrNew && numFree > 0 {
		a := p.freeAvatar[0]
		copy(p.freeAvatar, p.freeAvatar[1:])
		p.freeAvatar = p.freeAvatar[:numFree-1]
		a.inUse = true
		p.mu.Unlock()
		if a.expired(lifetime) {
			a.close()
			return nil, ErrExpired
		}
		return a.res, nil
	}
	// Out of free resources or we were asked not to use one. If we're not
	// allowed to open any more resources, make a request and wait.
	if p.maxOpen > 0 && p.numOpen >= p.maxOpen {
		// Make the avatarRequest channel. It's buffered so that the
		// resourceOpener doesn't block while waiting for the req to be read.
		req := make(chan avatarRequest, 1)
		reqKey := p.nextRequestKeyLocked()
		p.avatarRequests[reqKey] = req
		p.mu.Unlock()

		// Timeout the resource request with the context.
		select {
		case <-ctx.Done():
			// Remove the resource request and ensure no value has been sent
			// on it after removing.
			p.mu.Lock()
			delete(p.avatarRequests, reqKey)
			p.mu.Unlock()
			select {
			default:
			case ret, ok := <-req:
				if ok {
					p.putAvatar(ret.avatar, ret.err)
				}
			}
			return nil, ctx.Err()
		case ret, ok := <-req:
			if !ok {
				return nil, errResPoolClosed
			}
			if ret.err == nil {
				if ret.avatar.expired(lifetime) {
					ret.avatar.close()
					return nil, ErrExpired
				}
				return ret.avatar.res, nil
			}
			return nil, ret.err
		}
	}

	p.numOpen++ // optimistically
	p.mu.Unlock()
	res, err := p.newfunc(ctx)
	if err != nil {
		p.mu.Lock()
		p.numOpen-- // correct for earlier optimism
		p.maybeOpenNewResources()
		p.mu.Unlock()
		return nil, err
	}
	p.mu.Lock()
	avatar := &Avatar{
		p:         p,
		createdAt: coarsetime.FloorTimeNow(),
		res:       res,
	}
	res.SetAvatar(avatar)
	p.addDepLocked(avatar, avatar)
	avatar.inUse = true
	p.mu.Unlock()
	return avatar.res, nil
}

// Put gives a resource back to the ResPool.
// If error is not nil, close the avatar.
func (p *resPool) Put(res Resource, err error) {
	a := res.GetAvatar()
	if a == nil {
		res.Close()
		return
	}
	p.putAvatar(a, err)
}

// putAvatarHook is a hook for testing.
var putAvatarHook func(*resPool, *Avatar)

// debugGetPut determines whether getConn & putAvatar calls' stack traces
// are returned for more verbose crashes.
const debugGetPut = false

// putAvatar adds a resource to the ResPool's free resPool.
// err is optionally the last error that occurred on this avatar.
func (p *resPool) putAvatar(avatar *Avatar, err error) {
	p.mu.Lock()
	if err != nil {
		if avatar.inUse {
			// Don't reuse bad resources.
			// Since the conn is considered bad and is being discarded, treat it
			// as closed. Don't decrement the open count here, finalClose will
			// take care of that.
			p.maybeOpenNewResources()
		}
		p.mu.Unlock()
		avatar.close()
		return
	}
	if !avatar.inUse {
		p.mu.Unlock()
		return
	}
	if debugGetPut {
		p.lastPut[avatar] = stack()
	}
	avatar.inUse = false

	if putAvatarHook != nil {
		putAvatarHook(p, avatar)
	}
	added := p.putResPoolLocked(avatar, nil)
	p.mu.Unlock()

	if !added {
		avatar.close()
	}
}

// Satisfy a avatarRequest or put the Avatar in the idle resPool and return true
// or return false.
// putResPoolLocked will satisfy a avatarRequest if there is one, or it will
// return the *Avatar to the freeAvatar list if err == nil and the idle
// resource limit will not be exceeded.
// If err != nil, the value of avatar is ignored.
// If err == nil, then avatar must not equal nil.
// If a avatarRequest was fulfilled or the *Avatar was placed in the
// freeAvatar list, then true is returned, otherwise false is returned.
func (p *resPool) putResPoolLocked(avatar *Avatar, err error) bool {
	if p.closed {
		return false
	}
	if p.maxOpen > 0 && p.numOpen > p.maxOpen {
		return false
	}
	if c := len(p.avatarRequests); c > 0 {
		var req chan avatarRequest
		var reqKey uint64
		for reqKey, req = range p.avatarRequests {
			break
		}
		delete(p.avatarRequests, reqKey) // Remove from pending requests.
		if err == nil {
			avatar.inUse = true
		}
		req <- avatarRequest{
			avatar: avatar,
			err:    err,
		}
		return true
	} else if err == nil && !p.closed && p.maxIdleLocked() > len(p.freeAvatar) {
		p.freeAvatar = append(p.freeAvatar, avatar)
		p.startCleanerLocked()
		return true
	}
	return false
}

func stack() string {
	var buf [2 << 10]byte
	return string(buf[:runtime.Stack(buf[:], false)])
}

// withLock runs while holding lk.
func withLock(lk sync.Locker, fn func()) {
	lk.Lock()
	defer lk.Unlock() // in case fn panics
	fn()
}

// ResPools stores ResPool
type ResPools struct {
	// stores 'map[string]ResPool',
	// one server node has one connection pool.
	resPools atomic.Value
	// protects resPools
	mutex sync.Mutex
}

// NewResPools creates a new ResPools.
func NewResPools() *ResPools {
	c := &ResPools{}
	c.resPools.Store(make(map[string]ResPool))
	return c
}

// Get gets ResPool by name.
func (c *ResPools) Get(name string) (ResPool, bool) {
	pool, ok := c.resPools.Load().(map[string]ResPool)[name]
	return pool, ok
}

// GetAll gets all the ResPools.
func (c *ResPools) GetAll() []ResPool {
	all := c.resPools.Load().(map[string]ResPool)
	resPools := make(resPools, 0, len(all))
	for _, pool := range all {
		resPools = append(resPools, pool)
	}
	sort.Sort(resPools)
	return resPools
}

// Set stores ResPool.
// If the same name exists, will close and cover it.
func (c *ResPools) Set(pool ResPool) {
	c.mutex.Lock()
	resPools := c.resPools.Load().(map[string]ResPool)
	m := make(map[string]ResPool, len(resPools)+1)
	name := pool.Name()
	for k, v := range resPools {
		if k == name {
			v.Close()
		} else {
			m[k] = v
		}
	}
	m[name] = pool
	c.resPools.Store(m)
	c.mutex.Unlock()
}

// Del delects ResPool by name, and close the ResPool.
func (c *ResPools) Del(name string) {
	c.mutex.Lock()
	resPools := c.resPools.Load().(map[string]ResPool)
	m := make(map[string]ResPool, len(resPools))
	for k, v := range resPools {
		if k == name {
			v.Close()
		} else {
			m[k] = v
		}
	}
	c.resPools.Store(m)
	c.mutex.Unlock()
}

// Clean delects and close all the ResPools.
func (c *ResPools) Clean() {
	c.mutex.Lock()
	resPools := c.resPools.Load().(map[string]ResPool)
	for _, v := range resPools {
		v.Close()
	}
	c.resPools.Store(make(map[string]ResPool))
	c.mutex.Unlock()
}

type resPools []ResPool

func (p resPools) Len() int {
	return len(p)
}

func (p resPools) Less(i, j int) bool {
	return p[i].Name() < p[j].Name()
}

func (p resPools) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
