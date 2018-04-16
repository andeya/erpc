package xfer_test

import (
	"testing"

	"github.com/henrylee2cn/teleport/xfer"
)

func TestGzip(t *testing.T) {
	gzip := xfer.NewGzip('0', "gzip-5", 5)

	// test register
	xfer.Reg(gzip)
	if _, err := xfer.Get('0'); err != nil {
		t.Fatal(err)
	}
	if _, err := xfer.GetByName("gzip-5"); err != nil {
		t.Fatal(err)
	}
	xferPipe := xfer.NewXferPipe()
	xferPipe.Append('0')
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
