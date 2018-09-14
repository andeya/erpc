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
	"net/url"
	"sync"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/teleport/codec"
	"github.com/henrylee2cn/teleport/utils"
	"github.com/henrylee2cn/teleport/xfer"
)

type (
	// Message a socket message data.
	Message struct {
		// message sequence
		// NOTE: max len ≤ 65535!
		seq string
		// message type, such as CALL, PUSH, REPLY
		mtype byte
		// URI string
		// NOTE: max len ≤ 65535!
		uri string
		// URI object
		// NOTE: urlencoded URI max len ≤ 65535!
		uriObject *url.URL
		// metadata
		// NOTE: urlencoded string max len ≤ 65535!
		meta *utils.Args
		// body codec type
		bodyCodec byte
		// body object
		body interface{}
		// newBodyFunc creates a new body by message type and URI.
		// NOTE:
		//  only for writing message;
		//  should be nil when reading message.
		newBodyFunc NewBodyFunc
		// XferPipe transfer filter pipe, handlers from outer-most to inner-most.
		// NOTE: the length can not be bigger than 255!
		xferPipe *xfer.XferPipe
		// message size
		size uint32
		// ctx is the message handling context,
		// carries a deadline, a cancelation signal,
		// and other values across API boundaries.
		ctx context.Context
		// stack
		next *Message
	}
	// Header message header interface
	Header interface {
		// Mtype returns the message sequence
		Seq() string
		// SetSeq sets the message sequence
		// NOTE: max len ≤ 65535!
		SetSeq(string)
		// Mtype returns the message type, such as CALL, PUSH, REPLY
		Mtype() byte
		// Mtype sets the message type
		SetMtype(byte)
		// Uri returns the URI string
		Uri() string
		// UriObject returns the URI object
		UriObject() *url.URL
		// SetUri sets the message URI
		// NOTE: max len ≤ 65535!
		SetUri(string)
		// SetUriObject sets the message URI
		// NOTE: urlencoded URI max len ≤ 65535!
		SetUriObject(uriObject *url.URL)
		// Meta returns the metadata
		// NOTE: urlencoded string max len ≤ 65535!
		Meta() *utils.Args
	}
	// Body message body interface
	Body interface {
		// BodyCodec returns the body codec type id
		BodyCodec() byte
		// SetBodyCodec sets the body codec type id
		SetBodyCodec(bodyCodec byte)
		// Body returns the body object
		Body() interface{}
		// SetBody sets the body object
		SetBody(body interface{})
		// SetNewBody resets the function of geting body.
		SetNewBody(newBodyFunc NewBodyFunc)
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

	// NewBodyFunc creates a new body by header.
	NewBodyFunc func(Header) interface{}
)

var (
	_ Header = new(Message)
	_ Body   = new(Message)
)

var messageStack = new(struct {
	freeMessage *Message
	mu          sync.Mutex
})

// GetMessage gets a *Message form message stack.
// NOTE:
//  newBodyFunc is only for reading form connection;
//  settings are only for writing to connection.
func GetMessage(settings ...MessageSetting) *Message {
	messageStack.mu.Lock()
	m := messageStack.freeMessage
	if m == nil {
		m = NewMessage(settings...)
	} else {
		messageStack.freeMessage = m.next
		m.doSetting(settings...)
	}
	messageStack.mu.Unlock()
	return m
}

// PutMessage puts a *Message to message stack.
func PutMessage(m *Message) {
	messageStack.mu.Lock()
	m.Reset()
	m.next = messageStack.freeMessage
	messageStack.freeMessage = m
	messageStack.mu.Unlock()
}

// NewMessage creates a new *Message.
// NOTE:
//  NewBody is only for reading form connection;
//  settings are only for writing to connection.
func NewMessage(settings ...MessageSetting) *Message {
	var m = &Message{
		meta:     new(utils.Args),
		xferPipe: xfer.NewXferPipe(),
	}
	m.doSetting(settings...)
	return m
}

// Reset resets itself.
// NOTE:
//  newBodyFunc is only for reading form connection;
//  settings are only for writing to connection.
func (m *Message) Reset(settings ...MessageSetting) {
	m.next = nil
	m.body = nil
	m.meta.Reset()
	m.xferPipe.Reset()
	m.newBodyFunc = nil
	m.seq = ""
	m.mtype = 0
	m.uri = ""
	m.uriObject = nil
	m.size = 0
	m.ctx = nil
	m.bodyCodec = codec.NilCodecId
	m.doSetting(settings...)
}

func (m *Message) doSetting(settings ...MessageSetting) {
	for _, fn := range settings {
		if fn != nil {
			fn(m)
		}
	}
}

// Context returns the message handling context.
func (m *Message) Context() context.Context {
	if m.ctx == nil {
		return context.Background()
	}
	return m.ctx
}

// Seq returns the message sequence
func (m *Message) Seq() string {
	return m.seq
}

// SetSeq sets the message sequence
// NOTE: max len ≤ 65535!
func (m *Message) SetSeq(seq string) {
	m.seq = seq
}

// Mtype returns the message type, such as CALL, PUSH, REPLY
func (m *Message) Mtype() byte {
	return m.mtype
}

// SetMtype sets the message type
func (m *Message) SetMtype(mtype byte) {
	m.mtype = mtype
}

// Uri returns the URI string
func (m *Message) Uri() string {
	if m.uriObject != nil {
		return m.uriObject.String()
	}
	return m.uri
}

// UriObject returns the URI object
func (m *Message) UriObject() *url.URL {
	if m.uriObject == nil {
		m.uriObject, _ = url.Parse(m.uri)
		if m.uriObject == nil {
			m.uriObject = new(url.URL)
		}
		m.uri = ""
	}
	return m.uriObject
}

// SetUri sets the message URI
// NOTE: max len ≤ 65535!
func (m *Message) SetUri(uri string) {
	m.uri = uri
	m.uriObject = nil
}

// SetUriObject sets the message URI
// NOTE: urlencoded URI max len ≤ 65535!
func (m *Message) SetUriObject(uriObject *url.URL) {
	m.uriObject = uriObject
	m.uri = ""
}

// Meta returns the metadata.
// When the package is reset, it will be reset.
// NOTE: urlencoded string max len ≤ 65535!
func (m *Message) Meta() *utils.Args {
	return m.meta
}

// BodyCodec returns the body codec type id
func (m *Message) BodyCodec() byte {
	return m.bodyCodec
}

// SetBodyCodec sets the body codec type id
func (m *Message) SetBodyCodec(bodyCodec byte) {
	m.bodyCodec = bodyCodec
}

// Body returns the body object
func (m *Message) Body() interface{} {
	return m.body
}

// SetBody sets the body object
func (m *Message) SetBody(body interface{}) {
	m.body = body
}

// SetNewBody resets the function of geting body.
func (m *Message) SetNewBody(newBodyFunc NewBodyFunc) {
	m.newBodyFunc = newBodyFunc
}

// MarshalBody returns the encoding of body.
// NOTE: when the body is a stream of bytes, no marshalling is done.
func (m *Message) MarshalBody() ([]byte, error) {
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
func (m *Message) UnmarshalBody(bodyBytes []byte) error {
	if m.body == nil && m.newBodyFunc != nil {
		m.body = m.newBodyFunc(m)
	}
	if len(bodyBytes) == 0 {
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
		if body != nil {
			*body = make([]byte, len(bodyBytes))
			copy(*body, bodyBytes)
		}
		return nil
	}
}

// XferPipe returns transfer filter pipe, handlers from outer-most to inner-most.
// NOTE: the length can not be bigger than 255!
func (m *Message) XferPipe() *xfer.XferPipe {
	return m.xferPipe
}

// Size returns the size of message.
func (m *Message) Size() uint32 {
	return m.size
}

// SetSize sets the size of message.
// If the size is too big, returns error.
func (m *Message) SetSize(size uint32) error {
	err := checkMessageSize(size)
	if err != nil {
		return err
	}
	m.size = size
	return nil
}

const messageFormat = `
{
  "seq": %q,
  "mtype": %d,
  "uri": %q,
  "meta": %q,
  "body_codec": %d,
  "body": %s,
  "xfer_pipe": %s,
  "size": %d
}`

// String returns printing text.
func (m *Message) String() string {
	var xferPipeIds = make([]int, m.xferPipe.Len())
	for i, id := range m.xferPipe.Ids() {
		xferPipeIds[i] = int(id)
	}
	idsBytes, _ := json.Marshal(xferPipeIds)
	b, _ := json.Marshal(m.body)
	dst := bytes.NewBuffer(make([]byte, 0, len(b)*2))
	json.Indent(dst, goutil.StringToBytes(
		fmt.Sprintf(messageFormat,
			m.seq,
			m.mtype,
			m.uri,
			m.meta.QueryString(),
			m.bodyCodec,
			b,
			idsBytes,
			m.size,
		),
	), "", "  ")
	return goutil.BytesToString(dst.Bytes())
}

// MessageSetting is a pipe function type for setting message.
type MessageSetting func(*Message)

// WithContext sets the message handling context.
func WithContext(ctx context.Context) MessageSetting {
	return func(m *Message) {
		m.ctx = ctx
	}
}

// WithSeq sets the message sequence.
// NOTE: max len ≤ 65535!
func WithSeq(seq string) MessageSetting {
	return func(m *Message) {
		m.seq = seq
	}
}

// WithMtype sets the message type.
func WithMtype(mtype byte) MessageSetting {
	return func(m *Message) {
		m.mtype = mtype
	}
}

// WithUri sets the message URI string.
// NOTE: max len ≤ 65535!
func WithUri(uri string) MessageSetting {
	return func(m *Message) {
		m.SetUri(uri)
	}
}

// WithUriObject sets the message URI object.
// NOTE: urlencoded URI max len ≤ 65535!
func WithUriObject(uriObject *url.URL) MessageSetting {
	return func(m *Message) {
		m.SetUriObject(uriObject)
	}
}

// WithQuery sets the message URI query parameter.
// NOTE: urlencoded URI max len ≤ 65535!
func WithQuery(key, value string) MessageSetting {
	return func(m *Message) {
		u := m.UriObject()
		v := u.Query()
		v.Add(key, value)
		u.RawQuery = v.Encode()
	}
}

// WithAddMeta adds 'key=value' metadata argument.
// Multiple values for the same key may be added.
// NOTE: urlencoded string max len ≤ 65535!
func WithAddMeta(key, value string) MessageSetting {
	return func(m *Message) {
		m.meta.Add(key, value)
	}
}

// WithSetMeta sets 'key=value' metadata argument.
// NOTE: urlencoded string max len ≤ 65535!
func WithSetMeta(key, value string) MessageSetting {
	return func(m *Message) {
		m.meta.Set(key, value)
	}
}

// WithBodyCodec sets the body codec.
func WithBodyCodec(bodyCodec byte) MessageSetting {
	return func(m *Message) {
		m.bodyCodec = bodyCodec
	}
}

// WithBody sets the body object.
func WithBody(body interface{}) MessageSetting {
	return func(m *Message) {
		m.body = body
	}
}

// WithNewBody resets the function of geting body.
func WithNewBody(newBodyFunc NewBodyFunc) MessageSetting {
	return func(m *Message) {
		m.newBodyFunc = newBodyFunc
	}
}

// WithXferPipe sets transfer filter pipe.
// NOTE:
//  panic if the filterId is not registered
func WithXferPipe(filterId ...byte) MessageSetting {
	return func(m *Message) {
		if err := m.xferPipe.Append(filterId...); err != nil {
			panic(err)
		}
	}
}

var (
	messageSizeLimit uint32 = math.MaxUint32
	// ErrExceedMessageSizeLimit error
	ErrExceedMessageSizeLimit = errors.New("Size of package exceeds limit.")
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
