package socket

import (
	"testing"
)

func TestMessageString(t *testing.T) {
	var m = NewMessage()
	m.SetSeq("21")
	m.XferPipe().Append('g')
	m.SetMtype(3)
	m.SetSize(300)
	m.SetBody(map[string]int{"a": 1})
	m.SetUri("uri/b")
	m.SetBodyCodec(5)
	m.Meta().Set("key", "value")
	t.Logf("%%s:%s", m.String())
	t.Logf("%%v:%v", m)
	t.Logf("%%#v:%#v", m)
	t.Logf("%%+v:%+v", m)
}
