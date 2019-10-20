package overloader

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOverload_Conn(t *testing.T) {
	ol := NewOverloader(LimitCond{
		MaxConn:     50,
		MaxTotalQPS: 10000,
		QPSInterval: 100 * time.Millisecond,
	})
	for i := 0; i < 50; i++ {
		assert.True(t, ol.takeConn())
	}
	for i := 0; i < 10; i++ {
		assert.False(t, ol.takeConn())
	}
	for i := 0; i < 50; i++ {
		ol.releaseConn()
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10000; i++ {
			assert.True(t, ol.connLimiter.getNow() <= 50)
			time.Sleep(time.Millisecond / 10)
		}
	}()

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				if ol.takeConn() {
					time.Sleep(time.Millisecond)
					ol.releaseConn()
				} else {
					time.Sleep(time.Millisecond)
				}
			}
		}()
	}

	wg.Wait()
}

func TestOverloader_TotalQPS(t *testing.T) {
	ol := NewOverloader(LimitCond{
		MaxConn:     100,
		MaxTotalQPS: 100,
		QPSInterval: 100 * time.Millisecond,
	})
	for i := 0; i < 100; i++ {
		assert.True(t, ol.takeTotalQPS())
	}
	for i := 0; i < 100; i++ {
		assert.False(t, ol.takeTotalQPS())
	}
	time.Sleep(1*time.Second + 10*time.Millisecond)
	for i := 0; i < 100; i++ {
		assert.True(t, ol.takeTotalQPS())
	}
	for i := 0; i < 100; i++ {
		assert.False(t, ol.takeTotalQPS())
	}

	var wg sync.WaitGroup
	limitCond := ol.LimitCond()
	limitCond.MaxTotalQPS = 10000
	ol.UpdateLimitCond(limitCond)
	time.Sleep(time.Second)
	var success int32
	for i := 0; i <= 100000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok := ol.takeTotalQPS()
			if ok {
				atomic.AddInt32(&success, 1)
			}
		}()
	}
	wg.Wait()
	if success >= 11000 {
		t.Log(success)
		t.Fail()
	}
}
