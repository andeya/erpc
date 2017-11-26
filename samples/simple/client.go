package main

import (
	"encoding/json"
	"time"

	tp "github.com/henrylee2cn/teleport"
)

func main() {
	go tp.GraceSignal()
	tp.SetShutdown(time.Second*20, nil, nil)
	var cfg = &tp.PeerConfig{
		DefaultReadTimeout:  time.Minute * 5,
		DefaultWriteTimeout: time.Millisecond * 500,
		TlsCertFile:         "",
		TlsKeyFile:          "",
		SlowCometDuration:   time.Millisecond * 500,
		DefaultBodyCodec:    "json",
		PrintBody:           true,
		CountTime:           true,
	}

	var peer = tp.NewPeer(cfg)
	defer peer.Close()
	peer.PushRouter.Reg(new(Push))

	{
		var sess, err = peer.Dial("127.0.0.1:9090")
		if err != nil {
			tp.Fatalf("%v", err)
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

		if pullcmd.Rerror() != nil {
			tp.Fatalf("pull error: %v", pullcmd.Rerror())
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
				ConnPort   int
				RawMessage json.RawMessage
				Bytes      []byte
			}{
				9091,
				json.RawMessage(`{"RawMessage":"test9091"}`),
				[]byte("bytes-test"),
			},
			&reply,
		)

		if pullcmd.Rerror() != nil {
			tp.Fatalf("pull error: %v", pullcmd.Rerror())
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
