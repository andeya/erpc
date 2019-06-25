package goutil

import (
	"reflect"
	"unsafe"
)

// InitAndGetString if strPtr is empty string, initialize it with def,
// and return the final value.
func InitAndGetString(strPtr *string, def string) string {
	if strPtr == nil {
		return def
	}
	if *strPtr == "" {
		*strPtr = def
	}
	return *strPtr
}

// DereferenceType dereference, get the underlying non-pointer type.
func DereferenceType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// DereferenceValue dereference and unpack interface,
// get the underlying non-pointer and non-interface value.
func DereferenceValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	return v
}

// IsLittleEndian determine whether the current system is little endian.
func IsLittleEndian() bool {
	var i int32 = 0x01020304
	u := unsafe.Pointer(&i)
	pb := (*byte)(u)
	b := *pb
	return (b == 0x04)
}
