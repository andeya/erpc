package codec

import (
	"encoding/json"
	"io"

	"github.com/henrylee2cn/teleport/utils"
)

func init() {
	RegEncodeMaker("json", NewJsonEncoder)
	RegDecodeMaker("json", NewJsonDecoder)
}

// JsonEncoder a Encoder writes JSON values to an output stream, and count size.
type JsonEncoder struct {
	*json.Encoder
	counter *utils.Counter
}

// NewJsonEncoder returns a new json encoder that writes to w.
func NewJsonEncoder(w io.Writer) Encoder {
	counter := new(utils.Counter)
	encoder := json.NewEncoder(io.MultiWriter(w, counter))
	return &JsonEncoder{
		Encoder: encoder,
		counter: counter,
	}
}

// Encode writes the JSON encoding of v to the stream,
// followed by a newline character.
//
// See the documentation for Marshal for details about the
// conversion of Go values to JSON.
func (j *JsonEncoder) Encode(v interface{}) (int, error) {
	err := j.Encoder.Encode(v)
	n := j.counter.Count()
	j.counter.Reset()
	return n, err
}

// JsonDecoder a Decoder reads and decodes JSON values from an input stream, and count size.
type JsonDecoder struct {
	*json.Decoder
	counter *utils.Counter
}

// NewJsonDecoder returns a new decoder that reads from r.
//
// The decoder introduces its own buffering and may
// read data from r beyond the JSON values requested.
func NewJsonDecoder(r io.Reader) Decoder {
	counter := new(utils.Counter)
	decoder := json.NewDecoder(io.TeeReader(r, counter))
	return &JsonDecoder{
		Decoder: decoder,
		counter: counter,
	}
}

// Decode reads the next JSON-encoded value from its
// input and stores it in the value pointed to by v.
//
// See the documentation for Unmarshal for details about
// the conversion of JSON into a Go value.
func (j *JsonDecoder) Decode(v interface{}) (int, error) {
	err := j.Decoder.Decode(v)
	n := j.counter.Count()
	j.counter.Reset()
	return n, err
}
