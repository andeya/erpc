// Package httproto is implemented HTTP style socket communication protocol.
//
// Copyright 2018 HenryLee. All Rights Reserved.
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
package httproto

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/henrylee2cn/erpc/v6"
	"github.com/henrylee2cn/erpc/v6/codec"
	"github.com/henrylee2cn/erpc/v6/utils"
	"github.com/henrylee2cn/erpc/v6/xfer"
	"github.com/henrylee2cn/erpc/v6/xfer/gzip"
	"github.com/henrylee2cn/goutil"
)

var (
	bodyCodecMapping = map[string]byte{
		"application/x-protobuf":            codec.ID_PROTOBUF,
		"application/json":                  codec.ID_JSON,
		"application/x-www-form-urlencoded": codec.ID_FORM,
		"text/plain":                        codec.ID_PLAIN,
		"text/xml":                          codec.ID_XML,
	}
	contentTypeMapping = map[byte]string{
		codec.ID_PROTOBUF: "application/x-protobuf;charset=utf-8",
		codec.ID_JSON:     "application/json;charset=utf-8",
		codec.ID_FORM:     "application/x-www-form-urlencoded;charset=utf-8",
		codec.ID_PLAIN:    "text/plain;charset=utf-8",
		codec.ID_XML:      "text/xml;charset=utf-8",
	}
)

// RegBodyCodec registers a mapping of content type to body coder.
func RegBodyCodec(contentType string, codecID byte) {
	bodyCodecMapping[contentType] = codecID
	contentTypeMapping[codecID] = contentType
}

// GetBodyCodec returns the codec id from content type.
func GetBodyCodec(contentType string, defCodecID byte) byte {
	idx := strings.Index(contentType, ";")
	if idx != -1 {
		contentType = contentType[:idx]
	}
	codecID, ok := bodyCodecMapping[contentType]
	if !ok {
		return defCodecID
	}
	return codecID
}

// GetContentType returns the content type from codec id.
func GetContentType(codecID byte, defContentType string) string {
	contentType, ok := contentTypeMapping[codecID]
	if !ok {
		return defContentType
	}
	return contentType
}

// NewHTTProtoFunc is creation function of HTTP style socket protocol.
// NOTE:
//  Only support xfer filter: gzip
//  Must use HTTP service method mapper
func NewHTTProtoFunc(printMessage ...bool) erpc.ProtoFunc {
	erpc.SetServiceMethodMapper(erpc.HTTPServiceMethodMapper)
	var printable bool
	if len(printMessage) > 0 {
		printable = printMessage[0]
	}
	return func(rw erpc.IOWithReadBuffer) erpc.Proto {
		return &httproto{
			id:           'h',
			name:         "http",
			rw:           rw,
			printMessage: printable,
		}
	}
}

type httproto struct {
	rw           erpc.IOWithReadBuffer
	rMu          sync.Mutex
	name         string
	id           byte
	printMessage bool
}

// Version returns the protocol's id and name.
func (h *httproto) Version() (byte, string) {
	return h.id, h.name
}

// Pack writes the Message into the connection.
// NOTE: Make sure to write only once or there will be package contamination!
func (h *httproto) Pack(m erpc.Message) (err error) {
	// marshal body
	bodyBytes, err := m.MarshalBody()
	if err != nil {
		return err
	}

	var header = make(http.Header, m.Meta().Len())

	// do transfer pipe
	m.XferPipe().Range(func(idx int, filter xfer.XferFilter) bool {
		if !gzip.Is(filter.ID()) {
			err = fmt.Errorf("unsupport xfer filter: %s", filter.Name())
			return false
		}
		bodyBytes, err = filter.OnPack(bodyBytes)
		if err != nil {
			return false
		}
		header.Set("Content-Encoding", "gzip")
		header.Set("X-Content-Encoding", filter.Name())
		return true
	})
	if err != nil {
		return err
	}

	header.Set("X-Seq", strconv.FormatInt(int64(m.Seq()), 10))
	header.Set("X-Mtype", strconv.Itoa(int(m.Mtype())))
	m.Meta().VisitAll(func(k, v []byte) {
		header.Add(goutil.BytesToString(k), goutil.BytesToString(v))
	})

	bb := utils.AcquireByteBuffer()
	defer utils.ReleaseByteBuffer(bb)
	switch m.Mtype() {
	case erpc.TypeCall, erpc.TypeAuthCall:
		err = h.packRequest(m, header, bb, bodyBytes)
	case erpc.TypeReply, erpc.TypeAuthReply:
		err = h.packResponse(m, header, bb, bodyBytes)
	// case erpc.TypePush:
	default:
		return fmt.Errorf("unsupport message type: %d(%s)", m.Mtype(), erpc.TypeText(m.Mtype()))
	}
	if err != nil {
		return err
	}
	m.SetSize(uint32(bb.Len()))
	if h.printMessage {
		erpc.Printf("Send HTTP Message:\n%s", goutil.BytesToString(bb.B))
	}
	_, err = h.rw.Write(bb.B)
	return err
}

