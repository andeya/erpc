// Package pbSubProto is implemented PROTOBUF socket communication protocol.
package pbSubProto

import (
	"io"
	"io/ioutil"
	"sync"

	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/codec"
	"github.com/henrylee2cn/teleport/mixer/websocket/pbSubProto/pb"
)

// NewPbSubProtoFunc is creation function of PROTOBUF socket protocol.
var NewPbSubProtoFunc = func(rw io.ReadWriter) tp.Proto {
	return &pbSubProto{
		id:   'p',
		name: "protobuf",
		rw:   rw,
	}
}

type pbSubProto struct {
	id   byte
	name string
	rw   io.ReadWriter
	rMu  sync.Mutex
}

// Version returns the protocol's id and name.
func (psp *pbSubProto) Version() (byte, string) {
	return psp.id, psp.name
}

// Pack writes the Message into the connection.
// Note: Make sure to write only once or there will be package contamination!
func (psp *pbSubProto) Pack(m *tp.Message) error {
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

	_, err = psp.rw.Write(b)
	return err
}

// Unpack reads bytes from the connection to the Message.
// Note: Concurrent unsafe!
func (psp *pbSubProto) Unpack(m *tp.Message) error {
	psp.rMu.Lock()
	defer psp.rMu.Unlock()
	b, err := ioutil.ReadAll(psp.rw)
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
