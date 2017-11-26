package tp

import (
	"testing"

	"github.com/henrylee2cn/teleport/socket"
)

func TestRerror(t *testing.T) {
	reErr := new(Rerror)
	t.Logf("%v", reErr)
	reErr.Code = 400
	reErr.Message = "msg"
	t.Logf("%v", reErr)
	reErr.Detail = `"bala...bala..."`
	t.Logf("%v", reErr)
	header := new(socket.Header)
	reErr.SetToMeta(header)
	t.Logf("%v", header.Meta.String())
	b := header.Peek(MetaRerrorKey)
	t.Logf("%s", b)
	newReErr := NewRerrorFromMeta(header)
	t.Logf("%v", newReErr)
}
