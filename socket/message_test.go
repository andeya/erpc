package socket

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/henrylee2cn/erpc/v6/xfer/gzip"
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
	m.SetStatus(NewStatus(400, "this is msg", "this is cause"))
	m.Meta().Set("key", "value")
	t.Logf("%%s:%s", m.String())
	t.Logf("%%v:%v", m)
	t.Logf("%%#v:%#v", m)
	t.Logf("%%+v:%+v", m)
}

func TestUint32Minus(t *testing.T) {
	a := 1
	a, err := minus(a, 4)
	assert.EqualError(t, err, "raw proto: bad package")
	assert.Equal(t, int(1), a)
	a, err = minus(a, 0)
	assert.NoError(t, err)
	assert.Equal(t, int(1), a)
	a, err = minus(a, 1)
	assert.NoError(t, err)
	assert.Equal(t, int(0), a)
	a, err = minus(a, 1)
	assert.EqualError(t, err, "raw proto: bad package")
	assert.Equal(t, int(0), a)
}
