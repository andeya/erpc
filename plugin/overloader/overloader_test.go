package overloader

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/henrylee2cn/erpc/v6"
	"github.com/stretchr/testify/assert"
)

type Home struct {
	erpc.CallCtx
}

func (h *Home) Test(arg *map[string]string) (map[string]interface{}, *erpc.Status) {
	return map[string]interface{}{
		"arg": *arg,
	}, nil
}

func TestPlugin(t *testing.T) {
	ol := New(LimitConfig{
		MaxConn:     1,
		QPSInterval: 100 * time.Millisecond,
		MaxTotalQPS: 2,
		MaxHandlerQPS: []HandlerLimit{
			{ServiceMethod: "/home/test", MaxQPS: 1},
		},
	})
	// Server
	srv := erpc.NewPeer(
		erpc.PeerConfig{ListenPort: 9090, CountTime: true},
		ol,
	)
	srv.RouteCall(new(Home))
	go srv.ListenAndServe()
	time.Sleep(1e9)

	// Client
	cli := erpc.NewPeer(
		erpc.PeerConfig{CountTime: true},
	)
	var testClient = func(connNum, totalQPS int) (olConnCount, olQPSCount int64) {
		var connGW sync.WaitGroup
		connGW.Add(connNum)
		defer connGW.Wait()
		for index := 0; index < connNum; index++ {
			go func() {
				defer connGW.Done()
				sess, stat := cli.Dial(":9090")
				if !stat.OK() {
					t.Fatal(stat)
				}
				defer sess.Close()
				time.Sleep(time.Millisecond)
				if !sess.Health() {
					atomic.AddInt64(&olConnCount, 1)
					t.Logf("connNum:%d, totalQPS:%d, dial: Connection Closed", connNum, totalQPS)
					return
				}
				var qpsGW sync.WaitGroup
				qpsGW.Add(totalQPS)
				defer qpsGW.Wait()
				for i := 0; i < totalQPS; i++ {
					go func() {
						defer qpsGW.Done()
						stat = sess.Call("/home/test", nil, nil).Status()
						if !stat.OK() {
							atomic.AddInt64(&olQPSCount, 1)
							t.Logf("connNum:%d, totalQPS:%d, call:%s", connNum, totalQPS, stat)
						}
					}()
				}
			}()
		}
		return
	}

	{
		olConnCount, olQPSCount := testClient(1, 1)
		assert.Equal(t, int64(0), olConnCount)
		assert.Equal(t, int64(0), olQPSCount)
	}
	{
		time.Sleep(time.Second)
		assert.Equal(t, 0, srv.CountSession())
		olConnCount, olQPSCount := testClient(2, 1)
		assert.Equal(t, int64(1), olConnCount)
		assert.Equal(t, int64(0), olQPSCount)
	}
	{
		time.Sleep(time.Second)
		assert.Equal(t, 0, srv.CountSession())
		olConnCount, olQPSCount := testClient(1, 2)
		assert.Equal(t, int64(0), olConnCount)
		assert.Equal(t, int64(1), olQPSCount)
	}
	{
		ol.Update(LimitConfig{
			MaxConn:     100,
			QPSInterval: time.Second,
			MaxTotalQPS: 2000,
		})
		time.Sleep(time.Second) // Wait for one cycle to take effect
		assert.Equal(t, 0, srv.CountSession())
		olConnCount, olQPSCount := testClient(10, 200)
		assert.Equal(t, int64(0), olConnCount)
		assert.Equal(t, int64(0), olQPSCount)
	}
}

func TestConn(t *testing.T) {
	ol := New(LimitConfig{
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

func TestTotalQPS(t *testing.T) {
	ol := New(LimitConfig{
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
	limitCond := ol.LimitConfig()
	limitCond.MaxTotalQPS = 10000
	ol.Update(limitCond)
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
