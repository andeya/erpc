package main

import (
	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/plugin/auth"
)

//go:generate go build $GOFILE

func main() {
	defer tp.FlushLogger()
	srv := tp.NewPeer(
		tp.PeerConfig{
			ListenPort: 9090,
		},
		auth.NewCheckerPlugin(authChecker),
	)
	srv.ListenAndServe()
}

const (
	clientAuthInfo = "client-auth-info-12345"
	codeAuthFail   = 403
	textAuthFail   = "auth fail"
	detailAuthFail = "auth fail detail"
)

func authChecker(sess auth.Session, fn auth.Receiver) (ret interface{}, rerr *tp.Rerror) {
	var authInfo string
	rerr = fn(&authInfo)
	if rerr.HasError() {
		return
	}
	tp.Infof("auth info: %v", authInfo)
	if clientAuthInfo != authInfo {
		return nil, tp.NewRerror(codeAuthFail, textAuthFail, detailAuthFail)
	}
	return "pass", nil
}
