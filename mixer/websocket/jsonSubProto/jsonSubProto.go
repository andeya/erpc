// Package jsonSubProto is implemented JSON socket communication protocol.
package jsonSubProto

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"

	"github.com/henrylee2cn/goutil"
	tp "github.com/henrylee2cn/teleport"
	"github.com/tidwall/gjson"
)

// NewJsonSubProtoFunc is creation function of JSON socket protocol.
var NewJsonSubProtoFunc = func(rw tp.IOWithReadBuffer) tp.Proto {
	return &jsonSubProto{
		id:   'j',
		name: "json",
		rw:   rw,
	}
}

type jsonSubProto struct {
	id   byte
	name string
	rw   tp.IOWithReadBuffer
	rMu  sync.Mutex
}

// Version returns the protocol's id and name.
func (j *jsonSubProto) Version() (byte, string) {
	return j.id, j.name
}

const format = `{"seq":%d,"mtype":%d,"serviceMethod":%q,"meta":%q,"bodyCodec":%d,"body":"%s","xferPipe":%s}`

// Pack writes the Message into the connection.
// NOTE: Make sure to write only once or there will be package contamination!
func (j *jsonSubProto) Pack(m tp.Message) error {
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
	// marshal transfer pipe ids
	var xferPipeIds = make([]int, m.XferPipe().Len())
	for i, id := range m.XferPipe().Ids() {
		xferPipeIds[i] = int(id)
	}
	xferPipeIdsBytes, err := json.Marshal(xferPipeIds)
	if err != nil {
		return err
	}

	// join json format
	s := fmt.Sprintf(format,
		m.Seq(),
		m.Mtype(),
		m.ServiceMethod(),
		m.Meta().QueryString(),
		m.BodyCodec(),
		bytes.Replace(bodyBytes, []byte{'"'}, []byte{'\\', '"'}, -1),
		xferPipeIdsBytes,
	)

	b := goutil.StringToBytes(s)

	m.SetSize(uint32(len(b)))

	_, err = j.rw.Write(b)
	return err
}

// Unpack reads bytes from the connection to the Message.
// NOTE: Concurrent unsafe!
func (j *jsonSubProto) Unpack(m tp.Message) error {
	j.rMu.Lock()
	defer j.rMu.Unlock()
	b, err := ioutil.ReadAll(j.rw)
	if err != nil {
		return err
	}

	m.SetSize(uint32(len(b)))

	s := goutil.BytesToString(b)

	// read transfer pipe
	xferPipe := gjson.Get(s, "xferPipe")
	for _, r := range xferPipe.Array() {
		m.XferPipe().Append(byte(r.Int()))
	}

	// read body
	m.SetBodyCodec(byte(gjson.Get(s, "bodyCodec").Int()))
	body := gjson.Get(s, "body").String()
	bodyBytes, err := m.XferPipe().OnUnpack(goutil.StringToBytes(body))
	if err != nil {
		return err
	}

	// read other
	m.SetSeq(int32(gjson.Get(s, "seq").Int()))
	m.SetMtype(byte(gjson.Get(s, "mtype").Int()))
	m.SetServiceMethod(gjson.Get(s, "serviceMethod").String())
	meta := gjson.Get(s, "meta").String()
	m.Meta().ParseBytes(goutil.StringToBytes(meta))

	// unmarshal new body
	err = m.UnmarshalBody(bodyBytes)
	return err
}
