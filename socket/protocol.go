// Socket package provides a concise, powerful and high-performance TCP
//
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
	"compress/gzip"
	"encoding/binary"
	"io"
	"io/ioutil"

	"github.com/henrylee2cn/teleport/codec"
	"github.com/henrylee2cn/teleport/utils"
)

// Protocol socket communication protocol
type Protocol interface {
	// WritePacket writes header and body to the connection.
	WritePacket(
		packet *Packet,
		destWriter *utils.BufioWriter,
		tmpCodecWriterGetter func(codecName string) (*TmpCodecWriter, error),
		isActiveClosed func() bool,
	) error

	// ReadPacket reads header and body from the connection.
	ReadPacket(
		packet *Packet,
		bodyAdapter func() interface{},
		srcReader *utils.BufioReader,
		codecReaderGetter func(codecId byte) (*CodecReader, error),
		isActiveClosed func() bool,
	) error
}

// default socket communication protocol
var (
	ProtoLee        Protocol = new(protoLee)
	defaultProtocol Protocol = ProtoLee
)

// GetDefaultProtocol gets the default socket communication protocol
func GetDefaultProtocol() Protocol {
	return defaultProtocol
}

// SetDefaultProtocol sets the default socket communication protocol
func SetDefaultProtocol(protocol Protocol) {
	defaultProtocol = protocol
}

type protoLee struct{}

// WritePacket writes header and body to the connection.
// WritePacket can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
// Note:
//  For the byte stream type of body, write directly, do not do any processing;
//  Must be safe for concurrent use by multiple goroutines.
func (p protoLee) WritePacket(
	packet *Packet,
	destWriter *utils.BufioWriter,
	tmpCodecWriterGetter func(codecName string) (*TmpCodecWriter, error),
	isActiveClosed func() bool,
) error {

	// write header
	if len(packet.HeaderCodec) == 0 {
		packet.HeaderCodec = defaultHeaderCodec.Name()
	}
	tmpCodecWriter, err := tmpCodecWriterGetter(packet.HeaderCodec)
	if err != nil {
		return err
	}
	err = p.writeHeader(destWriter, tmpCodecWriter, packet.Header)
	packet.HeaderLength = destWriter.Count()
	packet.Length = packet.HeaderLength
	packet.BodyLength = 0
	if err != nil {
		return err
	}

	// write body
	defer func() {
		packet.Length = destWriter.Count()
		packet.BodyLength = packet.Length - packet.HeaderLength
	}()

	switch bo := packet.Body.(type) {
	case nil:
		codecId := GetCodecId(packet.BodyCodec)
		if codecId == codec.NilCodecId {
			err = binary.Write(destWriter, binary.BigEndian, uint32(0))
		} else {
			err = binary.Write(destWriter, binary.BigEndian, uint32(1))
			if err == nil {
				err = binary.Write(destWriter, binary.BigEndian, codecId)
			}
		}

	case []byte:
		err = p.writeBytesBody(destWriter, bo)
	case *[]byte:
		err = p.writeBytesBody(destWriter, *bo)
	default:
		if len(packet.BodyCodec) == 0 {
			packet.BodyCodec = defaultBodyCodec.Name()
		}
		tmpCodecWriter, err = tmpCodecWriterGetter(packet.BodyCodec)
		if err == nil {
			err = p.writeBody(destWriter, tmpCodecWriter, int(packet.Header.Gzip), bo)
		}
	}
	if err != nil {
		return err
	}
	return destWriter.Flush()
}

func (protoLee) writeHeader(destWriter *utils.BufioWriter, tmpCodecWriter *TmpCodecWriter, header *Header) error {
	err := binary.Write(tmpCodecWriter, binary.BigEndian, tmpCodecWriter.Id())
	if err != nil {
		return err
	}
	err = tmpCodecWriter.Encode(gzip.NoCompression, header)
	if err != nil {
		return err
	}
	headerSize := uint32(tmpCodecWriter.Len())
	err = binary.Write(destWriter, binary.BigEndian, headerSize)
	if err != nil {
		return err
	}
	_, err = tmpCodecWriter.WriteTo(destWriter)
	return err
}

func (protoLee) writeBytesBody(destWriter *utils.BufioWriter, body []byte) error {
	bodySize := uint32(len(body))
	err := binary.Write(destWriter, binary.BigEndian, bodySize)
	if err != nil {
		return err
	}
	_, err = destWriter.Write(body)
	return err
}

func (protoLee) writeBody(destWriter *utils.BufioWriter, tmpCodecWriter *TmpCodecWriter, gzipLevel int, body interface{}) error {
	err := binary.Write(tmpCodecWriter, binary.BigEndian, tmpCodecWriter.Id())
	if err != nil {
		return err
	}
	err = tmpCodecWriter.Encode(gzipLevel, body)
	if err != nil {
		return err
	}
	// write body to socket buffer
	bodySize := uint32(tmpCodecWriter.Len())
	err = binary.Write(destWriter, binary.BigEndian, bodySize)
	if err != nil {
		return err
	}
	_, err = tmpCodecWriter.WriteTo(destWriter)
	return err
}

