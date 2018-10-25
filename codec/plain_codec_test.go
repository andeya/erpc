package codec

import (
	"reflect"
	"testing"
)

func TestPlain(t *testing.T) {
	type (
		S string
		I int32
	)
	var (
		bytes = make([]byte, 0, 100)
		a     = []interface{}{S("sss"), I(123), []byte("1234567890"), []byte("asdfghjkl")}
		b     = []interface{}{new(S), new(I), make([]byte, 10), &bytes}
		c     = new(PlainCodec)
	)

	for k, v := range a {
		data, err := c.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		err = c.Unmarshal(data, b[k])
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(reflect.Indirect(reflect.ValueOf(b[k])).Interface(), v) {
			t.Logf("get: %v, but expect: %v", b[k], v)
		}
	}
}
