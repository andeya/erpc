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

	{
		var sess, err = peer.Dial("127.0.0.1:9090")
		if err != nil {
			teleport.Panicf("%v", err)
		}
		var reply string
		var pullcmd = sess.Pull("/group/home/test?port=9090", "test_args_9090", &reply)
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
		var pullcmd = sess.Pull("/group/home/test?port=9091", "test_args_9091", &reply)
		if pullcmd.Xerror != nil {
			teleport.Fatalf("pull error: %v", pullcmd.Xerror.Error())
		}
		teleport.Infof("9091reply: %s", reply)
	}
}
