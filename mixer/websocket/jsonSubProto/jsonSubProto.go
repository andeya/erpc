// Package jsonSubProto is implemented JSON socket communication protocol.
package jsonSubProto

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"sync"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/teleport/socket"
	"github.com/tidwall/gjson"
)

// NewJsonSubProtoFunc is creation function of JSON socket protocol.
var NewJsonSubProtoFunc = func(rw io.ReadWriter) socket.Proto {
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
	return &jsonSubProto{
		id:   'j',
		name: "json",
		r:    bufio.NewReaderSize(rw, readBufioSize),
		w:    rw,
	}
}

type jsonSubProto struct {
	id   byte
	name string
	r    *bufio.Reader
	w    io.Writer
	rMu  sync.Mutex
}

// Version returns the protocol's id and name.
func (j *jsonSubProto) Version() (byte, string) {
	return j.id, j.name
}

const format = `{"seq":%q,"ptype":%d,"uri":%q,"meta":%q,"body_codec":%d,"body":"%s","xfer_pipe":%s}`

// Pack writes the Packet into the connection.
// Note: Make sure to write only once or there will be package contamination!
func (j *jsonSubProto) Pack(p *socket.Packet) error {
	// marshal body
	bodyBytes, err := p.MarshalBody()
	if err != nil {
		return err
	}
	// do transfer pipe
	bodyBytes, err = p.XferPipe().OnPack(bodyBytes)
	if err != nil {
		return err
	}
	// marshal transfer pipe ids
	var xferPipeIds = make([]int, p.XferPipe().Len())
	for i, id := range p.XferPipe().Ids() {
		xferPipeIds[i] = int(id)
	}
	xferPipeIdsBytes, err := json.Marshal(xferPipeIds)
	if err != nil {
		return err
	}

	// join json format
	s := fmt.Sprintf(format,
		p.Seq(),
		p.Ptype(),
		p.Uri(),
		p.Meta().QueryString(),
		p.BodyCodec(),
		bytes.Replace(bodyBytes, []byte{'"'}, []byte{'\\', '"'}, -1),
		xferPipeIdsBytes,
	)

	b := goutil.StringToBytes(s)

	p.SetSize(uint32(len(b)))

	_, err = j.w.Write(b)
	return err
}

// Unpack reads bytes from the connection to the Packet.
// Note: Concurrent unsafe!
func (j *jsonSubProto) Unpack(p *socket.Packet) error {
	j.rMu.Lock()
	defer j.rMu.Unlock()
	b, err := ioutil.ReadAll(j.r)
	if err != nil {
		return err
	}

	p.SetSize(uint32(len(b)))

	s := goutil.BytesToString(b)

	// read transfer pipe
	xferPipe := gjson.Get(s, "xfer_pipe")
	for _, r := range xferPipe.Array() {
		p.XferPipe().Append(byte(r.Int()))
	}

	// read body
	p.SetBodyCodec(byte(gjson.Get(s, "body_codec").Int()))
	body := gjson.Get(s, "body").String()
	bodyBytes, err := p.XferPipe().OnUnpack(goutil.StringToBytes(body))
	if err != nil {
		return err
	}

	// read other
	p.SetSeq(gjson.Get(s, "seq").String())
	p.SetPtype(byte(gjson.Get(s, "ptype").Int()))
	p.SetUri(gjson.Get(s, "uri").String())
	meta := gjson.Get(s, "meta").String()
	p.Meta().ParseBytes(goutil.StringToBytes(meta))

	// unmarshal new body
	err = p.UnmarshalBody(bodyBytes)
	return err
}
