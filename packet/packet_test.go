package packet

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net"
)

func init() {
	conn, _ := net.Dial("tcp", "127.0.0.1:8000")
	bufconn := bufio.NewWriter(conn)
	zip := gzip.NewWriter(bufconn)
	EncodePacket(zip, []byte("hello"))
}

func EncodePacket(w io.Writer, body []byte) error {
	// len(Magic) + len(Checksum) == 8
	totalsize := uint32(len(RPC_MAGIC) + len(body) + 4)
	// write total size
	binary.Write(w, binary.BigEndian, totalsize)

	sum := adler32.New()
	ww := io.MultiWriter(sum, w)
	// write magic bytes
	binary.Write(ww, binary.BigEndian, RPC_MAGIC)

	// write body
	ww.Write(body)

	// calculate checksum
	checksum := sum.Sum32()

	// write checksum
	return binary.Write(w, binary.BigEndian, checksum)
}

func DecodePacket(r io.Reader) ([]byte, error) {
	var totalsize uint32
	err := binary.Read(r, binary.BigEndian, &totalsize)
	if err != nil {
		return nil, errors.Annotate(err, "read total size")
	}

	// at least len(magic) + len(checksum)
	if totalsize < 8 {
		return nil, errors.Errorf("bad packet. header:%d", totalsize)
	}

	sum := adler32.New()
	rr := io.TeeReader(r, sum)

	var magic [4]byte
	err = binary.Read(rr, binary.BigEndian, &magic)
	if err != nil {
		return nil, errors.Annotate(err, "read magic")
	}
	if magic != RPC_MAGIC {
		return nil, errors.Errorf("bad rpc magic:%v", magic)
	}

	body := make([]byte, totalsize-8)
	_, err = io.ReadFull(rr, body)
	if err != nil {
		return nil, errors.Annotate(err, "read body")
	}

	var checksum uint32
	err = binary.Read(r, binary.BigEndian, &checksum)
	if err != nil {
		return nil, errors.Annotate(err, "read checksum")
	}

	if checksum != sum.Sum32() {
		return nil, errors.Errorf("checkSum error, %d(calc) %d(remote)", sum.Sum32(), checksum)
	}
	return body, nil
}
