package tp_test

import (
	"testing"

	tp "github.com/henrylee2cn/teleport"
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
