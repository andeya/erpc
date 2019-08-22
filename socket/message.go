// Copyright 2017 HenryLee. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package socket

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sync"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/goutil/status"
	"github.com/henrylee2cn/teleport/codec"
	"github.com/henrylee2cn/teleport/utils"
	"github.com/henrylee2cn/teleport/xfer"
)

type (
	// Message a socket message interface.
	Message interface {
		// Reset resets and returns itself.
		Reset(settings ...MessageSetting) Message

		// Header is an operation interface of required message fields.
		// NOTE: Must be supported by Proto interface.
		Header

		// Body is an operation interface of optional message fields.
		// SUGGEST: For features complete, the protocol interface should support it.
		Body

		// XferPipe transfer filter pipe, handlers from outer-most to inner-most.
		// SUGGEST: The length can not be bigger than 255!
		XferPipe() *xfer.XferPipe

		// Size returns the size of message.
		// SUGGEST: For better statistics, Proto interfaces should support it.
		Size() uint32

		// SetSize sets the size of message.
		// If the size is too big, returns error.
		// SUGGEST: For better statistics, Proto interfaces should support it.
		SetSize(size uint32) error

		// Context returns the message handling context.
		Context() context.Context

		// String returns printing message information.
		String() string

		// AsHeader converts it to Header interface.
		AsHeader() Header
		// AsBody converts it to Body interface.
		AsBody() Body

		// messageIdentity prevents implementation outside the package.
		messageIdentity() *message
	}

	// Header is an operation interface of required message fields.
	// NOTE: Must be supported by Proto interface.
	Header interface {
		// Seq returns the message sequence.
		Seq() int32
		// SetSeq sets the message sequence.
		SetSeq(int32)
		// Mtype returns the message type, such as CALL, REPLY, PUSH.
		Mtype() byte
		// Mtype sets the message type, such as CALL, REPLY, PUSH.
		SetMtype(byte)
		// ServiceMethod returns the serviec method.
		// SUGGEST: max len ≤ 255!
		ServiceMethod() string
		// SetServiceMethod sets the serviec method.
		// SUGGEST: max len ≤ 255!
		SetServiceMethod(string)
		// StatusOK returns  the message status is OK or not.
		StatusOK() bool
		// Status returns the message status with code, msg, cause or stack.
		// NOTE:
		//  If it is nil and autoInit = true, assign a new object;
		//  Should use StatusOK to judge whether it is ok.
		Status(autoInit ...bool) *Status
		// SetStatus the message status.
		SetStatus(*Status)
		// Meta returns the metadata.
		// SUGGEST: urlencoded string max len ≤ 65535!
		Meta() *utils.Args
	}

	// Body is an operation interface of optional message fields.
	// SUGGEST: For features complete, the protocol interface should support it.
	Body interface {
		// BodyCodec returns the body codec type id.
		BodyCodec() byte
		// SetBodyCodec sets the body codec type id.
		SetBodyCodec(bodyCodec byte)
		// Body returns the body object.
		Body() interface{}
		// SetBody sets the body object.
		SetBody(body interface{})
		// SetNewBody resets the function of geting body.
		//  NOTE: NewBodyFunc is only for reading form connection;
		SetNewBody(NewBodyFunc)
		// MarshalBody returns the encoding of body.
		// NOTE: when the body is a stream of bytes, no marshalling is done.
		MarshalBody() ([]byte, error)
		// UnmarshalBody unmarshals the encoded data to the body.
		// NOTE:
		//  seq, mtype, uri must be setted already;
		//  if body=nil, try to use newBodyFunc to create a new one;
		//  when the body is a stream of bytes, no unmarshalling is done.
		UnmarshalBody(bodyBytes []byte) error
	}

	// NewBodyFunc creates a new body by header,
	// and only for reading form connection.
	NewBodyFunc func(Header) interface{}

	// Status a message status with code, msg, cause or stack.
	Status = status.Status
)

// message a socket message data.
type message struct {
	serviceMethod string
	status        *Status
	meta          *utils.Args
	body          interface{}
	newBodyFunc   NewBodyFunc
	xferPipe      *xfer.XferPipe
	ctx           context.Context
	size          uint32
	seq           int32
	mtype         byte
	bodyCodec     byte
}

var messagePool = sync.Pool{
	New: func() interface{} {
		return NewMessage()
	},
}

// GetMessage gets a *message form message pool.
// NOTE:
//  newBodyFunc is only for reading form connection;
//  settings are only for writing to connection.
func GetMessage(settings ...MessageSetting) Message {
	m := messagePool.Get().(*message)
	m.doSetting(settings...)
	return m
}

// PutMessage puts a *message to message pool.
func PutMessage(m Message) {
	m.Reset()
	messagePool.Put(m)
}

// NewMessage creates a new *message.
// NOTE:
//  NewBody is only for reading form connection;
//  settings are only for writing to connection.
func NewMessage(settings ...MessageSetting) Message {
	var m = &message{
		meta:     new(utils.Args),
		xferPipe: xfer.NewXferPipe(),
	}
	m.doSetting(settings...)
	return m
}

