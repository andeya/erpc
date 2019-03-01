package main

import (
	"flag"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	proto "github.com/gogo/protobuf/proto"
	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/examples/bench/msg"
	"github.com/montanaflynn/stats"
)

//go:generate go build $GOFILE

var concurrency = flag.Int("c", 1, "concurrency")
var total = flag.Int("n", 1, "total requests for all clients")
var host = flag.String("s", "127.0.0.1:8972", "server ip and port")

func main() {
	flag.Parse()
	n := *concurrency
	m := *total / n

	selected := -1
	servers := strings.Split(*host, ",")
	sNum := len(servers)

	log.Printf("Servers: %+v\n\n", servers)

	log.Printf("concurrency: %d\nrequests per client: %d\n\n", n, m)

	defer tp.SetLoggerLevel("ERROR")()
	tp.SetGopool(1024*1024*100, time.Minute*10)

	serviceMethod := "Hello.Say"
	client := tp.NewPeer(tp.PeerConfig{
		DefaultBodyCodec: "protobuf",
	})

	args := msg.PrepareArgs()

	b, _ := proto.Marshal(args)
	log.Printf("message size: %d bytes\n\n", len(b))

	var wg sync.WaitGroup
	wg.Add(n * m)

	var trans uint64
	var transOK uint64

	d := make([][]int64, n, n)

	//it contains warmup time but we can ignore it
	totalT := time.Now().UnixNano()
	for i := 0; i < n; i++ {
		dt := make([]int64, 0, m)
		d = append(d, dt)
		selected = (selected + 1) % sNum

		go func(i int, selected int) {
			sess, rerr := client.Dial(servers[selected])
			if rerr != nil {
				log.Fatalf("did not connect: %v", rerr)
			}
			defer sess.Close()
			var reply msg.BenchmarkMessage

			//warmup
			for j := 0; j < 5; j++ {
				sess.Call(serviceMethod, args, &reply)
			}

			for j := 0; j < m; j++ {
				t := time.Now().UnixNano()
				rerr := sess.Call(serviceMethod, args, &reply).Rerror()
				t = time.Now().UnixNano() - t

				d[i] = append(d[i], t)

				if rerr == nil && reply.Field1 == "OK" {
					atomic.AddUint64(&transOK, 1)
				}

				// if rerr != nil {
				// 	log.Print(rerr.String())
				// }

				atomic.AddUint64(&trans, 1)
				wg.Done()
			}

		}(i, selected)

	}

	wg.Wait()
	totalT = time.Now().UnixNano() - totalT
	totalT = totalT / 1000000
	tp.Printf("took %d ms for %d requests", totalT, n*m)

	totalD := make([]int64, 0, n*m)
	for _, k := range d {
		totalD = append(totalD, k...)
	}
	totalD2 := make([]float64, 0, n*m)
	for _, k := range totalD {
		totalD2 = append(totalD2, float64(k))
	}

	mean, _ := stats.Mean(totalD2)
	median, _ := stats.Median(totalD2)
	max, _ := stats.Max(totalD2)
	min, _ := stats.Min(totalD2)
	p99, _ := stats.Percentile(totalD2, 99.9)

	tp.Printf("sent     requests    : %d\n", n*m)
	tp.Printf("received requests    : %d\n", atomic.LoadUint64(&trans))
	tp.Printf("received requests_OK : %d\n", atomic.LoadUint64(&transOK))
	tp.Printf("throughput  (TPS)    : %d\n", int64(n*m)*1000/totalT)
	tp.Printf("mean: %.f ns, median: %.f ns, max: %.f ns, min: %.f ns, p99: %.f ns\n", mean, median, max, min, p99)
	tp.Printf("mean: %d ms, median: %d ms, max: %d ms, min: %d ms, p99: %d ms\n", int64(mean/1000000), int64(median/1000000), int64(max/1000000), int64(min/1000000), int64(p99/1000000))

}
