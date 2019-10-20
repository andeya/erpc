// Package overloader is a plugin to protect teleport from overload.
package overloader

import (
	"fmt"
	"sync"
	"time"

	tp "github.com/henrylee2cn/teleport"
)

type (
	// Overloader plug-in to protect teleport from overload
	Overloader struct {
		limitCond             *LimitCond
		limitCondLock         sync.RWMutex
		connLimiter           *connLimiter
		connLimiterLock       sync.RWMutex
		totalQPSLimiter       *qpsLimiter
		totalQPSLimiterLock   sync.RWMutex
		handlerQPSLimiter     map[string]*qpsLimiter
		handlerQPSLimiterLock sync.RWMutex
	}
	// LimitCond overload limitation condition
	LimitCond struct {
		MaxConn       int32
		QPSInterval   time.Duration
		MaxTotalQPS   int32
		MaxHandlerQPS []HandlerQPSLimit
	}
	// HandlerQPSLimit handler QPS overload limitation condition
	HandlerQPSLimit struct {
		ServiceMethod string
		MaxQPS        int32
	}
)

var (
	_ tp.PostAcceptPlugin         = (*Overloader)(nil)
	_ tp.PostReadCallHeaderPlugin = (*Overloader)(nil)
	_ tp.PostReadPushHeaderPlugin = (*Overloader)(nil)
)

// NewOverloader creates a plug-in to protect teleport from overload.
func NewOverloader(initLimitCond LimitCond) *Overloader {
	o := &Overloader{
		handlerQPSLimiter: make(map[string]*qpsLimiter),
	}
	o.UpdateLimitCond(initLimitCond)
	return o
}

// Name returns the plugin name.
func (o *Overloader) Name() string {
	return "overloader"
}

// PostAccept checks connection overload.
// If overload, print error log and close the connection.
func (o *Overloader) PostAccept(sess tp.PreSession) *tp.Status {
	if o.takeConn() {
		return nil
	}
	msg := fmt.Sprintf("connection overload, limit=%d, now=%d",
		o.connLimiter.getLimit(), o.connLimiter.getNow(),
	)
	return tp.NewStatus(tp.CodeInternalServerError, msg, nil)
}

// PostReadCallHeader checks PULL QPS overload.
// If overload, print error log and reply error.
func (o *Overloader) PostReadCallHeader(ctx tp.ReadCtx) *tp.Status {
	if !o.takeTotalQPS() {
		msg := fmt.Sprintf("qps overload, total_limit=%d",
			o.totalQPSLimiter.getLimit(),
		)
		return tp.NewStatus(tp.CodeInternalServerError, msg, nil)
	}
	limit, ok := o.takeHandlerQPS(ctx.ServiceMethod())
	if ok {
		return nil
	}
	msg := fmt.Sprintf("qps overload, handler_limit=%d",
		limit,
	)
	return tp.NewStatus(tp.CodeInternalServerError, msg, nil)
}

// PostReadPushHeader checks PUSH QPS overload.
// If overload, print warning log.
func (o *Overloader) PostReadPushHeader(ctx tp.ReadCtx) *tp.Status {
	return o.PostReadCallHeader(ctx)
}

// LimitCond returns the overload limitation condition.
func (o *Overloader) LimitCond() LimitCond {
	return *o.limitCond
}

// UpdateLimitCond updates the overload limitation condition.
func (o *Overloader) UpdateLimitCond(newLimitCond LimitCond) {
	limitCond := &newLimitCond
	o.updateConnLimiter(limitCond)
	o.updateTotalQPSLimiter(limitCond)
	o.updateHandlerQPSLimiter(limitCond)
	o.limitCondLock.Lock()
	o.limitCond = limitCond
	o.limitCondLock.Unlock()
}

func (o *Overloader) updateConnLimiter(limitCond *LimitCond) {
	o.limitCondLock.Lock()
	if limitCond.MaxConn <= 0 {
		o.connLimiter = nil
		o.limitCondLock.Unlock()
		return
	}
	if o.connLimiter == nil {
		o.connLimiter = newConnLimiter(limitCond.MaxConn)
	} else if o.limitCond.MaxConn != limitCond.MaxConn {
		o.connLimiter.update(limitCond.MaxConn)
	}
	o.limitCondLock.Unlock()
}

func (o *Overloader) updateTotalQPSLimiter(limitCond *LimitCond) {
	o.totalQPSLimiterLock.Lock()
	if limitCond.MaxTotalQPS <= 0 {
		o.totalQPSLimiter = nil
		o.totalQPSLimiterLock.Unlock()
		return
	}
	if o.totalQPSLimiter == nil {
		o.totalQPSLimiter = newQPSLimiter(limitCond.MaxTotalQPS, limitCond.QPSInterval)
	} else if o.limitCond.MaxTotalQPS != limitCond.MaxTotalQPS || o.limitCond.QPSInterval != limitCond.QPSInterval {
		o.totalQPSLimiter.update(limitCond.MaxTotalQPS, limitCond.QPSInterval)
	}
	o.totalQPSLimiterLock.Unlock()
}

func (o *Overloader) updateHandlerQPSLimiter(limitCond *LimitCond) {
	o.handlerQPSLimiterLock.Lock()
	for _, v := range limitCond.MaxHandlerQPS {
		if v.MaxQPS <= 0 {
			delete(o.handlerQPSLimiter, v.ServiceMethod)
		}
		l, ok := o.handlerQPSLimiter[v.ServiceMethod]
		if !ok {
			l = newQPSLimiter(v.MaxQPS, limitCond.QPSInterval)
			o.handlerQPSLimiter[v.ServiceMethod] = l
		} else {
			l.update(v.MaxQPS, limitCond.QPSInterval)
		}
	}
	for k := range o.handlerQPSLimiter {
		var has bool
		for _, v := range limitCond.MaxHandlerQPS {
			if v.ServiceMethod == k {
				has = true
				break
			}
		}
		if !has {
			delete(o.handlerQPSLimiter, k)
		}
	}
	o.handlerQPSLimiterLock.Unlock()
}

func (o *Overloader) takeConn() bool {
	o.connLimiterLock.RLock()
	bol := o.connLimiter == nil || o.connLimiter.take()
	o.connLimiterLock.RUnlock()
	return bol
}

func (o *Overloader) releaseConn() {
	o.connLimiterLock.RLock()
	if o.connLimiter != nil {
		o.connLimiter.release()
	}
	o.connLimiterLock.RUnlock()
}

func (o *Overloader) takeTotalQPS() bool {
	o.totalQPSLimiterLock.RLock()
	bol := o.totalQPSLimiter == nil || o.totalQPSLimiter.take()
	o.totalQPSLimiterLock.RUnlock()
	return bol
}

func (o *Overloader) takeHandlerQPS(serviceMethod string) (limit int32, ok bool) {
	o.handlerQPSLimiterLock.RLock()
	if l, exist := o.handlerQPSLimiter[serviceMethod]; exist {
		ok = l.take()
		if !ok {
			limit = l.getLimit()
		}
	} else {
		ok = true
	}
	o.handlerQPSLimiterLock.RUnlock()
	return limit, ok
}
