package main

import (
	"time"

	"github.com/json-iterator/go"

	tp "github.com/henrylee2cn/teleport"
)

func main() {
	go tp.GraceSignal()
	tp.SetShutdown(time.Second*20, nil, nil)
	var cfg = &tp.PeerConfig{
		DefaultReadTimeout:   time.Minute * 3,
		DefaultWriteTimeout:  time.Minute * 3,
		TlsCertFile:          "",
		TlsKeyFile:           "",
		SlowCometDuration:    time.Millisecond * 500,
		DefaultHeaderCodec:   "protobuf",
		DefaultBodyCodec:     "json",
		DefaultBodyGzipLevel: 5,
		PrintBody:            false,
	}

	var peer = tp.NewPeer(cfg)
	peer.PushRouter.Reg(new(Push))

	{
		var sess, err = peer.Dial("127.0.0.1:9090", "simple_server:9090")
		if err != nil {
			tp.Panicf("%v", err)
		}

		var reply interface{}
		var pullcmd = sess.Pull(
			"/group/home/test?peer_id=client9090",
			map[string]interface{}{
				"conn_port": 9090,
				"bytes":     []byte("bytestest9090"),
			},
			&reply,
		)

		if pullcmd.Xerror != nil {
			tp.Fatalf("pull error: %v", pullcmd.Xerror.Error())
		}
		tp.Infof("9090reply: %#v", reply)
	}

	{
		var sess, err = peer.Dial("127.0.0.1:9091")
		if err != nil {
			tp.Panicf("%v", err)
		}

		var reply interface{}
		var pullcmd = sess.Pull(
			"/group/home/test_unknown?peer_id=client9091",
			struct {
				ConnPort int
				jsoniter.RawMessage
				Bytes []byte
			}{
				9091,
				jsoniter.RawMessage(`{"RawMessage":"test9091"}`),
				[]byte("bytes-test"),
			},
			&reply,
		)

		if pullcmd.Xerror != nil {
			tp.Fatalf("pull error: %v", pullcmd.Xerror.Error())
		}
		tp.Infof("9091reply test_unknown: %#v", reply)
	}
}

// Push controller
type Push struct {
	tp.PushCtx
}

// Test handler
func (p *Push) Test(args *map[string]interface{}) {
	tp.Infof("receive push(%s):\nargs: %#v\nquery: %#v\n", p.Ip(), args, p.Query())
}
