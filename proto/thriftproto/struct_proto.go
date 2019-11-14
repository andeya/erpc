package thriftproto

import (
	"errors"
	"fmt"

	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/henrylee2cn/erpc/v6"
	"github.com/henrylee2cn/erpc/v6/codec"
	"github.com/henrylee2cn/erpc/v6/utils"
	"github.com/henrylee2cn/goutil"
)

// NewStructProtoFunc creates erpc.ProtoFunc of Thrift protocol.
// NOTE:
//  The body codec must be thrift, directly encoded as a thrift.TStruct;
//  Support the Meta, but not support the BodyCodec and XferPipe.
func NewStructProtoFunc() erpc.ProtoFunc {
	return func(rw erpc.IOWithReadBuffer) erpc.Proto {
		p := &tStructProto{
			id:        's',
			name:      "thrift-struct",
			rwCounter: utils.NewReadWriteCounter(rw),
		}
		p.tProtocol = thrift.NewTHeaderProtocol(&BaseTTransport{
			ReadWriteCounter: p.rwCounter,
		})
		return p
	}
}

type tStructProto tBinaryProto

// Version returns the protocol's id and name.
func (t *tStructProto) Version() (byte, string) {
	return t.id, t.name
}

// Pack writes the Message into the connection.
// NOTE: Make sure to write only once or there will be package contamination!
func (t *tStructProto) Pack(m erpc.Message) error {
	err := t.structPack(m)
	if err != nil {
		t.tProtocol.Transport().Close()
	}
	return err
}

func (t *tStructProto) Unpack(m erpc.Message) error {
	err := t.structUnpack(m)
	if err != nil {
		t.tProtocol.Transport().Close()
	}
	return err
}

func (t *tStructProto) structPack(m erpc.Message) error {
	if m.XferPipe().Len() > 0 {
		return errors.New("unsupport transfer pipe")
	}
	bodyCodec := m.BodyCodec()
	if bodyCodec == codec.NilCodecID {
		m.SetBodyCodec(codec.ID_THRIFT)
	} else if bodyCodec != codec.ID_THRIFT {
		return errors.New("body codec must be thrift")
	}
	t.packLock.Lock()
	defer t.packLock.Unlock()
	t.rwCounter.WriteCounter.Zero()

	err := writeMessageBegin(t.tProtocol, m)
	if err != nil {
		return err
	}

	s, ok := m.Body().(thrift.TStruct)
	if !ok {
		return fmt.Errorf("thrift codec: %T does not implement thrift.TStruct", m.Body())
	}
	if err = s.Write(t.tProtocol); err != nil {
		return err
	}

	t.tProtocol.ClearWriteHeaders()
	t.tProtocol.SetWriteHeader(HeaderStatus, m.Status(true).QueryString())
	t.tProtocol.SetWriteHeader(HeaderMeta, goutil.BytesToString(m.Meta().QueryString()))

	if err = t.tProtocol.WriteMessageEnd(); err != nil {
		return err
	}
	if err = t.tProtocol.Flush(m.Context()); err != nil {
		return err
	}

	return m.SetSize(uint32(t.rwCounter.Writed()))
}

func (t *tStructProto) structUnpack(m erpc.Message) error {
	t.unpackLock.Lock()
	defer t.unpackLock.Unlock()
	t.rwCounter.WriteCounter.Zero()
	err := readMessageBegin(t.tProtocol, m)
	if err != nil {
		return err
	}

	m.UnmarshalBody(nil)
	s, ok := m.Body().(thrift.TStruct)
	if !ok {
		return fmt.Errorf("thrift codec: %T does not implement thrift.TStruct", m.Body())
	}
	if err = s.Read(t.tProtocol); err != nil {
		return err
	}

	if err = t.tProtocol.ReadMessageEnd(); err != nil {
		return err
	}

	headers := t.tProtocol.GetReadHeaders()
	m.Status(true).DecodeQuery(goutil.StringToBytes(headers[HeaderStatus]))
	m.Meta().Parse(headers[HeaderMeta])

	m.SetBodyCodec(codec.ID_THRIFT)
	return m.SetSize(uint32(t.rwCounter.Readed()))
}
