package sos

// #cgo LDFLAGS: -Wl,-rpath,\$ORIGIN -L${SRCDIR}/c_sdk -limprobable_worker
// #include "c_sdk/include/improbable/c_schema.h"
// #include "c_sdk/include/improbable/c_worker.h"
// #include <inttypes.h>
// #include <stdio.h>
// #include <stdlib.h>
// #include <string.h>
import "C"
import (
	"reflect"
	"sort"
	"strconv"
	"unsafe"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func init() {
	//log.SetLevel(logrus.DebugLevel)
}

func handleStructField(o *C.Schema_Object, i int, f reflect.Value) {

	switch f.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		log.Debugf("C.Schema_AddInt32 %d: %d", i, f.Int())
		C.Schema_AddInt32(o, C.uint(i), C.int(f.Int()))
	case reflect.Int64:
		log.Debugf("C.Schema_AddInt64 %d: %d", i, f.Int())
		C.Schema_AddInt64(o, C.uint(i), C.int64_t(f.Int()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		log.Debugf("C.Schema_AddUint32 %d: %d", i, f.Uint())
		C.Schema_AddUint32(o, C.uint(i), C.uint(f.Uint()))
	case reflect.Uint64:
		log.Debugf("C.Schema_AddUint64 %d: %d", i, f.Uint())
		C.Schema_AddUint64(o, C.uint(i), C.uint64_t(f.Uint()))
	case reflect.Float32:
		log.Debugf("C.Schema_AddFloat %d: %f", i, f.Float())
		C.Schema_AddFloat(o, C.uint(i), C.float(f.Float()))
	case reflect.Float64:
		log.Debugf("C.Schema_AddDouble %d: %f", i, f.Float())
		C.Schema_AddDouble(o, C.uint(i), C.double(f.Float()))
	case reflect.String:
		str := f.String()
		log.Debugf("C.Schema_AddBytes %d: %s", i, str)
		C.Schema_AddBytes(o, C.uint(i), (*C.uchar)(C.CBytes([]byte(str))), C.uint(len(str)))
	case reflect.Bool:
		log.Debugf("C.Schema_AddBoolean %d: %v", i, f.Bool())
		if f.Bool() {
			C.Schema_AddBool(o, C.uint(i), C.uint8_t(1))
		} else {
			C.Schema_AddBool(o, C.uint(i), C.uint8_t(0))
		}
	case reflect.Struct:
		log.Debugf("C.Schema_AddObject %d: %+v", i, f.Interface())
		obj := C.Schema_AddObject(o, C.uint(i))
		structToObj(obj, f.Interface())
	case reflect.Slice:
		for j := 0; j < f.Len(); j++ {
			v := f.Index(j)
			handleStructField(o, i, v)
		}
	case reflect.Map:
		log.Debugf("C.Schema_AddObject %d: %+v", i, f)
		iter := f.MapRange()
		for iter.Next() {
			obj := C.Schema_AddObject(o, C.uint(i))
			handleStructField(obj, C.SCHEMA_MAP_KEY_FIELD_ID, iter.Key())
			handleStructField(obj, C.SCHEMA_MAP_VALUE_FIELD_ID, iter.Value())

		}
	case reflect.Array:
		log.Debugf("Array:  %d %+v", i, f)
		//obj := C.Schema_AddObject(o, C.uint(i))
		for j := 0; j < f.Len(); j++ {
			v := f.Index(j)
			handleStructField(o, i, v)
		}
	case reflect.Ptr:
		if !f.IsZero() {
			log.Printf("Got a pointer: %+v %+v", f, reflect.Indirect(f))
			handleStructField(o, i, reflect.Indirect(f))
		}

	default:
		log.Printf("unknown kind: %+v: %+v", f.Kind(), f)
	}

}

func structToObj(o *C.Schema_Object, s interface{}) {
	v := reflect.ValueOf(s)
	t := reflect.TypeOf(s)
	slot := 1
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		st := t.Field(i)
		if st.Tag.Get("sos") == "-" {
			continue
		}

		handleStructField(o, slot, f)
		slot++
	}

}

func structToSchema(cid ComponentID, s interface{}) C.Worker_ComponentData {
	var cd C.Worker_ComponentData

	cd.component_id = C.uint(cid)

	cd.schema_type = C.Schema_CreateComponentData()
	obj := C.Schema_GetComponentDataFields(cd.schema_type)
	structToObj(obj, s)

	return cd
}

func handleSchemaField(o *C.Schema_Object, i int, j int, f reflect.Value) {
	if !f.CanSet() {
		log.Debugf("Can't set field: %v", f)
		return
	}
	switch f.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		v := C.Schema_IndexInt32(o, C.uint(i), C.uint(j))
		f.SetInt(int64(v))
		log.Debugf("C.Schema_GetInt32 %d %d %d", i, f.Int(), v)
	case reflect.Int64:
		v := C.Schema_IndexInt64(o, C.uint(i), C.uint(j))
		f.SetInt(int64(v))
		log.Debugf("C.Schema_IndexInt64 %d: %d", i, f.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		v := C.Schema_IndexUint32(o, C.uint(i), C.uint(j))
		f.SetUint(uint64(v))
		log.Debugf("C.Schema_IndexUint32 %d: %d", i, f.Uint())
	case reflect.Uint64:
		v := C.Schema_IndexUint32(o, C.uint(i), C.uint(j))
		f.SetUint(uint64(v))
		log.Debugf("C.Schema_IndexUint64 %d: %d", i, f.Uint())
	case reflect.Float32:
		v := C.Schema_IndexFloat(o, C.uint(i), C.uint(j))
		f.SetFloat(float64(v))
		log.Debugf("C.Schema_IndexFloat %d: %f", i, f.Float())
	case reflect.Float64:
		v := C.Schema_IndexDouble(o, C.uint(i), C.uint(j))
		f.SetFloat(float64(v))
		log.Debugf("C.Schema_IndexDouble %d: %f", i, f.Float())
	case reflect.String:
		b := C.Schema_IndexBytes(o, C.uint(i), C.uint(j))
		l := C.Schema_IndexBytesLength(o, C.uint(i), C.uint(j))
		f.SetString(C.GoStringN((*C.char)(unsafe.Pointer(b)), C.int(l)))
		log.Debugf("C.Schema_IndexBytes %d: %s", i, f.String())
	case reflect.Bool:
		b := C.Schema_IndexBool(o, C.uint(i), C.uint(j))
		f.SetBool(b == 1)
		log.Debugf("C.Schema_IndexBool %d: %v", i, f.Bool())
	case reflect.Struct:
		obj := C.Schema_IndexObject(o, C.uint(i), C.uint(j))
		schemaToStruct(f.Addr().Interface(), obj)
		log.Debugf("C.Schema_IndexObject %d: %+v", i, f.Interface())
	case reflect.Slice:
		num := C.Schema_GetObjectCount(o, C.uint(i))
		typ := reflect.TypeOf(f.Interface())
		slice := reflect.MakeSlice(typ, int(num), int(num))
		for j := 0; j < slice.Len(); j++ {
			v := slice.Index(j)
			handleSchemaField(o, i, j, v)
		}
		f.Set(slice)
		log.Debugf("Got a slice: %+v %d", f, num)
	case reflect.Map:
		num := C.Schema_GetObjectCount(o, C.uint(i))
		typ := reflect.TypeOf(f.Interface())
		m := reflect.MakeMapWithSize(typ, int(num))
		for j := 0; j < int(num); j++ {
			obj := C.Schema_IndexObject(o, C.uint(i), C.uint(j))
			k := reflect.Indirect(reflect.New(typ.Key()))
			v := reflect.Indirect(reflect.New(typ.Elem()))
			handleSchemaField(obj, C.SCHEMA_MAP_KEY_FIELD_ID, 0, k)
			handleSchemaField(obj, C.SCHEMA_MAP_VALUE_FIELD_ID, 0, v)

			m.SetMapIndex(k, v)

		}
		log.Debugf("C.Schema_IndexObject %d: %+v", i, f)
		f.Set(m)

	case reflect.Ptr:
		ptr := reflect.New(reflect.TypeOf(f.Interface()).Elem())
		ptrVal := reflect.Indirect(ptr)
		handleSchemaField(o, i, j, ptrVal)
		log.Debugf("Converted pointer to: %+v %+v", ptr, ptrVal)
		f.Set(ptr)
	case reflect.Array:
		for j := 0; j < f.Len(); j++ {
			v := f.Index(j)
			handleSchemaField(o, i, j, v)
		}
		log.Debugf("Got an array, %+v", f)

	default:
		log.Printf("unknown kind: %+v: %+v", f.Kind(), f)
	}

}

func schemaToStruct(ent interface{}, o *C.Schema_Object) interface{} {
	v := reflect.ValueOf(ent).Elem()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		handleSchemaField(o, i+1, 0, f)
	}

	return v

}

