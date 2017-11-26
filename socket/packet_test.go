package socket

import (
	"testing"
)

func TestPacketString(t *testing.T) {
	var p = NewPacket(nil)
	p.SetSeq(21)
	p.SetPtype(3)
	p.SetSize(300)
	p.SetBody(map[string]int{"a": 1})
	p.SetUri("uri/b")
	p.SetBodyCodec(5)
	p.Meta().Set("key", "value")
	t.Logf("%%s:%s", p.String())
	t.Logf("%%v:%v", p)
	t.Logf("%%#v:%#v", p)
	t.Logf("%%+v:%+v", p)
}