var (
	methodBytes  = []byte("POST")
	versionBytes = []byte("HTTP/1.1")
	crlfBytes    = []byte("\r\n")
)

func (h *httproto) packRequest(m erpc.Message, header http.Header, bb *utils.ByteBuffer, bodyBytes []byte) error {
	u, err := url.Parse(m.ServiceMethod())
	if err != nil {
		return err
	}
	if u.Host != "" {
		header.Set("Host", u.Host)
	}
	header.Set("User-Agent", "erpc-httproto/1.1")
	bb.Write(methodBytes)
	bb.WriteByte(' ')
	if u.RawQuery == "" {
		bb.WriteString(u.Path)
	} else {
		bb.WriteString(u.Path + "?" + u.RawQuery)
	}
	bb.WriteByte(' ')
	bb.Write(versionBytes)
	bb.Write(crlfBytes)
	header.Set("Content-Type", GetContentType(m.BodyCodec(), "text/plain;charset=utf-8"))
	header.Set("Content-Length", strconv.Itoa(len(bodyBytes)))
	header.Set("Accept-Encoding", "gzip")
	header.Write(bb)
	bb.Write(crlfBytes)
	bb.Write(bodyBytes)
	return nil
}

var (
	okBytes     = []byte("200 OK")
	bizErrBytes = []byte("299 Business Error")
)

func (h *httproto) packResponse(m erpc.Message, header http.Header, bb *utils.ByteBuffer, bodyBytes []byte) error {
	bb.Write(versionBytes)
	bb.WriteByte(' ')
	if stat := m.Status(); !stat.OK() {
		statBytes, _ := stat.MarshalJSON()
		bb.Write(bizErrBytes)
		bb.Write(crlfBytes)
		if gzipName := header.Get("X-Content-Encoding"); gzipName != "" {
			gz, _ := xfer.GetByName(gzipName)
			statBytes, _ = gz.OnPack(statBytes)
		}
		header.Set("Content-Type", "application/json")
		header.Set("Content-Length", strconv.Itoa(len(statBytes)))
		header.Write(bb)
		bb.Write(crlfBytes)
		bb.Write(statBytes)
		return nil
	}
	bb.Write(okBytes)
	bb.Write(crlfBytes)
	header.Set("Content-Type", GetContentType(m.BodyCodec(), "text/plain"))
	header.Set("Content-Length", strconv.Itoa(len(bodyBytes)))
	header.Write(bb)
	bb.Write(crlfBytes)
	bb.Write(bodyBytes)
	return nil
}

var respPrefix = []byte("HTTP/")

// Unpack reads bytes from the connection to the Message.
func (h *httproto) Unpack(m erpc.Message) error {
	h.rMu.Lock()
	defer h.rMu.Unlock()
	bb := utils.AcquireByteBuffer()
	defer utils.ReleaseByteBuffer(bb)
	var size = 5
	bb.ChangeLen(size)
	_, err := io.ReadFull(h.rw, bb.B)
	if err != nil {
		return err
	}
	prefixBytes := make([]byte, 5, 128)
	copy(prefixBytes, bb.B)
	err = h.readLine(bb)
	if err != nil {
		return err
	}
	size += bb.Len()
	firstLine := append(prefixBytes, bb.B...)
	var msg []byte

	// response
	if bytes.Equal(prefixBytes, respPrefix) {
		m.SetMtype(erpc.TypeReply)
		// status line
		var ok bool
		a := bytes.SplitN(firstLine, spaceBytes, 2)
		if len(a) != 2 {
			return errBadHTTPMsg
		}
		if bytes.Equal(a[1], okBytes) {
			ok = true
		} else if !bytes.Equal(a[1], bizErrBytes) {
			// TODO support
			return errUnsupportHTTPCode
		}
		size, msg, err = h.unpack(m, bb)
		if err != nil {
			return err
		}
		if h.printMessage {
			erpc.Printf("Recv HTTP Message:\n%s\r\n%s",
				goutil.BytesToString(firstLine), goutil.BytesToString(msg))
		}
		size += len(firstLine)
		m.SetSize(uint32(size))
		if ok {
			return m.UnmarshalBody(bb.B)
		}
		m.UnmarshalBody(nil)
		return m.Status(true).UnmarshalJSON(bb.B)
	}

	// request
	m.SetMtype(erpc.TypeCall)
	a := bytes.SplitN(firstLine, spaceBytes, 3)
	if len(a) != 3 {
		return errBadHTTPMsg
	}
	u, err := url.Parse(goutil.BytesToString(a[1]))
	if err != nil {
		return err
	}
	m.SetServiceMethod(u.Path)
	if u.RawQuery != "" {
		m.Meta().ParseBytes(goutil.StringToBytes(u.RawQuery))
	}
	size, msg, err = h.unpack(m, bb)
	if err != nil {
		return err
	}
	if h.printMessage {
		erpc.Printf("Recv HTTP Message:\n%s\r\n%s",
			goutil.BytesToString(firstLine), goutil.BytesToString(msg))
	}
	size += len(firstLine)
	m.SetSize(uint32(size))
	return m.UnmarshalBody(bb.B)
}

