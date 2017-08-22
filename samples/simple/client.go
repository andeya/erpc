package main

import (
	"time"

	"github.com/henrylee2cn/teleport"
)

func main() {
	var cfg = &teleport.Config{
		Id:                       "client-peer",
		ReadTimeout:              time.Second * 10,
		WriteTimeout:             time.Second * 10,
		TlsCertFile:              "",
		TlsKeyFile:               "",
		SlowCometDuration:        time.Millisecond * 500,
		DefaultCodec:             "json",
		DefaultGzipLevel:         5,
		MaxGoroutinesAmount:      1024,
		MaxGoroutineIdleDuration: time.Second * 10,
	}

	var peer = teleport.NewPeer(cfg)
	peer.PushRouter.Reg(new(Push))

	{
		var sess, err = peer.Dial("127.0.0.1:9090")
		if err != nil {
			teleport.Panicf("%v", err)
		}

		var reply string
		var pullcmd = sess.Pull("/group/home/test?peer_id=client9090", "test_args_9090", &reply)
		if pullcmd.Xerror != nil {
			teleport.Fatalf("pull error: %v", pullcmd.Xerror.Error())
		}
		teleport.Infof("9090reply: %s", reply)
	}

	{
		var sess, err = peer.Dial("127.0.0.1:9091")
		if err != nil {
			teleport.Panicf("%v", err)
		}

		var reply string
		var pullcmd = sess.Pull("/group/home/test?peer_id=client9091", "test_args_9091", &reply)
		if pullcmd.Xerror != nil {
			teleport.Fatalf("pull error: %v", pullcmd.Xerror.Error())
		}
		teleport.Infof("9091reply: %s", reply)
	}
}

// Push controller
type Push struct {
	teleport.PushCtx
}

// Test handler
func (p *Push) Test(args *map[string]interface{}) {
	teleport.Infof("push-test:\nargs: %#v\nquery: %#v\n", args, p.Query())
}
