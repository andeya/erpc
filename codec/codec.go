package codec

import (
	"fmt"
	"io"
)

type (
	Encoder interface {
		Encode(v interface{}) error
	}
	EncodeMaker func(io.Writer) Encoder
)

var encodeMakerMap = make(map[string]EncodeMaker)

func RegEncodeMaker(name string, newFunc EncodeMaker) {
	if _, ok := encodeMakerMap[name]; ok {
		panic("multi-register encode maker: " + name)
	}
	encodeMakerMap[name] = newFunc
}

func GetEncodeMaker(name string) (EncodeMaker, error) {
	newFunc, ok := encodeMakerMap[name]
	if !ok {
		return nil, fmt.Errorf("unsupported encode maker: %s", name)
	}
	return newFunc, nil
}

func MakeEncoder(name string, w io.Writer) (Encoder, error) {
	newFunc, err := GetEncodeMaker(name)
	if err != nil {
		return nil, err
	}
	return newFunc(w), nil
}

type (
	Decoder interface {
		Decode(v interface{}) error
		// Buffered() io.Reader
	}
	DecodeMaker func(io.Reader) Decoder
)

var decodeMakerMap = make(map[string]DecodeMaker)

func RegDecodeMaker(name string, newFunc DecodeMaker) {
	if _, ok := decodeMakerMap[name]; ok {
		panic("multi-register decode maker: " + name)
	}
	decodeMakerMap[name] = newFunc
}

func GetDecodeMaker(name string) (DecodeMaker, error) {
	newFunc, ok := decodeMakerMap[name]
	if !ok {
		return nil, fmt.Errorf("unsupported decode maker: %s", name)
	}
	return newFunc, nil
}

func MakeDecoder(name string, r io.Reader) (Decoder, error) {
	newFunc, err := GetDecodeMaker(name)
	if err != nil {
		return nil, err
	}
	return newFunc(r), nil
}
