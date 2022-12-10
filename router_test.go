package erpc

import (
	"reflect"
	"testing"

	"github.com/andeya/goutil"
	"github.com/stretchr/testify/assert"
)

type A struct {
	B
}

func (A) X0()  {}
func (*A) X1() {}

type B struct {
}

func (B) Y0()  {}
func (*B) Y1() {}

func TestIsCompositionMethod(t *testing.T) {
	assert.False(t, goutil.IsCompositionMethod(reflect.TypeOf(A{}).Method(0)))
	assert.True(t, goutil.IsCompositionMethod(reflect.TypeOf(A{}).Method(1)))

	assert.False(t, goutil.IsCompositionMethod(reflect.TypeOf(&A{}).Method(0)), reflect.TypeOf(&A{}).Method(0))
	assert.False(t, goutil.IsCompositionMethod(reflect.TypeOf(&A{}).Method(1)))
	assert.True(t, goutil.IsCompositionMethod(reflect.TypeOf(&A{}).Method(2)))
	assert.True(t, goutil.IsCompositionMethod(reflect.TypeOf(&A{}).Method(3)))

	assert.False(t, goutil.IsCompositionMethod(reflect.TypeOf(B{}).Method(0)))

	assert.False(t, goutil.IsCompositionMethod(reflect.TypeOf(&B{}).Method(0)))
	assert.False(t, goutil.IsCompositionMethod(reflect.TypeOf(&B{}).Method(1)))
}

func TestResolveCtrlStructOrPoolFunc(t *testing.T) {
	builder, err := resolveCtrlStructOrPoolFunc(func() CtrlStructPtr { return new(A) })
	assert.NoError(t, err)
	assert.NotNil(t, builder)
	assert.Equal(t, reflect.TypeOf(&A{}), builder().Type())
}
