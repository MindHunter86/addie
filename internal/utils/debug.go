package utils

import (
	"fmt"
	"reflect"
	"unsafe"
)

func PrintSliceHeader[S ~[]E, E any](name string, s S) {
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&s)) // skipcq: GSC-G103 it's ok here
	fmt.Printf("%s: ptr=%#x len=%d cap=%d\n", name, hdr.Data, hdr.Len, hdr.Cap)
}