var (
	// NewStatus creates a message status with code, msg and cause.
	// NOTE:
	//  code=0 means no error
	// TYPE:
	//  func NewStatus(code int32, msg string, cause interface{}) *Status
	NewStatus = status.New

	// NewStatusWithStack creates a message status with code, msg and cause and stack.
	// NOTE:
	//  code=0 means no error
	// TYPE:
	//  func NewStatusWithStack(code int32, msg string, cause interface{}) *Status
	NewStatusWithStack = status.NewWithStack
)

// AsHeader converts it to Header interface.
func (m *message) AsHeader() Header { return m }

// AsBody converts it to Body interface.
func (m *message) AsBody() Body { return m }

// messageIdentity prevents implementation outside the package.
func (*message) messageIdentity() *message { return nil }

// Reset resets and returns itself.
// NOTE:
//  settings are only for writing to connection.
func (m *message) Reset(settings ...MessageSetting) Message {
	m.body = nil
	m.status = nil
	m.meta.Reset()
	m.xferPipe.Reset()
	m.newBodyFunc = nil
	m.seq = 0
	m.mtype = 0
	m.serviceMethod = ""
	m.size = 0
	m.ctx = nil
	m.bodyCodec = codec.NilCodecID
	m.doSetting(settings...)
	return m
}

func (m *message) doSetting(settings ...MessageSetting) {
	for _, fn := range settings {
		if fn != nil {
			fn(m)
		}
	}
}

// Context returns the message handling context.
func (m *message) Context() context.Context {
	if m.ctx == nil {
		return context.Background()
	}
	return m.ctx
}

// Seq returns the message sequence.
func (m *message) Seq() int32 {
	return m.seq
}

// SetSeq sets the message sequence.
func (m *message) SetSeq(seq int32) {
	m.seq = seq
}

// Mtype returns the message type, such as CALL, REPLY, PUSH.
func (m *message) Mtype() byte {
	return m.mtype
}

// Mtype sets the message type, such as CALL, REPLY, PUSH.
func (m *message) SetMtype(mtype byte) {
	m.mtype = mtype
}

// ServiceMethod returns the serviec method.
// SUGGEST: max len ≤ 255!
func (m *message) ServiceMethod() string {
	return m.serviceMethod
}

// SetServiceMethod sets the serviec method.
// SUGGEST: max len ≤ 255!
func (m *message) SetServiceMethod(serviceMethod string) {
	m.serviceMethod = serviceMethod
}

// StatusOK returns  the message status is OK or not.
func (m *message) StatusOK() bool {
	return m.status.OK()
}

// Status returns the message status with code, msg, cause or stack.
// NOTE:
//  If it is nil and autoInit = true, assign a new object;
//  Should use StatusOK to judge whether it is ok.
func (m *message) Status(autoInit ...bool) *Status {
	if m.status == nil && len(autoInit) > 0 && autoInit[0] {
		m.status = new(Status)
	}
	return m.status
}

// SetStatus the message status.
func (m *message) SetStatus(stat *Status) {
	m.status = stat
}

// Meta returns the metadata.
// When the package is reset, it will be reset.
// SUGGEST: urlencoded string max len ≤ 65535!
func (m *message) Meta() *utils.Args {
	return m.meta
}

// BodyCodec returns the body codec type id.
func (m *message) BodyCodec() byte {
	return m.bodyCodec
}

// SetBodyCodec sets the body codec type id.
func (m *message) SetBodyCodec(bodyCodec byte) {
	m.bodyCodec = bodyCodec
}

// Body returns the body object.
func (m *message) Body() interface{} {
	return m.body
}

// SetBody sets the body object.
func (m *message) SetBody(body interface{}) {
	m.body = body
}

// SetNewBody resets the function of geting body.
//  NOTE: newBodyFunc is only for reading form connection;
func (m *message) SetNewBody(newBodyFunc NewBodyFunc) {
	m.newBodyFunc = newBodyFunc
}

// MarshalBody returns the encoding of body.
// NOTE: when the body is a stream of bytes, no marshalling is done.
func (m *message) MarshalBody() ([]byte, error) {
	switch body := m.body.(type) {
	default:
		c, err := codec.Get(m.bodyCodec)
		if err != nil {
			return []byte{}, err
		}
		return c.Marshal(body)
	case nil:
		return []byte{}, nil
	case *[]byte:
		if body == nil {
			return []byte{}, nil
		}
		return *body, nil
	case []byte:
		return body, nil
	}
}

// UnmarshalBody unmarshals the encoded data to the body.
// NOTE:
//  seq, mtype, uri must be setted already;
//  if body=nil, try to use newBodyFunc to create a new one;
//  when the body is a stream of bytes, no unmarshalling is done.
func (m *message) UnmarshalBody(bodyBytes []byte) error {
	if m.body == nil && m.newBodyFunc != nil {
		m.body = m.newBodyFunc(m)
	}
	length := len(bodyBytes)
	if length == 0 {
		return nil
	}
	switch body := m.body.(type) {
	default:
		c, err := codec.Get(m.bodyCodec)
		if err != nil {
			return err
		}
		return c.Unmarshal(bodyBytes, m.body)
	case nil:
		return nil
	case *[]byte:
		if cap(*body) < length {
			*body = make([]byte, length)
		} else {
			*body = (*body)[:length]
		}
		copy(*body, bodyBytes)
		return nil
	}
}

