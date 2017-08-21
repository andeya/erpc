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
		MaxGoroutineIdleDuration: time.Second * 1,
	}

	var peer = teleport.NewPeer(cfg)

	var sess, err = peer.Dial("127.0.0.1:9090")
	if err != nil {
		teleport.Panicf("%v", err)
	}
	var reply string
	pullcmd := sess.Pull("/group/home/test", "test_args", reply)
	if pullcmd.Xerror != nil {
		teleport.Fatalf("pull error: %v", pullcmd.Xerror.Error())
	}
	teleport.Infof("reply: %s", reply)
}
