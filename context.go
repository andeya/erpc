// Copyright 2015-2017 HenryLee. All Rights Reserved.
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

package tp

import (
	"net/url"
	"reflect"
	"time"

	"github.com/henrylee2cn/goutil"

	"github.com/henrylee2cn/teleport/codec"
	"github.com/henrylee2cn/teleport/socket"
)

type (
	// PushCtx push handler context.
	// For example:
	//  type HomePush struct{ PushCtx }
	PushCtx interface {
		Seq() uint64
		GetBodyCodec() byte
		GetMeta(key string) []byte
		Uri() string
		Path() string
		RawQuery() string
		Query() url.Values
		Public() goutil.Map
		PublicLen() int
		Ip() string
		Peer() *Peer
		Session() Session
	}
	// PullCtx request handler context.
	// For example:
	//  type HomePull struct{ PullCtx }
	PullCtx interface {
		PushCtx
		SetBodyCodec(byte)
		SetMeta(key, value string)
		AddXferPipe(filterId ...byte)
	}
	UnknownPushCtx interface {
		PushCtx
		InputBodyBytes() []byte
		Bind(v interface{}) (bodyCodec byte, err error)
	}
	UnknownPullCtx interface {
		UnknownPushCtx
		AddXferPipe(filterId ...byte)
	}
	// WriteCtx for writing packet.
	WriteCtx interface {
		Output() *socket.Packet
		Public() goutil.Map
		PublicLen() int
		Ip() string
		Peer() *Peer
		Session() Session
	}
	// ReadCtx for reading packet.
	ReadCtx interface {
		Input() *socket.Packet
		Public() goutil.Map
		PublicLen() int
		Ip() string
		Peer() *Peer
		Session() Session
	}
	// readHandleCtx the underlying common instance of PullCtx and PushCtx.
	readHandleCtx struct {
		sess            *session
		input           *socket.Packet
		output          *socket.Packet
		apiType         *Handler
		arg             reflect.Value
		pullCmd         *PullCmd
		uri             *url.URL
		query           url.Values
		public          goutil.Map
		start           time.Time
		cost            time.Duration
		pluginContainer PluginContainer
		next            *readHandleCtx
	}
)

var (
	_ PullCtx        = new(readHandleCtx)
	_ PushCtx        = new(readHandleCtx)
	_ WriteCtx       = new(readHandleCtx)
	_ ReadCtx        = new(readHandleCtx)
	_ UnknownPullCtx = new(readHandleCtx)
	_ UnknownPushCtx = new(readHandleCtx)
)

var (
	emptyValue  = reflect.Value{}
	emptyMethod = reflect.Method{}
)

// newReadHandleCtx creates a readHandleCtx for one request/response or push.
func newReadHandleCtx() *readHandleCtx {
	c := new(readHandleCtx)
	c.input = socket.NewPacket()
	c.input.SetNewBody(c.binding)
	c.output = socket.NewPacket()
	return c
}

func (c *readHandleCtx) reInit(s *session) {
	c.sess = s
	count := s.socket.PublicLen()
	c.public = goutil.RwMap(count)
	if count > 0 {
		s.socket.Public().Range(func(key, value interface{}) bool {
			c.public.Store(key, value)
			return true
		})
	}
}

func (c *readHandleCtx) clean() {
	c.sess = nil
	c.apiType = nil
	c.arg = emptyValue
	c.pullCmd = nil
	c.public = nil
	c.uri = nil
	c.query = nil
	c.cost = 0
	c.pluginContainer = nil
	c.input.Reset(socket.WithNewBody(c.binding))
	c.output.Reset()
}

// Peer returns the peer.
func (c *readHandleCtx) Peer() *Peer {
	return c.sess.peer
}

// Session returns the session.
func (c *readHandleCtx) Session() Session {
	return c.sess
}

// Input returns readed packet.
func (c *readHandleCtx) Input() *socket.Packet {
	return c.input
}

// Output returns writed packet.
func (c *readHandleCtx) Output() *socket.Packet {
	return c.output
}

// Public returns temporary public data of context.
func (c *readHandleCtx) Public() goutil.Map {
	return c.public
}

// PublicLen returns the length of public data of context.
func (c *readHandleCtx) PublicLen() int {
	return c.public.Len()
}

// Seq returns the input packet sequence.
func (c *readHandleCtx) Seq() uint64 {
	return c.input.Seq()
}

// Uri returns the input packet uri.
func (c *readHandleCtx) Uri() string {
	return c.input.Uri()
}

// Path returns the input packet uri path.
func (c *readHandleCtx) Path() string {
	return c.uri.Path
}

// RawQuery returns the input packet uri query string.
func (c *readHandleCtx) RawQuery() string {
	return c.uri.RawQuery
}

// Query returns the input packet uri query object.
func (c *readHandleCtx) Query() url.Values {
	if c.query == nil {
		c.query = c.uri.Query()
	}
	return c.query
}