// XferPipe returns transfer filter pipe, handlers from outer-most to inner-most.
// NOTE: the length can not be bigger than 255!
func (m *message) XferPipe() *xfer.XferPipe {
	return m.xferPipe
}

// Size returns the size of message.
// SUGGEST: For better statistics, Proto interfaces should support it.
func (m *message) Size() uint32 {
	return m.size
}

// SetSize sets the size of message.
// If the size is too big, returns error.
// SUGGEST: For better statistics, Proto interfaces should support it.
func (m *message) SetSize(size uint32) error {
	err := checkMessageSize(size)
	if err != nil {
		return err
	}
	m.size = size
	return nil
}

const messageFormat = `
{
  "seq": %d,
  "mtype": %d,
  "serviceMethod": %q,
  "status": %q,
  "meta": %q,
  "bodyCodec": %d,
  "body": %s,
  "xferPipe": %s,
  "size": %d
}`

// String returns printing message information.
func (m *message) String() string {
	var xferPipeIDs = make([]int, m.xferPipe.Len())
	for i, id := range m.xferPipe.IDs() {
		xferPipeIDs[i] = int(id)
	}
	idsBytes, _ := json.Marshal(xferPipeIDs)
	b, _ := json.Marshal(m.body)
	dst := bytes.NewBuffer(make([]byte, 0, len(b)*2))
	json.Indent(dst, goutil.StringToBytes(
		fmt.Sprintf(messageFormat,
			m.seq,
			m.mtype,
			m.serviceMethod,
			m.status.QueryString(),
			m.meta.QueryString(),
			m.bodyCodec,
			b,
			idsBytes,
			m.size,
		),
	), "", "  ")
	return goutil.BytesToString(dst.Bytes())
}

// MessageSetting is a pipe function type for setting message,
// and only for writing to connection.
type MessageSetting func(Message)

// WithNothing nothing to do.
func WithNothing() MessageSetting {
	return func(Message) {}
}

// WithContext sets the message handling context.
func WithContext(ctx context.Context) MessageSetting {
	return func(m Message) {
		m.(*message).ctx = ctx
	}
}

// WithServiceMethod sets the message service method.
// SUGGEST: max len ≤ 255!
func WithServiceMethod(serviceMethod string) MessageSetting {
	return func(m Message) {
		m.SetServiceMethod(serviceMethod)
	}
}

// WithStatus sets the message status.
func WithStatus(stat *Status) MessageSetting {
	return func(m Message) {
		m.SetStatus(stat)
	}
}

// WithAddMeta adds 'key=value' metadata argument.
// Multiple values for the same key may be added.
// SUGGEST: urlencoded string max len ≤ 65535!
func WithAddMeta(key, value string) MessageSetting {
	return func(m Message) {
		m.Meta().Add(key, value)
	}
}

// WithSetMeta sets 'key=value' metadata argument.
// SUGGEST: urlencoded string max len ≤ 65535!
func WithSetMeta(key, value string) MessageSetting {
	return func(m Message) {
		m.Meta().Set(key, value)
	}
}

// WithBodyCodec sets the body codec.
func WithBodyCodec(bodyCodec byte) MessageSetting {
	return func(m Message) {
		m.SetBodyCodec(bodyCodec)
	}
}

// WithBody sets the body object.
func WithBody(body interface{}) MessageSetting {
	return func(m Message) {
		m.SetBody(body)
	}
}

// WithNewBody resets the function of geting body.
//  NOTE: newBodyFunc is only for reading form connection.
func WithNewBody(newBodyFunc NewBodyFunc) MessageSetting {
	return func(m Message) {
		m.SetNewBody(newBodyFunc)
	}
}

// WithXferPipe sets transfer filter pipe.
// NOTE: Panic if the filterID is not registered.
// SUGGEST: The length can not be bigger than 255!
func WithXferPipe(filterID ...byte) MessageSetting {
	return func(m Message) {
		if err := m.XferPipe().Append(filterID...); err != nil {
			panic(err)
		}
	}
}

var (
	messageSizeLimit uint32 = math.MaxUint32
	// ErrExceedMessageSizeLimit error
	ErrExceedMessageSizeLimit = errors.New("Size of package exceeds limit")
)

// MessageSizeLimit gets the message size upper limit of reading.
func MessageSizeLimit() uint32 {
	return messageSizeLimit
}

// SetMessageSizeLimit sets max message size.
// If maxSize<=0, set it to max uint32.
func SetMessageSizeLimit(maxMessageSize uint32) {
	if maxMessageSize <= 0 {
		messageSizeLimit = math.MaxUint32
	} else {
		messageSizeLimit = maxMessageSize
	}
}

func checkMessageSize(messageSize uint32) error {
	if messageSize > messageSizeLimit {
		return ErrExceedMessageSizeLimit
	}
	return nil
}
