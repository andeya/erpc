// Package md5 provides a integrity check transfer filter
package md5

import (
	"bytes"
	"crypto/md5"
	"errors"

	"github.com/henrylee2cn/erpc/v6/xfer"
)

// Reg registers a md5 checker filter for transfer.
func Reg(id byte, name string) {
	xfer.Reg(&md5Hash{
		id:   id,
		name: name,
	})
}

// md5Hash compression filter
type md5Hash struct {
	id   byte
	name string
}

const md5Length = 16

var errDataCheck = errors.New("check failed")

// ID returns transfer filter id.
func (m *md5Hash) ID() byte {
	return m.id
}

// ID returns transfer filter name.
func (m *md5Hash) Name() string {
	return m.name
}

func (m *md5Hash) OnPack(src []byte) ([]byte, error) {
	content, err := getMd5(src)
	if err != nil {
		return nil, err
	}
	src = append(src, content...)

	return src, nil
}

func (m *md5Hash) OnUnpack(src []byte) ([]byte, error) {
	srcLength := len(src)
	if srcLength < md5Length {
		return nil, errDataCheck
	}
	srcData := src[:srcLength-md5Length]
	content, err := getMd5(srcData)
	if err != nil {
		return nil, err
	}
	// Check
	if !bytes.Equal(content, src[srcLength-md5Length:]) {
		return nil, errDataCheck
	}
	return srcData, nil
}

// Get md5 data
func getMd5(src []byte) ([]byte, error) {
	newMd5 := md5.New()
	_, err := newMd5.Write(src)
	if err != nil {
		return nil, err
	}

	return newMd5.Sum(nil), nil
}
