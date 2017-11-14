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
		checkReadLimit func(int64) error,
	) error
}

// default socket communication protocol
var (
	ProtoLee        Protocol = new(protoLee)
	defaultProtocol Protocol = ProtoLee
	lengthSize               = int64(binary.Size(uint32(0)))
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
	packet.Size = destWriter.Count()
	packet.HeaderLength = destWriter.Count() - lengthSize
	packet.BodyLength = 0
	if err != nil {
		return err
	}

	// write body
	defer func() {
		packet.Size = destWriter.Count()
		packet.BodyLength = packet.Size - packet.HeaderLength - lengthSize*2
	}()

	switch bo := packet.Body.(type) {
	case nil:
		codecId := GetCodecId(packet.BodyCodec)
		if codecId == codec.NilCodecId {
			err = binary.Write(destWriter, binary.BigEndian, uint32(0))
		} else {
			err = binary.Write(destWriter, binary.BigEndian, uint32(1))
			if err == nil {
				err = destWriter.WriteByte(codecId)
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
	err := tmpCodecWriter.WriteByte(tmpCodecWriter.Id())
	if err != nil {
		return err
	}
	err = tmpCodecWriter.Encode(gzip.NoCompression, header)
	if err != nil {
		return err
	}
	headerLength := uint32(tmpCodecWriter.Len())
	err = binary.Write(destWriter, binary.BigEndian, headerLength)
	if err != nil {
		return err
	}
	_, err = tmpCodecWriter.WriteTo(destWriter)
	return err
}

func (protoLee) writeBytesBody(destWriter *utils.BufioWriter, body []byte) error {
	bodyLength := uint32(len(body))
	err := binary.Write(destWriter, binary.BigEndian, bodyLength)
	if err != nil {
		return err
	}
	_, err = destWriter.Write(body)
	return err
}

func (protoLee) writeBody(destWriter *utils.BufioWriter, tmpCodecWriter *TmpCodecWriter, gzipLevel int, body interface{}) error {
	err := tmpCodecWriter.WriteByte(tmpCodecWriter.Id())
	if err != nil {
		return err
	}
	err = tmpCodecWriter.Encode(gzipLevel, body)
	if err != nil {
		return err
	}
	// write body to socket buffer
	bodyLength := uint32(tmpCodecWriter.Len())
	err = binary.Write(destWriter, binary.BigEndian, bodyLength)
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
	checkReadLimit func(int64) error,
) error {

	var (
		hErr, bErr error
		b          interface{}
	)
	srcReader.ResetCount()
	packet.HeaderCodec, hErr = p.readHeader(srcReader, codecReaderGetter, packet.Header, checkReadLimit)
	packet.Size = srcReader.Count()
	if srcReader.Count() > lengthSize {
		packet.HeaderLength = srcReader.Count() - lengthSize
	}

	if hErr == nil {
		b = bodyAdapter()
	} else {
		if hErr == io.EOF || hErr == io.ErrUnexpectedEOF {
			packet.Size = packet.HeaderLength
			packet.BodyLength = 0
			packet.BodyCodec = ""
			return hErr
		} else if isActiveClosed() {
			packet.Size = packet.HeaderLength
			packet.BodyLength = 0
			packet.BodyCodec = ""
			return ErrProactivelyCloseSocket
		}
	}

	srcReader.ResetCount()
	packet.BodyCodec, bErr = p.readBody(srcReader, codecReaderGetter, int(packet.Header.Gzip), b, packet.HeaderLength, checkReadLimit)
	packet.Size += srcReader.Count()
	if srcReader.Count() > lengthSize {
		packet.BodyLength = srcReader.Count() - lengthSize
	}
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
	checkReadLimit func(int64) error,
) (string, error) {

	srcReader.ResetLimit(-1)

	var headerLength uint32
	err := binary.Read(srcReader, binary.BigEndian, &headerLength)
	if err != nil {
		return "", err
	}

	// check packet size
	err = checkReadLimit(int64(headerLength) + lengthSize)
	if err != nil {
		return "", err
	}

	srcReader.ResetLimit(int64(headerLength))

	codecId, err := srcReader.ReadByte()
	if err != nil {
		return GetCodecName(codecId), err
	}

	codecReader, err := codecReaderGetter(codecId)
	if err != nil {
		return GetCodecName(codecId), err
	}

	err = codecReader.Decode(gzip.NoCompression, header)
	return codecReader.Name(), err
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
	headerLength int64,
	checkReadLimit func(int64) error,
) (string, error) {

	srcReader.ResetLimit(-1)

	var (
		bodyLength uint32
		codecId    = codec.NilCodecId
	)

	err := binary.Read(srcReader, binary.BigEndian, &bodyLength)
	if err != nil {
		return "", err
	}
	if bodyLength == 0 {
		return "", err
	}

	// check packet size
	err = checkReadLimit(headerLength + int64(bodyLength) + lengthSize*2)
	if err != nil {
		return "", err
	}

	srcReader.ResetLimit(int64(bodyLength))

	// read body
	switch bo := body.(type) {
	case nil:
		var codecName string
		codecName, err = readAll(srcReader, make([]byte, bodyLength))
		return codecName, err

	case []byte:
		var codecName string
		codecName, err = readAll(srcReader, bo)
		return codecName, err

	case *[]byte:
		*bo, err = ioutil.ReadAll(srcReader)
		return GetCodecNameFromBytes(*bo), err

	default:
		codecId, err = srcReader.ReadByte()
		if bodyLength == 1 || err != nil {
			return GetCodecName(codecId), err
		}
		codecReader, err := codecReaderGetter(codecId)
		if err != nil {
			return GetCodecName(codecId), err
		}
		err = codecReader.Decode(gzipLevel, body)
		return codecReader.Name(), err
	}
}

func readAll(reader io.Reader, p []byte) (string, error) {
	perLen := len(p)
	_, err := reader.Read(p[:perLen])
	if err == nil {
		_, err = io.Copy(ioutil.Discard, reader)
	}
	return GetCodecNameFromBytes(p), err
}
