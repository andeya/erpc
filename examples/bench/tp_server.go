package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"time"

	tp "github.com/henrylee2cn/teleport"
)

//go:generate go build $GOFILE benchmark.pb.go

type Hello struct {
	tp.CallCtx
}

func (t *Hello) Say(args *BenchmarkMessage) (*BenchmarkMessage, *tp.Rerror) {
	s := "OK"
	var i int32 = 100
	args.Field1 = s
	args.Field2 = i
	if *delay > 0 {
		time.Sleep(*delay)
	} else {
		runtime.Gosched()
	}
	return args, nil
}

var (
	port      = flag.Int64("p", 8972, "listened port")
	delay     = flag.Duration("delay", 0, "delay to mock business processing")
	debugAddr = flag.String("d", "127.0.0.1:9981", "server ip and port")
)

func main() {
	flag.Parse()

	tp.SetLoggerLevel("ERROR")
	tp.SetGopool(1024*1024*100, time.Minute*10)

	go func() {
		log.Println(http.ListenAndServe(*debugAddr, nil))
	}()

	tp.SetServiceMethodMapper(tp.RPCServiceMethodMapper)
	server := tp.NewPeer(tp.PeerConfig{
		DefaultBodyCodec: "protobuf",
		ListenPort:       uint16(*port),
	})
	server.RouteCall(new(Hello))
	server.ListenAndServe()
}
