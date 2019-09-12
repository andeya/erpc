package tp

import (
	"strconv"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/teleport/codec"
	"github.com/henrylee2cn/teleport/socket"
	"github.com/henrylee2cn/teleport/utils"
)

// Message types
const (
	TypeUndefined byte = 0
	TypeCall      byte = 1
	TypeReply     byte = 2 // reply to call
	TypePush      byte = 3
	TypeAuthCall  byte = 4
	TypeAuthReply byte = 5
)

// TypeText returns the message type text.
// If the type is undefined returns 'Undefined'.
func TypeText(typ byte) string {
	switch typ {
	case TypeCall:
		return "CALL"
	case TypeReply:
		return "REPLY"
	case TypePush:
		return "PUSH"
	case TypeAuthCall:
		return "AUTH_CALL"
	case TypeAuthReply:
		return "AUTH_REPLY"
	default:
		return "Undefined"
	}
}

type (
	// Socket is a generic stream-oriented network connection.
	// NOTE:
	//  Multiple goroutines may invoke methods on a Socket simultaneously.
	Socket = socket.Socket
	// Proto pack/unpack protocol scheme of socket message.
	Proto = socket.Proto
	// ProtoFunc function used to create a custom Proto interface.
	ProtoFunc = socket.ProtoFunc
	// IOWithReadBuffer implements buffered I/O with buffered reader.
	IOWithReadBuffer = socket.IOWithReadBuffer
)

type (
	// Message a socket message data.
	Message = socket.Message
	// Header message header interface
	Header = socket.Header
	// Body message body interface
	Body = socket.Body
	// NewBodyFunc creates a new body by header.
	NewBodyFunc = socket.NewBodyFunc
	// MessageSetting is a pipe function type for setting message.
	MessageSetting = socket.MessageSetting
)

const (
	// MetaRealIP real IP metadata key
	MetaRealIP = "X-Real-IP"
	// MetaAcceptBodyCodec the key of body codec that the sender wishes to accept
	MetaAcceptBodyCodec = "X-Accept-Body-Codec"
)

var (
	// GetMessage gets a Message form message pool.
	// NOTE:
	//  newBodyFunc is only for reading form connection;
	//  settings are only for writing to connection.
	//  func GetMessage(settings ...MessageSetting) Message
	GetMessage = socket.GetMessage
	// PutMessage puts a Message to message pool.
	//  func PutMessage(m Message)
	PutMessage = socket.PutMessage
)

var (
	// WithNothing nothing to do.
	//  func WithNothing() MessageSetting
	WithNothing = socket.WithNothing
	// WithStatus sets the message status.
	// TYPE:
	//  func WithStatus(stat *Status) MessageSetting
	WithStatus = socket.WithStatus
	// WithContext sets the message handling context.
	//  func WithContext(ctx context.Context) MessageSetting
	WithContext = socket.WithContext
	// WithServiceMethod sets the message service method.
	// SUGGEST: max len ≤ 255!
	//  func WithServiceMethod(serviceMethod string) MessageSetting
	WithServiceMethod = socket.WithServiceMethod
	// WithAddMeta adds 'key=value' metadata argument.
	// Multiple values for the same key may be added.
	// SUGGEST: urlencoded string max len ≤ 65535!
	//  func WithAddMeta(key, value string) MessageSetting
	WithAddMeta = socket.WithAddMeta
	// WithSetMeta sets 'key=value' metadata argument.
	// SUGGEST: urlencoded string max len ≤ 65535!
	//  func WithSetMeta(key, value string) MessageSetting
	WithSetMeta = socket.WithSetMeta
	// WithDelMeta deletes metadata argument.
	//   func WithDelMeta(key string) MessageSetting
	WithDelMeta = socket.WithDelMeta
	// WithBodyCodec sets the body codec.
	//  func WithBodyCodec(bodyCodec byte) MessageSetting
	WithBodyCodec = socket.WithBodyCodec
	// WithBody sets the body object.
	//  func WithBody(body interface{}) MessageSetting
	WithBody = socket.WithBody
	// WithNewBody resets the function of geting body.
	//  NOTE: newBodyFunc is only for reading form connection.
	//  func WithNewBody(newBodyFunc socket.NewBodyFunc) MessageSetting
	WithNewBody = socket.WithNewBody
	// WithXferPipe sets transfer filter pipe.
	// NOTE: Panic if the filterID is not registered.
	// SUGGEST: The length can not be bigger than 255!
	//  func WithXferPipe(filterID ...byte) MessageSetting
	WithXferPipe = socket.WithXferPipe
)

// WithRealIP sets the real IP to metadata.
func WithRealIP(ip string) MessageSetting {
	return socket.WithAddMeta(MetaRealIP, ip)
}

// WithAcceptBodyCodec sets the body codec that the sender wishes to accept.
// NOTE: If the specified codec is invalid, the receiver will ignore the mate data.
func WithAcceptBodyCodec(bodyCodec byte) MessageSetting {
	if bodyCodec == codec.NilCodecID {
		return WithNothing()
	}
	return socket.WithAddMeta(MetaAcceptBodyCodec, strconv.FormatUint(uint64(bodyCodec), 10))
}

// withMtype sets the message type.
func withMtype(mtype byte) MessageSetting {
	return func(m Message) {
		m.SetMtype(mtype)
	}
}

// GetAcceptBodyCodec gets the body codec that the sender wishes to accept.
// NOTE: If the specified codec is invalid, the receiver will ignore the mate data.
func GetAcceptBodyCodec(meta *utils.Args) (byte, bool) {
	s := meta.Peek(MetaAcceptBodyCodec)
	if len(s) == 0 || len(s) > 3 {
		return 0, false
	}
	b, err := strconv.ParseUint(goutil.BytesToString(s), 10, 8)
	if err != nil {
		return 0, false
	}
	c := byte(b)
	return c, c != codec.NilCodecID
}
