package gzip_test

import (
	"testing"

	"github.com/henrylee2cn/teleport/xfer"
	"github.com/henrylee2cn/teleport/xfer/gzip"
)

func TestGzip(t *testing.T) {
	// test register
	gzip.Reg('g', "gzip-5", 5)

	if _, err := xfer.Get('g'); err != nil {
		t.Fatal(err)
	}
	if _, err := xfer.GetByName("gzip-5"); err != nil {
		t.Fatal(err)
	}
	xferPipe := xfer.NewXferPipe()
	xferPipe.Append('g')
	t.Logf("transfer filter: ids:%v, names:%v", xferPipe.Ids(), xferPipe.Names())

	// test logic
	b, err := xferPipe.OnPack([]byte("src"))
	if err != nil {
		t.Fatalf("nopack: %v", err)
	}
	src, err := xferPipe.OnUnpack(b)
	if err != nil {
		t.Fatalf("nounpack: %v", err)
	}
	if string(src) != "src" {
		t.Fatalf("gunzip has error: want \"src\", have %q", string(src))
	}
}
