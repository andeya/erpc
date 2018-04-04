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
		a = []interface{}{S("sss"), I(123)}
		b = []interface{}{new(S), new(I)}
		c = new(PlainCodec)
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
		vv := reflect.ValueOf(b[k]).Elem().Interface()
		if v != vv {
			t.Logf("get: %v, but expect: %v", vv, v)
		}
	}
}
