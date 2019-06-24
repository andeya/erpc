package thriftproto

import (
	"context"
	"sync"

	"git.apache.org/thrift.git/lib/go/thrift"
	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/proto/thriftproto/gen-go/payload"
	"github.com/henrylee2cn/teleport/utils"
)

func init() {
	tp.Printf("Setting thrift-style service method mapper...")
	tp.SetServiceMethodMapper(tp.RPCServiceMethodMapper)
}

// NewTProtoFunc creates tp.ProtoFunc of Thrift protocol.
// NOTE:
//  If @protoFactory is not provided, use the default binary protocol.
func NewTProtoFunc(transFactory thrift.TTransportFactory, protoFactory thrift.TProtocolFactory) tp.ProtoFunc {
	if protoFactory == nil {
		protoFactory = thrift.NewTBinaryProtocolFactoryDefault()
	}
	return func(rw tp.IOWithReadBuffer) tp.Proto {
		p := &thriftproto{
			id:        't',
			name:      "thrift",
			rwCounter: utils.NewReadWriteCounter(rw),
		}
		var tTransport thrift.TTransport = &BaseTTransport{
			ReadWriteCounter: p.rwCounter,
		}
		if transFactory != nil {
			t, err := transFactory.GetTransport(tTransport)
			if err != nil {
				tp.Errorf("still using the base transport because it failed to wrap it: %s", err.Error())
			} else {
				tTransport = t
			}
		}
		p.tProtocol = protoFactory.GetProtocol(tTransport)
		p.payloadPool = sync.Pool{
			New: func() interface{} {
				return payload.NewPayload()
			},
		}
		return p
	}
}

type thriftproto struct {
	id          byte
	name        string
	rwCounter   *utils.ReadWriteCounter
	tProtocol   thrift.TProtocol
	packLock    sync.Mutex
	unpackLock  sync.Mutex
	payloadPool sync.Pool
}

// Version returns the protocol's id and name.
func (t *thriftproto) Version() (byte, string) {
	return t.id, t.name
}

// Pack writes the Message into the connection.
// NOTE: Make sure to write only once or there will be package contamination!
func (t *thriftproto) Pack(m tp.Message) error {
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

	pd := t.payloadPool.Get().(*payload.Payload)
	*pd = payload.Payload{}
	defer t.payloadPool.Put(pd)

	var typeID thrift.TMessageType
	switch m.Mtype() {
	case tp.TypeCall:
		typeID = thrift.CALL
	case tp.TypeReply:
		typeID = thrift.REPLY
	case tp.TypePush:
		typeID = thrift.ONEWAY
	}
	pd.Meta = m.Meta().QueryString()
	pd.BodyCodec = int32(m.BodyCodec())
	pd.XferPipe = m.XferPipe().IDs()
	pd.Body = bodyBytes

	t.packLock.Lock()
	defer t.packLock.Unlock()

	// pack
	t.rwCounter.WriteCounter.Zero()
	if err := t.tProtocol.WriteMessageBegin(m.ServiceMethod(), typeID, m.Seq()); err != nil {
		return err
	}

	if err = pd.Write(t.tProtocol); err != nil {
		return err
	}

	if err = t.tProtocol.WriteMessageEnd(); err != nil {
		return err
	}
	if err = t.tProtocol.Flush(nil); err != nil {
		return err
	}

	// set size
	m.SetSize(uint32(t.rwCounter.Writed()))

	return nil
}

func (t *thriftproto) Unpack(m tp.Message) error {
	pd, err := t.unpack(m)
	if err != nil {
		return err
	}
	defer t.payloadPool.Put(pd)

	err = m.XferPipe().Append(pd.XferPipe...)
	if err != nil {
		return err
	}
	body, err := m.XferPipe().OnUnpack(pd.Body)
	if err != nil {
		return err
	}
	m.Meta().ParseBytes(pd.Meta)
	m.SetBodyCodec(byte(pd.BodyCodec))
	return m.UnmarshalBody(body)
}

func (t *thriftproto) unpack(m tp.Message) (*payload.Payload, error) {
	t.unpackLock.Lock()
	defer t.unpackLock.Unlock()
	t.rwCounter.WriteCounter.Zero()
	rMethod, rTypeID, rSeqID, err := t.tProtocol.ReadMessageBegin()
	if err != nil {
		return nil, err
	}

	err = m.SetSize(uint32(t.rwCounter.Writed()))
	if err != nil {
		return nil, err
	}

	pd := t.payloadPool.Get().(*payload.Payload)
	*pd = payload.Payload{}

	if err = pd.Read(t.tProtocol); err != nil {
		t.payloadPool.Put(pd)
		return nil, err
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
	default:
		m.SetMtype(tp.TypePush)
	}
	if err = t.tProtocol.ReadMessageEnd(); err != nil {
		t.payloadPool.Put(pd)
		return nil, err
	}
	return pd, nil
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