func ComponentIDsFromInterface(ent interface{}) []ComponentID {
	var out []ComponentID

	v := reflect.ValueOf(ent)
	t := reflect.TypeOf(ent)
	for i := 0; i < v.NumField(); i++ {
		st := t.Field(i)
		if st.Tag.Get("sos") == "-" {
			continue
		}
		cid, err := strconv.Atoi(st.Tag.Get("sos"))
		if err != nil {
			log.Printf("Error in struct tag for %+v: %d", ent, i)
		}

		out = append(out, ComponentID(cid))
	}
	log.Printf("Component IDS: %+v", out)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out

}

type simpleObject struct {
	Int int
}

func getSimpleObject(val int) *C.Schema_Object {
	st := C.Schema_CreateComponentData()
	obj := C.Schema_GetComponentDataFields(st)

	C.Schema_AddInt32(obj, 1, C.int(val))

	return obj
}

type complicatedObject struct {
	Int   int32
	Float float32
	Name  string
	Pos   [3]float32
	Ptr   *int32
}

func getComplicatedObject(i int, f float32, name string, x float32, y float32, z float32, ptr int) *C.Schema_Object {
	st := C.Schema_CreateComponentData()
	obj := C.Schema_GetComponentDataFields(st)

	C.Schema_AddInt32(obj, 1, C.int(i))
	C.Schema_AddFloat(obj, 2, C.float(f))
	C.Schema_AddBytes(obj, 3, (*C.uchar)(C.CBytes([]byte(name))), C.uint(len(name)))
	C.Schema_AddFloat(obj, 4, C.float(x))
	C.Schema_AddFloat(obj, 4, C.float(y))
	C.Schema_AddFloat(obj, 4, C.float(z))
	C.Schema_AddInt32(obj, 5, C.int(ptr))

	return obj
}

type compoundObject struct {
	Int struct {
		Val int
	}
	Float struct {
		Val float32
	}
	String struct {
		Val string
	}
}

func getCompoundObject(i int, f float32, name string) *C.Schema_Object {
	st := C.Schema_CreateComponentData()
	obj := C.Schema_GetComponentDataFields(st)

	iobj := C.Schema_AddObject(obj, 1)
	C.Schema_AddInt32(iobj, 1, C.int(i))

	fobj := C.Schema_AddObject(obj, 2)
	C.Schema_AddFloat(fobj, 1, C.float(f))

	sobj := C.Schema_AddObject(obj, 3)
	C.Schema_AddBytes(sobj, 1, (*C.uchar)(C.CBytes([]byte(name))), C.uint(len(name)))

	return obj
}

func blankSchemaObject() *C.Schema_Object {
	st := C.Schema_CreateComponentData()
	obj := C.Schema_GetComponentDataFields(st)
	return obj

}