var (
	spaceBytes            = []byte(" ")
	colonBytes            = []byte(":")
	contentTypeBytes      = []byte("Content-Type")
	contentLengthBytes    = []byte("Content-Length")
	xContentEncodingBytes = []byte("X-Content-Encoding")
	xSeqBytes             = []byte("X-Seq")
	xMtypeBytes           = []byte("X-Mtype")
	errBadHTTPMsg         = errors.New("bad HTTP message")
	errUnsupportHTTPCode  = errors.New("unsupport HTTP status code")
)

func (h *httproto) unpack(m erpc.Message, bb *utils.ByteBuffer) (size int, msg []byte, err error) {
	var bodySize int
	var a [][]byte
	for i := 0; true; i++ {
		err = h.readLine(bb)
		if err != nil {
			return 0, nil, err
		}
		if h.printMessage {
			msg = append(msg, bb.B...)
			msg = append(msg, '\r', '\n')
		}
		size += bb.Len()
		// blank line, to read body
		if bb.Len() == 0 {
			break
		}
		// header
		a = bytes.SplitN(bb.B, colonBytes, 2)
		if len(a) != 2 {
			return 0, nil, errBadHTTPMsg
		}
		a[1] = bytes.TrimSpace(a[1])
		if bytes.Equal(contentTypeBytes, a[0]) {
			m.SetBodyCodec(GetBodyCodec(goutil.BytesToString(a[1]), codec.NilCodecID))
			continue
		}
		if bytes.Equal(contentLengthBytes, a[0]) {
			bodySize, err = strconv.Atoi(goutil.BytesToString(a[1]))
			if err != nil {
				return 0, nil, errBadHTTPMsg
			}
			size += bodySize
			continue
		}
		if bytes.Equal(xContentEncodingBytes, a[0]) {
			zg, err := xfer.GetByName(goutil.BytesToString(a[1]))
			if err != nil {
				return 0, nil, err
			}
			m.XferPipe().Append(zg.ID())
			continue
		}
		if bytes.Equal(xSeqBytes, a[0]) {
			var seq int
			seq, err = strconv.Atoi(goutil.BytesToString(a[1]))
			if err != nil {
				return 0, nil, errBadHTTPMsg
			}
			m.SetSeq(int32(seq))
			continue
		}
		if bytes.Equal(xMtypeBytes, a[0]) {
			var mtype int
			mtype, err = strconv.Atoi(goutil.BytesToString(a[1]))
			if err != nil {
				return 0, nil, errBadHTTPMsg
			}
			m.SetMtype(byte(mtype))
			continue
		}
		m.Meta().SetBytesKV(a[0], a[1])
	}
	if bodySize == 0 {
		return size, msg, nil
	}
	bb.ChangeLen(bodySize)
	_, err = io.ReadFull(h.rw, bb.B)
	if err != nil {
		return 0, nil, err
	}
	if h.printMessage {
		msg = append(msg, bb.B...)
		msg = append(msg, '\r', '\n')
	}
	bb.B, err = m.XferPipe().OnUnpack(bb.B)
	return size, msg, err
}

func (h *httproto) readLine(bb *utils.ByteBuffer) error {
	bb.Reset()
	oneByte := make([]byte, 1)
	var err error
	for {
		_, err = io.ReadFull(h.rw, oneByte)
		if err != nil {
			return err
		}
		if oneByte[0] == '\n' {
			n := bb.Len()
			if n > 0 && bb.B[n-1] == '\r' {
				bb.B = bb.B[:n-1]
			}
			return nil
		}
		bb.Write(oneByte)
	}
}
