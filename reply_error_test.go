package tp

import (
	"testing"

	"github.com/henrylee2cn/teleport/utils"
)

func TestRerror(t *testing.T) {
	rerr := new(Rerror)
	t.Logf("%v", rerr)
	rerr.Code = 400
	rerr.Message = "msg"
	t.Logf("%v", rerr)
	rerr.Detail = `"bala...bala..."`
	t.Logf("%v", rerr)
	meta := new(utils.Args)
	rerr.SetToMeta(meta)
	t.Logf("%v", meta.String())
	b := meta.Peek(MetaRerrorKey)
	t.Logf("%s", b)
	newRerr := NewRerrorFromMeta(meta)
	t.Logf("%v", newRerr)
	t.Logf("test ToError: %s", newRerr.ToError().Error())
}
