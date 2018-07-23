package tp

import (
	"errors"
	"testing"

	"github.com/henrylee2cn/teleport/utils"
)

func TestRerror(t *testing.T) {
	rerr := new(Rerror)
	t.Logf("%v", rerr)
	rerr.Code = 400
	rerr.Message = "msg"
	t.Logf("%v", rerr)
	rerr.Reason = `"bala...bala..."`
	t.Logf("%v", rerr)
	meta := new(utils.Args)
	rerr.SetToMeta(meta)
	t.Logf("%v", meta.String())
	b := meta.Peek(MetaRerror)
	t.Logf("%s", b)
	newRerr := NewRerrorFromMeta(meta)
	t.Logf("%v", newRerr)
	err := newRerr.ToError()
	t.Logf("test ToError 1: %v", err)
	newRerr = ToRerror(err)
	t.Logf("test ToRerror 1: %s", newRerr)
	newRerr = nil
	err = newRerr.ToError()
	t.Logf("test ToError 2: %v", err)
	newRerr = ToRerror(nil)
	t.Logf("test ToRerror 2: %s", newRerr)
	newRerr = ToRerror(errors.New("text error"))
	t.Logf("test ToRerror 3: %s", newRerr)
}
