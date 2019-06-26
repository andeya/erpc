package thriftproto

import (
	"context"
	"sync"

	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/henrylee2cn/goutil"
	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/codec"
	"github.com/henrylee2cn/teleport/utils"
)

const (
	// HeaderMeta the Meta key in header of thrift message
	HeaderMeta = "Tp-Meta"
	// HeaderBodyCodec the BodyCodec key in header of thrift message
	HeaderBodyCodec = "Tp-BodyCodec"
	// HeaderXferPipe the XferPipe key in header of thrift message
	HeaderXferPipe = "Tp-XferPipe"
)

func init() {
	tp.Printf("Setting thrift service method mapper and default thrift body codec...")
	tp.SetServiceMethodMapper(tp.RPCServiceMethodMapper)
	tp.SetDefaultBodyCodec(codec.ID_THRIFT)
}

// NewBinaryProtoFunc creates tp.ProtoFunc of Thrift protocol.
// NOTE:
//  Marshal the body into binary;
//  Support the Meta, BodyCodec and XferPipe.
func NewBinaryProtoFunc() tp.ProtoFunc {
	return func(rw tp.IOWithReadBuffer) tp.Proto {
		p := &tBinaryProto{
			id:        'b',
			name:      "thrift-binary",
			rwCounter: utils.NewReadWriteCounter(rw),
		}
		p.tProtocol = thrift.NewTHeaderProtocol(&BaseTTransport{
			ReadWriteCounter: p.rwCounter,
		})
		return p
	}
}

type tBinaryProto struct {
	id         byte
	name       string
	rwCounter  *utils.ReadWriteCounter
	tProtocol  *thrift.THeaderProtocol
	packLock   sync.Mutex
	unpackLock sync.Mutex
}

// Version returns the protocol's id and name.
func (t *tBinaryProto) Version() (byte, string) {
	return t.id, t.name
}

// Pack writes the Message into the connection.
// NOTE: Make sure to write only once or there will be package contamination!
func (t *tBinaryProto) Pack(m tp.Message) error {
	err := t.binaryPack(m)
	if err != nil {
		t.tProtocol.Transport().Close()
	}
	return err
}

func (t *tBinaryProto) Unpack(m tp.Message) error {
	err := t.binaryUnpack(m)
	if err != nil {
		t.tProtocol.Transport().Close()
	}
	return err
}

func (t *tBinaryProto) binaryPack(m tp.Message) error {
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

	t.packLock.Lock()
	defer t.packLock.Unlock()
	t.rwCounter.WriteCounter.Zero()

	if err := writeMessageBegin(t.tProtocol, m); err != nil {
		return err
	}

	if err = t.tProtocol.WriteBinary(bodyBytes); err != nil {
		return err
	}

	t.tProtocol.ClearWriteHeaders()
	t.tProtocol.SetWriteHeader(HeaderMeta, goutil.BytesToString(m.Meta().QueryString()))
	t.tProtocol.SetWriteHeader(HeaderBodyCodec, string(m.BodyCodec()))
	t.tProtocol.SetWriteHeader(HeaderXferPipe, goutil.BytesToString(m.XferPipe().IDs()))

	if err = t.tProtocol.WriteMessageEnd(); err != nil {
		return err
	}
	if err = t.tProtocol.Flush(m.Context()); err != nil {
		return err
	}

	return m.SetSize(uint32(t.rwCounter.Writed()))
}

func (t *tBinaryProto) binaryUnpack(m tp.Message) error {
	t.unpackLock.Lock()
	defer t.unpackLock.Unlock()
	t.rwCounter.WriteCounter.Zero()

	err := readMessageBegin(t.tProtocol, m)
	if err != nil {
		return err
	}

	bodyBytes, err := t.tProtocol.ReadBinary()
	if err != nil {
		return err
	}
	if err = t.tProtocol.ReadMessageEnd(); err != nil {
		return err
	}

	headers := t.tProtocol.GetReadHeaders()
	m.Meta().Parse(headers[HeaderMeta])
	if codecID := headers[HeaderBodyCodec]; codecID != "" {
		m.SetBodyCodec(byte(codecID[0]))
	}
	err = m.XferPipe().Append(goutil.StringToBytes(headers[HeaderXferPipe])...)
	if err != nil {
		return err
	}

	body, err := m.XferPipe().OnUnpack(bodyBytes)
	if err != nil {
		return err
	}
	err = m.UnmarshalBody(body)
	if err != nil {
		return err
	}

	return m.SetSize(uint32(t.rwCounter.Readed()))
}

// writeMessageBegin write a message header to the wire.
func writeMessageBegin(tProtocol thrift.TProtocol, m tp.Message) error {
	var typeID thrift.TMessageType
	switch m.Mtype() {
	case tp.TypeCall:
		typeID = thrift.CALL
	case tp.TypeReply:
		typeID = thrift.REPLY
	case tp.TypePush:
		typeID = thrift.ONEWAY
	}
	return tProtocol.WriteMessageBegin(m.ServiceMethod(), typeID, m.Seq())
}

// readMessageBegin read a message header.
func readMessageBegin(tProtocol thrift.TProtocol, m tp.Message) error {
	rMethod, rTypeID, rSeqID, err := tProtocol.ReadMessageBegin()
	if err != nil {
		return err
	}
	m.SetServiceMethod(rMethod)
	m.SetSeq(rSeqID)
	switch rTypeID {
	case thrift.CALL:
		m.SetMtype(tp.TypeCall)
	case thrift.REPLY:
		m.SetMtype(tp.TypeReply)
	case thrift.ONEWAY:
		m.SetMtype(tp.TypePush)
	case thrift.EXCEPTION:
		error0 := thrift.NewTApplicationException(thrift.UNKNOWN_APPLICATION_EXCEPTION, "Unknown Exception")
		err = error0.Read(tProtocol)
		if err != nil {
			return err
		}
		return error0
	default:
		m.SetMtype(tp.TypePush)
	}
	return nil
}

// BaseTTransport the base thrift transport
type BaseTTransport struct {
	*utils.ReadWriteCounter
}

var _ thrift.TTransport = new(BaseTTransport)

// Open opens the transport for communication.
func (*BaseTTransport) Open() error {
	return nil
}

// IsOpen returns true if the transport is open.
func (*BaseTTransport) IsOpen() bool {
	return true
}

// Close close the transport.
func (*BaseTTransport) Close() error {
	return nil
}

// Flush flushing a memory buffer is a no-op.
func (*BaseTTransport) Flush(context.Context) error {
	return nil
}

// RemainingBytes returns the number of remaining bytes.
func (*BaseTTransport) RemainingBytes() (numBytes uint64) {
	const maxSize = ^uint64(0)
	return maxSize // the thruth is, we just don't know unless framed is used
}
