package tp_test

import (
	"testing"

	"github.com/henrylee2cn/tp/v6"
	"github.com/stretchr/testify/assert"
)

func TestHTTPServiceMethodMapper(t *testing.T) {
	var cases = []struct{ src, dst string }{
		{"AaBb", "/aa_bb"},
		{"ABcXYz", "/abc_xyz"},
		{"Aa__Bb", "/aa_bb"},
		{"Aa________Bb", "/aa_bb"},
		{"aa__bb", "/aa_bb"},
		{"ABC__XYZ", "/abc_xyz"},
		{"Aa_Bb", "/aa/bb"},
		{"aa_bb", "/aa/bb"},
		{"ABC_XYZ", "/abc/xyz"},
	}
	for _, c := range cases {
		got := tp.HTTPServiceMethodMapper("", c.src)
		if got != c.dst {
			t.Fatalf("%s: got: %s, expect: %s", c.src, got, c.dst)
		}
	}
}

func TestRPCServiceMethodMapper(t *testing.T) {
	var cases = []struct{ src, dst string }{
		{"AaBb", "AaBb"},
		{"ABcXYz", "ABcXYz"},
		{"Aa__Bb", "Aa_Bb"},
		{"Aa________Bb", "Aa_Bb"},
		{"aa__bb", "aa_bb"},
		{"ABC__XYZ", "ABC_XYZ"},
		{"Aa_Bb", "Aa.Bb"},
		{"aa_bb", "aa.bb"},
		{"ABC_XYZ", "ABC.XYZ"},
	}
	for _, c := range cases {
		got := tp.RPCServiceMethodMapper("", c.src)
		if got != c.dst {
			t.Fatalf("%s: got: %s, expect: %s", c.src, got, c.dst)
		}
	}
}

func TestFakeAddr(t *testing.T) {
	addr := tp.NewFakeAddr("", "", "")
	assert.Equal(t, "0.0.0.0:0", addr.String())
	assert.Equal(t, "tcp", addr.Network())

	addr = tp.NewFakeAddr("tcp", "", "1234")
	assert.Equal(t, "0.0.0.0:1234", addr.String())

	addr, err := tp.NewFakeAddr2("", "")
	assert.NoError(t, err)
	assert.Equal(t, "0.0.0.0:0", addr.String())
	assert.Equal(t, "tcp", addr.Network())
	assert.Equal(t, "0.0.0.0", addr.Host())
	assert.Equal(t, "0", addr.Port())

	addr, err = tp.NewFakeAddr2("tcp6", ":1234")
	assert.NoError(t, err)
	assert.Equal(t, "0.0.0.0:1234", addr.String())
	assert.Equal(t, "tcp6", addr.Network())
	assert.Equal(t, "0.0.0.0", addr.Host())
	assert.Equal(t, "1234", addr.Port())

	addr, err = tp.NewFakeAddr2("tcp6", "192.0.0.10:1234")
	assert.NoError(t, err)
	assert.Equal(t, "192.0.0.10:1234", addr.String())
	assert.Equal(t, "tcp6", addr.Network())
	assert.Equal(t, "192.0.0.10", addr.Host())
	assert.Equal(t, "1234", addr.Port())
}
