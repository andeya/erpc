package pool

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/henrylee2cn/goutil/coarsetime"
)

/**
 * Non-blocking asynchronous multiplex resource pool.
 *
 * Conditions of Use:
 * - Limited resources
 * - Resources can be multiplexed non-blockingly and asynchronously
 * - Typical application scenarios, such as connection pool for asynchronous communication
 *
 * Performance:
 * - The longer the business is, the more obvious the performance improvement.
 * If the service is executed for 1ms each time, the performance is improved by about 4 times;
 * If the business is executed for 10ms each time, the performance is improved by about 28 times
 * - The average time spent on each operation will not change significantly,
 * but the overall throughput is greatly improved
 */

type (
	// Worker woker interface
	// Note: Worker can not be implemented using empty structures(struct{})!
	Worker interface {
		Health() bool
		Close() error
	}
	// Workshop working workshop
	Workshop struct {
		addFn           func() (*workerInfo, error)
		maxQuota        int
		maxIdleDuration time.Duration
		infos           map[Worker]*workerInfo
		minLoadInfo     *workerInfo
		stats           *WorkshopStats
		statsReader     atomic.Value
		lock            sync.Mutex
		wg              sync.WaitGroup
		closeCh         chan struct{}
		closeLock       sync.Mutex
	}
	workerInfo struct {
		worker     Worker
		jobNum     int32
		idleExpire time.Time
	}
	// WorkshopStats workshop stats
	WorkshopStats struct {
		Worker  int32  // The current total number of workers
		Idle    int32  // The current total number of idle workers
		Created uint64 // Total created workers
		Doing   int32  // The total number of tasks in progress
		Done    uint64 // The total number of tasks completed
		MaxLoad int32  // In the current load balancing, the maximum number of tasks
		MinLoad int32  // In the current load balancing, the minimum number of tasks
	}
)

const (
	defaultWorkerMaxQuota        = 64
	defaultWorkerMaxIdleDuration = 3 * time.Minute
)

var (
	// ErrWorkshopClosed error: 'workshop is closed'
	ErrWorkshopClosed = fmt.Errorf("%s", "workshop is closed")
)

// NewWorkshop creates a new workshop(non-blocking asynchronous multiplex resource pool).
// If maxQuota<=0, will use default value.
// If maxIdleDuration<=0, will use default value.
// Note: Worker can not be implemented using empty structures(struct{})!
func NewWorkshop(maxQuota int, maxIdleDuration time.Duration, newWorkerFunc func() (Worker, error)) *Workshop {
	if maxQuota <= 0 {
		maxQuota = defaultWorkerMaxQuota
	}
	if maxIdleDuration <= 0 {
		maxIdleDuration = defaultWorkerMaxIdleDuration
	}
	w := new(Workshop)
	w.stats = new(WorkshopStats)
	w.reportStatsLocked()
	w.maxQuota = maxQuota
	w.maxIdleDuration = maxIdleDuration
	w.infos = make(map[Worker]*workerInfo, maxQuota)
	w.closeCh = make(chan struct{})
	w.addFn = func() (info *workerInfo, err error) {
		defer func() {
			if p := recover(); p != nil {
				err = fmt.Errorf("%v", p)
			}
		}()
		worker, err := newWorkerFunc()
		if err != nil {
			return nil, err
		}
		info = &workerInfo{
			worker: worker,
		}
		w.infos[worker] = info
		w.stats.Created++
		w.stats.Worker++
		return info, nil
	}
	go w.gc()
	return w
}

// Callback assigns a healthy worker to execute the function.
func (w *Workshop) Callback(fn func(Worker) error) (err error) {
	select {
	case <-w.closeCh:
		return ErrWorkshopClosed
	default:
	}
	w.lock.Lock()
	info, err := w.hireLocked()
	w.lock.Unlock()
	if err != nil {
		return err
	}
	worker := info.worker
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("%v", p)
		}
		w.lock.Lock()
		_, ok := w.infos[worker]
		if !ok {
			worker.Close()
		} else {
			w.fireLocked(info)
		}
		w.lock.Unlock()
	}()
	return fn(worker)
}

// Close wait for all the work to be completed and close the workshop.
func (w *Workshop) Close() {
	w.closeLock.Lock()
	defer w.closeLock.Unlock()
	select {
	case <-w.closeCh:
		return
	default:
		close(w.closeCh)
	}

	w.wg.Wait()
	w.lock.Lock()
	defer w.lock.Unlock()
	for _, info := range w.infos {
		info.worker.Close()
	}
	w.infos = nil
	w.stats.Idle = 0
	w.stats.Worker = 0
	w.refreshLocked(true)
}