// ReadPacket reads header and body from the connection.
// Note:
//  For the byte stream type of body, read directly, do not do any processing;
//  Must be safe for concurrent use by multiple goroutines.
func (p protoLee) ReadPacket(
	packet *Packet,
	bodyAdapter func() interface{},
	srcReader *utils.BufioReader,
	codecReaderGetter func(codecId byte) (*CodecReader, error),
	isActiveClosed func() bool,
) error {

	var (
		hErr, bErr error
		b          interface{}
	)
	packet.HeaderLength, packet.HeaderCodec, hErr = p.readHeader(srcReader, codecReaderGetter, packet.Header)
	if hErr == nil {
		b = bodyAdapter()
	} else {
		if hErr == io.EOF || hErr == io.ErrUnexpectedEOF {
			packet.Length = packet.HeaderLength
			packet.BodyLength = 0
			packet.BodyCodec = ""
			return hErr
		} else if isActiveClosed() {
			packet.Length = packet.HeaderLength
			packet.BodyLength = 0
			packet.BodyCodec = ""
			return ErrProactivelyCloseSocket
		}
	}

	packet.BodyLength, packet.BodyCodec, bErr = p.readBody(srcReader, codecReaderGetter, int(packet.Header.Gzip), b)
	packet.Length = packet.HeaderLength + packet.BodyLength
	if isActiveClosed() {
		return ErrProactivelyCloseSocket
	}
	return bErr
}

// readHeader reads header from the connection.
// readHeader can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
// Note: must use only one goroutine call.
func (protoLee) readHeader(
	srcReader *utils.BufioReader,
	codecReaderGetter func(byte) (*CodecReader, error),
	header *Header,
) (int64, string, error) {

	srcReader.ResetCount()
	srcReader.ResetLimit(-1)

	var headerSize uint32
	err := binary.Read(srcReader, binary.BigEndian, &headerSize)
	if err != nil {
		return srcReader.Count(), "", err
	}

	srcReader.ResetLimit(int64(headerSize))

	var codecId = codec.NilCodecId

	err = binary.Read(srcReader, binary.BigEndian, &codecId)
	if err != nil {
		return srcReader.Count(), GetCodecName(codecId), err
	}

	codecReader, err := codecReaderGetter(codecId)
	if err != nil {
		return srcReader.Count(), GetCodecName(codecId), err
	}

	err = codecReader.Decode(gzip.NoCompression, header)
	return srcReader.Count(), codecReader.Name(), err
}

// readBody reads body from the connection.
// readBody can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
// Note: must use only one goroutine call, and it must be called after calling the readHeader().
func (protoLee) readBody(
	srcReader *utils.BufioReader,
	codecReaderGetter func(byte) (*CodecReader, error),
	gzipLevel int,
	body interface{},
) (int64, string, error) {

	srcReader.ResetCount()
	srcReader.ResetLimit(-1)

	var (
		bodySize uint32
		codecId  = codec.NilCodecId
	)

	err := binary.Read(srcReader, binary.BigEndian, &bodySize)
	if err != nil {
		return srcReader.Count(), "", err
	}
	if bodySize == 0 {
		return srcReader.Count(), "", err
	}

	srcReader.ResetLimit(int64(bodySize))

	// read body
	switch bo := body.(type) {
	case nil:
		var codecName string
		codecName, err = readAll(srcReader, make([]byte, 1))
		return srcReader.Count(), codecName, err

	case []byte:
		var codecName string
		codecName, err = readAll(srcReader, bo)
		return srcReader.Count(), codecName, err

	case *[]byte:
		*bo, err = ioutil.ReadAll(srcReader)
		return srcReader.Count(), GetCodecNameFromBytes(*bo), err

	default:
		err = binary.Read(srcReader, binary.BigEndian, &codecId)
		if bodySize == 1 || err != nil {
			return srcReader.Count(), GetCodecName(codecId), err
		}
		codecReader, err := codecReaderGetter(codecId)
		if err != nil {
			return srcReader.Count(), GetCodecName(codecId), err
		}
		err = codecReader.Decode(gzipLevel, body)
		return srcReader.Count(), codecReader.Name(), err
	}
}

func readAll(reader io.Reader, p []byte) (string, error) {
	perLen := len(p)
	_, err := reader.Read(p[perLen:])
	if err == nil {
		_, err = io.Copy(ioutil.Discard, reader)
	}
	if len(p) > perLen {
		return GetCodecName(p[perLen]), err
	}
	return "", err
}
