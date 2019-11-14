// Package tps statistics requests per second
package erpcs

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/henrylee2cn/erpc/v6"
)

//go:generate go build $GOFILE

func NewTPS(intervalSecond uint32) *TPS {
	if intervalSecond < 1 {
		intervalSecond = 1
	}
	return &TPS{
		stat:           map[string]*uint32{},
		intervalSecond: intervalSecond,
	}
}

type TPS struct {
	stat           map[string]*uint32
	intervalSecond uint32
	once           sync.Once
}

var (
	_ erpc.PostRegPlugin          = (*TPS)(nil)
	_ erpc.PostWriteReplyPlugin   = (*TPS)(nil)
	_ erpc.PostReadPushBodyPlugin = (*TPS)(nil)
)

func (t *TPS) start() {
	go func() {
		ticker := time.NewTicker(time.Second * time.Duration(t.intervalSecond))
		intervalSecond := float32(t.intervalSecond)
		for {
			<-ticker.C
			for k, v := range t.stat {
				log.Printf("[TPS] %s: %f in last %ds", k, float32(atomic.SwapUint32(v, 0))/intervalSecond, t.intervalSecond)
			}
		}
	}()
}

func (t *TPS) Name() string {
	return "TPS"
}

func (t *TPS) PostReg(h *erpc.Handler) error {
	t.stat[h.Name()] = new(uint32)
	return nil
}

func (t *TPS) PostWriteReply(ctx erpc.WriteCtx) *erpc.Status {
	t.once.Do(t.start)
	atomic.AddUint32(t.stat[ctx.Output().ServiceMethod()], 1)
	return nil
}

func (t *TPS) PostReadPushBody(ctx erpc.ReadCtx) *erpc.Status {
	t.once.Do(t.start)
	atomic.AddUint32(t.stat[ctx.ServiceMethod()], 1)
	return nil
}