// Fire marks the worker to reduce a job.
// If the worker does not belong to the workshop, close the worker.
func (w *Workshop) Fire(worker Worker) {
	w.lock.Lock()
	info, ok := w.infos[worker]
	if !ok {
		if worker != nil {
			worker.Close()
		}
		w.lock.Unlock()
		return
	}
	w.fireLocked(info)
	w.lock.Unlock()
}

// Hire hires a healthy worker and marks the worker to add a job.
func (w *Workshop) Hire() (Worker, error) {
	select {
	case <-w.closeCh:
		return nil, ErrWorkshopClosed
	default:
	}
	w.lock.Lock()
	info, err := w.hireLocked()
	if err != nil {
		w.lock.Unlock()
		return nil, err
	}
	w.lock.Unlock()
	return info.worker, nil
}

// Stats returns the current workshop stats.
func (w *Workshop) Stats() WorkshopStats {
	return w.statsReader.Load().(WorkshopStats)
}

func (w *Workshop) fireLocked(info *workerInfo) {
	{
		w.stats.Doing--
		w.stats.Done++
		info.jobNum--
		w.wg.Add(-1)
	}
	jobNum := info.jobNum
	if jobNum == 0 {
		info.idleExpire = coarsetime.CeilingTimeNow().Add(w.maxIdleDuration)
		w.stats.Idle++
	}

	if jobNum+1 >= w.stats.MaxLoad {
		w.refreshLocked(true)
		return
	}

	if !w.checkInfoLocked(info) {
		if info == w.minLoadInfo {
			w.refreshLocked(true)
		}
		return
	}

	if jobNum < w.stats.MinLoad {
		w.stats.MinLoad = jobNum
		w.minLoadInfo = info
	}
	w.reportStatsLocked()
}

func (w *Workshop) hireLocked() (*workerInfo, error) {
	var info *workerInfo
GET:
	info = w.minLoadInfo
	if len(w.infos) >= w.maxQuota || (info != nil && info.jobNum == 0) {
		if !w.checkInfoLocked(info) {
			w.refreshLocked(false)
			goto GET
		}
		if info.jobNum == 0 {
			w.stats.Idle--
		}
		info.jobNum++
		w.setMinLoadInfoLocked()
		w.stats.MinLoad = w.minLoadInfo.jobNum
		if w.stats.MaxLoad < info.jobNum {
			w.stats.MaxLoad = info.jobNum
		}

	} else {
		var err error
		info, err = w.addFn()
		if err != nil {
			return nil, err
		}
		info.jobNum = 1
		w.stats.MinLoad = 1
		if w.stats.MaxLoad == 0 {
			w.stats.MaxLoad = 1
		}
		w.minLoadInfo = info
	}

	w.wg.Add(1)
	w.stats.Doing++

	w.reportStatsLocked()

	return info, nil
}

func (w *Workshop) gc() {
	for {
		select {
		case <-w.closeCh:
			return
		default:
			time.Sleep(w.maxIdleDuration)
			w.lock.Lock()
			w.refreshLocked(true)
			w.lock.Unlock()
		}
	}
}

func (w *Workshop) setMinLoadInfoLocked() {
	if len(w.infos) == 0 {
		w.minLoadInfo = nil
		return
	}
	var minLoadInfo *workerInfo
	for _, info := range w.infos {
		if minLoadInfo != nil && info.jobNum >= minLoadInfo.jobNum {
			continue
		}
		minLoadInfo = info
	}
	w.minLoadInfo = minLoadInfo
}

// Remove the expired or unhealthy idle workers.
// The time complexity is O(n).
func (w *Workshop) refreshLocked(reportStats bool) {
	var max, min, tmp int32
	min = math.MaxInt32
	var minLoadInfo *workerInfo
	for _, info := range w.infos {
		if !w.checkInfoLocked(info) {
			continue
		}
		tmp = info.jobNum
		if tmp > max {
			max = tmp
		}
		if tmp < min {
			min = tmp
		}
		if minLoadInfo != nil && tmp >= minLoadInfo.jobNum {
			continue
		}
		minLoadInfo = info
	}
	if min == math.MaxInt32 {
		min = 0
	}
	w.stats.MinLoad = min
	w.stats.MaxLoad = max
	if reportStats {
		w.reportStatsLocked()
	}
	w.minLoadInfo = minLoadInfo
}

func (w *Workshop) checkInfoLocked(info *workerInfo) bool {
	if !info.worker.Health() ||
		(info.jobNum == 0 && coarsetime.FloorTimeNow().After(info.idleExpire)) {
		delete(w.infos, info.worker)
		info.worker.Close()
		w.stats.Worker--
		if info.jobNum == 0 {
			w.stats.Idle--
		} else {
			w.wg.Add(-int(info.jobNum))
			w.stats.Doing -= info.jobNum
			w.stats.Done += uint64(info.jobNum)
		}
		return false
	}
	return true
}

func (w *Workshop) reportStatsLocked() {
	w.statsReader.Store(*w.stats)
}
