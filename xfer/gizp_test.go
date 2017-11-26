package xfer

import (
	"testing"
)

func TestGzip(t *testing.T) {
	gzip := newGzip('g', 5)
	b, err := gzip.OnPack([]byte("src"))
	if err != nil {
		t.Fatalf("nopack: %v", err)
	}
	src, err := gzip.OnUnpack(b)
	if err != nil {
		t.Fatalf("nounpack: %v", err)
	}
	if string(src) != "src" {
		t.Fatalf("gunzip has error: want \"src\", have %q", string(src))
	}
	t.Logf("gunzip ok: want \"src\", have %q", string(src))
}
