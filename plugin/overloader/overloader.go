// Package overloader is a plugin to protect erpc from overload.
package overloader

import (
	"fmt"
	"sync"
	"time"

	"github.com/henrylee2cn/erpc/v6"
)

type (
	// Overloader plug-in to protect erpc from overload
	Overloader struct {
		limitConfig           *LimitConfig
		limitConfigLock       sync.RWMutex
		connLimiter           *connLimiter
		connLimiterLock       sync.RWMutex
		totalQPSLimiter       *qpsLimiter
		totalQPSLimiterLock   sync.RWMutex
		handlerQPSLimiter     map[string]*qpsLimiter
		handlerQPSLimiterLock sync.RWMutex
	}
	// LimitConfig overload limitation condition
	LimitConfig struct {
		MaxConn       int32
		QPSInterval   time.Duration
		MaxTotalQPS   int32
		MaxHandlerQPS []HandlerLimit
	}
	// HandlerLimit handler QPS overload limitation condition
	HandlerLimit struct {
		ServiceMethod string
		MaxQPS        int32
	}
)

var (
	_ erpc.PostDialPlugin           = (*Overloader)(nil)
	_ erpc.PostAcceptPlugin         = (*Overloader)(nil)
	_ erpc.PostDisconnectPlugin     = (*Overloader)(nil)
	_ erpc.PostReadCallHeaderPlugin = (*Overloader)(nil)
	_ erpc.PostReadPushHeaderPlugin = (*Overloader)(nil)
)

// New creates a plug-in to protect erpc from overload.
func New(initLimitConfig LimitConfig) *Overloader {
	o := &Overloader{
		handlerQPSLimiter: make(map[string]*qpsLimiter),
	}
	o.Update(initLimitConfig)
	return o
}

// Name returns the plugin name.
func (o *Overloader) Name() string {
	return "overloader"
}

// PostDial checks connection overload.
// If overload, print error log and close the connection.
func (o *Overloader) PostDial(sess erpc.PreSession, isRedial bool) *erpc.Status {
	if isRedial {
		return nil
	}
	return o.PostAccept(sess)
}

// PostAccept checks connection overload.
// If overload, print error log and close the connection.
func (o *Overloader) PostAccept(_ erpc.PreSession) *erpc.Status {
	if o.takeConn() {
		return nil
	}
	msg := fmt.Sprintf("connection overload, limit=%d, now=%d",
		o.connLimiter.getLimit(), o.connLimiter.getNow(),
	)
	return erpc.NewStatus(erpc.CodeInternalServerError, msg, nil)
}

// PostDisconnect releases connection count.
func (o *Overloader) PostDisconnect(_ erpc.BaseSession) *erpc.Status {
	o.releaseConn()
	return nil
}

// PostReadCallHeader checks PULL QPS overload.
// If overload, print error log and reply error.
func (o *Overloader) PostReadCallHeader(ctx erpc.ReadCtx) *erpc.Status {
	if !o.takeTotalQPS() {
		msg := fmt.Sprintf("qps overload, total_limit=%d",
			o.totalQPSLimiter.getLimit(),
		)
		return erpc.NewStatus(erpc.CodeInternalServerError, msg, nil)
	}
	limit, ok := o.takeHandlerQPS(ctx.ServiceMethod())
	if ok {
		return nil
	}
	msg := fmt.Sprintf("qps overload, handler_limit=%d",
		limit,
	)
	return erpc.NewStatus(erpc.CodeInternalServerError, msg, nil)
}

// PostReadPushHeader checks PUSH QPS overload.
// If overload, print warning log.
func (o *Overloader) PostReadPushHeader(ctx erpc.ReadCtx) *erpc.Status {
	return o.PostReadCallHeader(ctx)
}

// LimitConfig returns the overload limitation condition.
func (o *Overloader) LimitConfig() LimitConfig {
	return *o.limitConfig
}

// Update updates the overload limitation condition.
func (o *Overloader) Update(newLimitConfig LimitConfig) {
	limitConfig := &newLimitConfig
	o.updateConnLimiter(limitConfig)
	o.updateTotalQPSLimiter(limitConfig)
	o.updateHandlerLimiter(limitConfig)
	o.limitConfigLock.Lock()
	o.limitConfig = limitConfig
	o.limitConfigLock.Unlock()
}

func (o *Overloader) updateConnLimiter(limitConfig *LimitConfig) {
	o.limitConfigLock.Lock()
	if limitConfig.MaxConn <= 0 {
		o.connLimiter = nil
		o.limitConfigLock.Unlock()
		return
	}
	if o.connLimiter == nil {
		o.connLimiter = newConnLimiter(limitConfig.MaxConn)
	} else if o.limitConfig.MaxConn != limitConfig.MaxConn {
		o.connLimiter.update(limitConfig.MaxConn)
	}
	o.limitConfigLock.Unlock()
}

func (o *Overloader) updateTotalQPSLimiter(limitConfig *LimitConfig) {
	o.totalQPSLimiterLock.Lock()
	if limitConfig.MaxTotalQPS <= 0 {
		o.totalQPSLimiter = nil
		o.totalQPSLimiterLock.Unlock()
		return
	}
	if o.totalQPSLimiter == nil {
		o.totalQPSLimiter = newQPSLimiter(limitConfig.MaxTotalQPS, limitConfig.QPSInterval)
	} else if o.limitConfig.MaxTotalQPS != limitConfig.MaxTotalQPS ||
		o.limitConfig.QPSInterval != limitConfig.QPSInterval {
		o.totalQPSLimiter.update(limitConfig.MaxTotalQPS, limitConfig.QPSInterval)
	}
	o.totalQPSLimiterLock.Unlock()
}

func (o *Overloader) updateHandlerLimiter(limitConfig *LimitConfig) {
	o.handlerQPSLimiterLock.Lock()
	for _, v := range limitConfig.MaxHandlerQPS {
		if v.MaxQPS <= 0 {
			delete(o.handlerQPSLimiter, v.ServiceMethod)
		}
		l, ok := o.handlerQPSLimiter[v.ServiceMethod]
		if !ok {
			l = newQPSLimiter(v.MaxQPS, limitConfig.QPSInterval)
			o.handlerQPSLimiter[v.ServiceMethod] = l
		} else {
			l.update(v.MaxQPS, limitConfig.QPSInterval)
		}
	}
	for k := range o.handlerQPSLimiter {
		var has bool
		for _, v := range limitConfig.MaxHandlerQPS {
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