// GetMeta gets the header metadata for the input packet.
func (c *readHandleCtx) GetMeta(key string) []byte {
	return c.input.Meta().Peek(key)
}

// SetMeta sets the header metadata for reply packet.
func (c *readHandleCtx) SetMeta(key, value string) {
	c.output.Meta().Set(key, value)
}

// GetBodyCodec gets the body codec type of the input packet.
func (c *readHandleCtx) GetBodyCodec() byte {
	return c.input.BodyCodec()
}

// SetBodyCodec sets the body codec for reply packet.
func (c *readHandleCtx) SetBodyCodec(bodyCodec byte) {
	c.output.SetBodyCodec(bodyCodec)
}

// AddXferPipe appends transfer filter pipe of reply packet.
func (c *readHandleCtx) AddXferPipe(filterId ...byte) {
	c.output.XferPipe().Append(filterId...)
}

// Ip returns the remote addr.
func (c *readHandleCtx) Ip() string {
	return c.sess.RemoteIp()
}

// Be executed synchronously when reading packet
func (c *readHandleCtx) binding(header socket.Header) (body interface{}) {
	c.start = c.Peer().timeNow()
	c.pluginContainer = c.sess.peer.pluginContainer
	switch header.Ptype() {
	case TypeReply:
		return c.bindReply(header.Seq(), header.Uri())

	case TypePush:
		return c.bindPush(header.Uri())

	case TypePull:
		return c.bindPull(header.Seq(), header.Uri())

	default:
		return nil
	}
}

// Be executed asynchronously after readed packet
func (c *readHandleCtx) handle() {
	switch c.input.Ptype() {
	case TypeReply:
		// handles pull reply
		c.handleReply()

	case TypePush:
		//  handles push
		c.handlePush()

	case TypePull:
		// handles and replies pull
		c.handlePull()

	default:
		// if unsupported, disconnected.
		notImplemented_metaSetting.Inject(c.output.Meta())
		if c.sess.peer.printBody {
			logformat := "disconnect(%s) due to unsupported type: %d |\nseq: %d |uri: %-30s |\nRECV:\n size: %d\n body[-json]: %s\n"
			Errorf(logformat, c.Ip(), c.input.Ptype(), c.input.Seq(), c.input.Uri(), c.input.Size(), bodyLogBytes(c.input))
		} else {
			logformat := "disconnect(%s) due to unsupported type: %d |\nseq: %d |uri: %-30s |\nRECV:\n size: %d\n"
			Errorf(logformat, c.Ip(), c.input.Ptype(), c.input.Seq(), c.input.Uri(), c.input.Size())
		}
		go c.sess.Close()
	}
}

func (c *readHandleCtx) bindPush(uri string) interface{} {
	if c.pluginContainer.PostReadPushHeader(c) != nil {
		return nil
	}
	var err error
	c.uri, err = url.Parse(uri)
	if err != nil {
		return nil
	}

	var ok bool
	c.apiType, ok = c.sess.pushRouter.get(c.Path())
	if !ok {
		return nil
	}

	c.pluginContainer = c.apiType.pluginContainer
	c.arg = reflect.New(c.apiType.argElem)
	c.input.SetBody(c.arg.Interface())

	if c.pluginContainer.PreReadPushBody(c) != nil {
		return nil
	}

	return c.input.Body()
}

// handlePush handles push.
func (c *readHandleCtx) handlePush() {
	defer func() {
		c.cost = c.Peer().timeSince(c.start)
		c.sess.runlog(c.cost, c.input, nil, typePushHandle)
	}()

	if c.apiType != nil {
		if c.pluginContainer.PostReadPushBody(c) == nil {
			if c.apiType.isUnknown {
				c.apiType.unknownHandleFunc(c)
			} else {
				c.apiType.handleFunc(c, c.arg)
			}
		}
	}
}

func (c *readHandleCtx) bindPull(seq uint64, uri string) interface{} {
	c.output.SetSeq(seq)
	c.output.SetUri(uri)
	rerr := c.pluginContainer.PostReadPullHeader(c)
	if rerr != nil {
		rerr.SetToMeta(c.output.Meta())
		return nil
	}
	var err error
	c.uri, err = url.Parse(uri)
	if err != nil {
		badPacket_metaSetting.Inject(c.output.Meta(), err.Error())
		return nil
	}

	var ok bool
	c.apiType, ok = c.sess.pullRouter.get(c.Path())
	if !ok {
		notFound_metaSetting.Inject(c.output.Meta())
		return nil
	}

	c.pluginContainer = c.apiType.pluginContainer
	if c.apiType.isUnknown {
		c.input.SetBody(new([]byte))
	} else {
		c.arg = reflect.New(c.apiType.argElem)
		c.input.SetBody(c.arg.Interface())
	}

	rerr = c.pluginContainer.PreReadPullBody(c)
	if rerr != nil {
		rerr.SetToMeta(c.output.Meta())
		return nil
	}

	return c.input.Body()
}

