package socket

import (
	"testing"

	"github.com/henrylee2cn/teleport/xfer/gzip"
)

func TestMessageString(t *testing.T) {
	gzip.Reg('g', "gzip", 5)

	var m = GetMessage()
	defer PutMessage(m)
	m.SetSeq(21)

	m.XferPipe().Append('g')
	m.SetMtype(3)
	m.SetSize(300)
	m.SetBody(map[string]int{"a": 1})
	m.SetServiceMethod("service/method")
	m.SetBodyCodec(5)
	m.Meta().Set("key", "value")
	t.Logf("%%s:%s", m.String())
	t.Logf("%%v:%v", m)
	t.Logf("%%#v:%#v", m)
	t.Logf("%%+v:%+v", m)
}
