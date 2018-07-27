package main

import (
	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/plugin/auth"
)

func main() {
	srv := tp.NewPeer(
		tp.PeerConfig{
			ListenPort: 9090,
		},
		auth.VerifyAuth(verifyAuthInfo),
	)
	srv.ListenAndServe()
}

const (
	clientAuthInfo = "client-auth-info-12345"
	codeAuthFail   = 403
	textAuthFail   = "auth fail"
	detailAuthFail = "auth fail detail"
)

func verifyAuthInfo(authInfo string, sess auth.AuthSession) *tp.Rerror {
	tp.Infof("auth info: %v", authInfo)
	if clientAuthInfo != authInfo {
		return tp.NewRerror(codeAuthFail, textAuthFail, detailAuthFail)
	}
	return nil
}
