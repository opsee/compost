package aws

import (
	opsee_types "github.com/opsee/protobuf/opseeproto/types"
	"io"
	"reflect"
	"time"
)

// Copy deeply copies a src structure to dst. Useful for copying request and
// response structures.
//
// Can copy between structs of different type, but will only copy fields which
// are assignable, and exist in both structs. Fields which are not assignable,
// or do not exist in both structs are ignored.
func CopyInto(dst, src interface{}) {
	dstval := reflect.ValueOf(dst)
	if !dstval.IsValid() {
		panic("Copy dst cannot be nil")
	}

	rcopy(dstval, reflect.ValueOf(src), true)
}

// rcopy performs a recursive copy of values from the source to destination.
//
// root is used to skip certain aspects of the copy which are not valid
// for the root node of a object.
func rcopy(dst, src reflect.Value, root bool) {
	if !src.IsValid() {
		return
	}

	switch src.Kind() {
	case reflect.Ptr:
		if _, ok := src.Interface().(io.Reader); ok {
			if dst.Kind() == reflect.Ptr && dst.Elem().CanSet() {
				dst.Elem().Set(src)
			} else if dst.CanSet() {
				dst.Set(src)
			}
		} else if tt, ok := src.Interface().(*time.Time); ok {
			if tt != nil && dst.CanSet() {
				timestamp := &opsee_types.Timestamp{}
				timestamp.Scan(*tt)
				dst.Set(reflect.ValueOf(timestamp))
			}
		} else {
			de := dst.Type().Elem()
			if dst.CanSet() && !src.IsNil() {
				dst.Set(reflect.New(de))
			}
			if src.Elem().IsValid() {
				// Keep the current root state since the depth hasn't changed
				rcopy(dst.Elem(), src.Elem(), root)
			}
		}
	case reflect.Struct:
		t := dst.Type()
		for i := 0; i < t.NumField(); i++ {
			name := t.Field(i).Name
			srcVal := src.FieldByName(name)
			dstVal := dst.FieldByName(name)
			if srcVal.IsValid() && dstVal.CanSet() {
				rcopy(dstVal, srcVal, false)
			}
		}
	case reflect.Slice:
		if src.IsNil() {
			break
		}

		s := reflect.MakeSlice(dst.Type(), src.Len(), src.Cap())
		dst.Set(s)
		for i := 0; i < src.Len(); i++ {
			rcopy(dst.Index(i), src.Index(i), false)
		}
	case reflect.Map:
		if src.IsNil() {
			break
		}

		s := reflect.MakeMap(dst.Type())
		dst.Set(s)
		for _, k := range src.MapKeys() {
			v := src.MapIndex(k)
			v2 := reflect.New(v.Type()).Elem()
			rcopy(v2, v, false)
			dst.SetMapIndex(k, v2)
		}
	default:

		if src.Type().AssignableTo(dst.Type()) {
			dst.Set(src)
		} else {
			dst.Set(src.Convert(dst.Type()))
		}
	}
}