// handlePull handles and replies pull.
func (c *readHandleCtx) handlePull() {
	defer func() {
		c.cost = c.Peer().timeSince(c.start)
		c.sess.runlog(c.cost, c.input, c.output, typePullHandle)
	}()

	// set packet type
	c.output.SetPtype(TypeReply)

	// handle pull
	if !hasRerror(c.output.Meta()) {
		rerr := c.pluginContainer.PostReadPullBody(c)
		if rerr != nil {
			rerr.SetToMeta(c.output.Meta())
		} else {
			if c.apiType.isUnknown {
				c.apiType.unknownHandleFunc(c)
			} else {
				c.apiType.handleFunc(c, c.arg)
			}
		}
	}

	// reply pull
	c.pluginContainer.PreWriteReply(c)

	if err := c.sess.write(c.output); err != nil {
		writeFailed_metaSetting.Inject(c.output.Meta(), err.Error())
	}

	c.pluginContainer.PostWriteReply(c)
}

func (c *readHandleCtx) bindReply(seq uint64, uri string) interface{} {
	pullCmd, ok := c.sess.pullCmdMap.Load(seq)
	if !ok {
		Debugf("bindReply() not found: %v", c.input)
		return nil
	}
	c.pullCmd = pullCmd.(*PullCmd)
	c.public = c.pullCmd.public

	rerr := c.pluginContainer.PostReadReplyHeader(c)
	if rerr != nil {
		c.pullCmd.rerr = rerr
		return nil
	}
	rerr = c.pluginContainer.PreReadReplyBody(c)
	if rerr != nil {
		c.pullCmd.rerr = rerr
		return nil
	}
	return c.pullCmd.reply
}

// handleReply handles pull reply.
func (c *readHandleCtx) handleReply() {
	if c.pullCmd == nil {
		Debugf("c.pullCmd == nil:\npacket:%v\n", c.input)
		return
	}
	defer func() {
		c.pullCmd.done()
		c.pullCmd.cost = c.Peer().timeSince(c.pullCmd.start)
		c.sess.runlog(c.pullCmd.cost, c.input, c.pullCmd.output, typePullLaunch)
	}()
	if c.pullCmd.rerr != nil {
		return
	}
	rerr := NewRerrorFromMeta(c.input.Meta())
	if rerr == nil {
		rerr = c.pluginContainer.PostReadReplyBody(c)
	}
	c.pullCmd.rerr = rerr
}

// InputBodyBytes if the input body binder is []byte type, returns it, else returns nil.
func (c *readHandleCtx) InputBodyBytes() []byte {
	b, ok := c.input.Body().(*[]byte)
	if !ok {
		return nil
	}
	return *b
}

// Bind when the raw body binder is []byte type, now binds the input body to v.
func (c *readHandleCtx) Bind(v interface{}) (byte, error) {
	b := c.InputBodyBytes()
	if b == nil {
		return codec.NilCodecId, nil
	}
	c.input.SetBody(v)
	err := c.input.UnmarshalBody(b)
	return c.input.BodyCodec(), err
}

// PullCmd the command of the pulling operation's response.
type PullCmd struct {
	sess     *session
	output   *socket.Packet
	reply    interface{}
	rerr     *Rerror
	doneChan chan *PullCmd // Strobes when pull is complete.
	start    time.Time
	cost     time.Duration
	public   goutil.Map
}

var _ WriteCtx = new(PullCmd)

// Peer returns the peer.
func (c *PullCmd) Peer() *Peer {
	return c.sess.peer
}

// Session returns the session.
func (c *PullCmd) Session() Session {
	return c.sess
}

// Ip returns the remote addr.
func (c *PullCmd) Ip() string {
	return c.sess.RemoteIp()
}

// Public returns temporary public data of context.
func (c *PullCmd) Public() goutil.Map {
	return c.public
}

// PublicLen returns the length of public data of context.
func (c *PullCmd) PublicLen() int {
	return c.public.Len()
}

// Output returns writed packet.
func (c *PullCmd) Output() *socket.Packet {
	return c.output
}

// Result returns the pull result.
func (c *PullCmd) Result() (interface{}, *Rerror) {
	return c.reply, c.rerr
}

// *Rerror returns the pull error.
func (c *PullCmd) Rerror() *Rerror {
	return c.rerr
}

// CostTime returns the pulled cost time.
// If PeerConfig.CountTime=false, always returns 0.
func (c *PullCmd) CostTime() time.Duration {
	return c.cost
}

func (p *PullCmd) cancel() {
	p.rerr = rerror_connClosed.Copy()
	p.doneChan <- p
	p.sess.pullCmdMap.Delete(p.output.Seq())
	{
		// free count pull-launch
		p.sess.gracePullCmdWaitGroup.Done()
	}
}

func (p *PullCmd) done() {
	p.doneChan <- p
	p.sess.pullCmdMap.Delete(p.output.Seq())
	{
		// free count pull-launch
		p.sess.gracePullCmdWaitGroup.Done()
	}
}
