package thriftproto

import (
	"context"
	"sync"

	"git.apache.org/thrift.git/lib/go/thrift"
	tp "github.com/henrylee2cn/teleport"
	"github.com/henrylee2cn/teleport/proto/thriftproto/gen-go/payload"
)

func init() {
	tp.Printf("Setting thrift-style service method mapper...")
	tp.SetServiceMethodMapper(tp.RPCServiceMethodMapper)
}

// NewTProtoFactory creates tp.ProtoFunc of Thrift protocol.
// NOTE:
//  If @factory is not provided, use the default binary protocol.
func NewTProtoFactory(factory ...thrift.TProtocolFactory) tp.ProtoFunc {
	var fa thrift.TProtocolFactory
	if len(factory) > 0 {
		fa = factory[0]
	} else {
		fa = thrift.NewTBinaryProtocolFactoryDefault()
	}
	return func(rw tp.IOWithReadBuffer) tp.Proto {
		p := &thriftproto{
			id:   't',
			name: "thrift",
			rw:   rw,
		}
		p.ioprot = fa.GetProtocol(p)
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
	rw          tp.IOWithReadBuffer
	ioprot      thrift.TProtocol
	packLock    sync.Mutex
	unpackLock  sync.Mutex
	currReaded  int
	currWrited  int
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
	t.currWrited = 0
	if err := t.ioprot.WriteMessageBegin(m.ServiceMethod(), typeID, m.Seq()); err != nil {
		return err
	}

	if err = pd.Write(t.ioprot); err != nil {
		return err
	}

	if err = t.ioprot.WriteMessageEnd(); err != nil {
		return err
	}
	if err = t.ioprot.Flush(nil); err != nil {
		return err
	}

	// set size
	m.SetSize(uint32(t.currWrited))

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
	t.currWrited = 0
	rMethod, rTypeID, rSeqID, err := t.ioprot.ReadMessageBegin()
	if err != nil {
		return nil, err
	}

	err = m.SetSize(uint32(t.currWrited))
	if err != nil {
		return nil, err
	}

	pd := t.payloadPool.Get().(*payload.Payload)
	*pd = payload.Payload{}

	if err = pd.Read(t.ioprot); err != nil {
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
	if err = t.ioprot.ReadMessageEnd(); err != nil {
		t.payloadPool.Put(pd)
		return nil, err
	}
	return pd, nil
}

func (t *thriftproto) IsOpen() bool {
	return true
}

func (t *thriftproto) Open() error {
	return nil
}

func (t *thriftproto) Close() error {
	return nil
}

func (t *thriftproto) Read(p []byte) (int, error) {
	n, err := t.rw.Read(p)
	if err != nil {
		return 0, err
	}
	t.currReaded += n
	return n, err
}

func (t *thriftproto) Write(p []byte) (int, error) {
	n, err := t.rw.Write(p)
	if err != nil {
		return 0, err
	}
	t.currWrited += n
	return n, err
}

// Flushing a memory buffer is a no-op
func (t *thriftproto) Flush(context.Context) error {
	return nil
}

func (t *thriftproto) RemainingBytes() (num_bytes uint64) {
	const maxSize = ^uint64(0)
	return maxSize // the thruth is, we just don't know unless framed is used
}
