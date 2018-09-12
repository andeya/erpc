// Package pbSubProto is implemented PROTOBUF socket communication protocol.
package pbSubProto

import (
	"bufio"
	"io"
	"io/ioutil"
	"sync"

	"github.com/henrylee2cn/teleport/codec"
	"github.com/henrylee2cn/teleport/mixer/websocket/pbSubProto/pb"
	"github.com/henrylee2cn/teleport/socket"
)

// NewPbSubProtoFunc is creation function of PROTOBUF socket protocol.
var NewPbSubProtoFunc = func(rw io.ReadWriter) socket.Proto {
	var (
		readBufioSize             int
		readBufferSize, isDefault = socket.ReadBuffer()
	)
	if isDefault {
		readBufioSize = 1024 * 4
	} else if readBufferSize == 0 {
		readBufioSize = 1024 * 35
	} else {
		readBufioSize = readBufferSize / 2
	}
	return &pbSubProto{
		id:   'p',
		name: "protobuf",
		r:    bufio.NewReaderSize(rw, readBufioSize),
		w:    rw,
	}
}

type pbSubProto struct {
	id   byte
	name string
	r    *bufio.Reader
	w    io.Writer
	rMu  sync.Mutex
}

// Version returns the protocol's id and name.
func (psp *pbSubProto) Version() (byte, string) {
	return psp.id, psp.name
}

// Pack writes the Message into the connection.
// Note: Make sure to write only once or there will be package contamination!
func (psp *pbSubProto) Pack(m *socket.Message) error {
	// marshal body
	bodyBytes, err := m.MarshalBody()
	if err != nil {
		return err
	}
	// do transfer pipe
	bodyBytes, err = m.XferPipe().OnPack(bodyBytes)
	if err != nil {
		return err
	}

	b, err := codec.ProtoMarshal(&pb.Format{
		Seq:       m.Seq(),
		Mtype:     int32(m.Mtype()),
		Uri:       m.Uri(),
		Meta:      m.Meta().QueryString(),
		BodyCodec: int32(m.BodyCodec()),
		Body:      bodyBytes,
		XferPipe:  m.XferPipe().Ids(),
	})
	if err != nil {
		return err
	}

	m.SetSize(uint32(len(b)))

	_, err = psp.w.Write(b)
	return err
}

// Unpack reads bytes from the connection to the Message.
// Note: Concurrent unsafe!
func (psp *pbSubProto) Unpack(m *socket.Message) error {
	psp.rMu.Lock()
	defer psp.rMu.Unlock()
	b, err := ioutil.ReadAll(psp.r)
	if err != nil {
		return err
	}

	m.SetSize(uint32(len(b)))

	s := &pb.Format{}
	err = codec.ProtoUnmarshal(b, s)
	if err != nil {
		return err
	}

	// read transfer pipe
	for _, r := range s.XferPipe {
		m.XferPipe().Append(r)
	}

	// read body
	m.SetBodyCodec(byte(s.BodyCodec))
	bodyBytes, err := m.XferPipe().OnUnpack(s.Body)
	if err != nil {
		return err
	}

	// read other
	m.SetSeq(s.Seq)
	m.SetMtype(byte(s.Mtype))
	m.SetUri(s.Uri)
	m.Meta().ParseBytes(s.Meta)

	// unmarshal new body
	err = m.UnmarshalBody(bodyBytes)
	return err
}
