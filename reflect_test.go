package sos

import (
	"log"
	"testing"
)

func TestSimpleSchemaDeserialization(t *testing.T) {
	obj := getSimpleObject(123)

	s := simpleObject{}

	schemaToStruct(&s, obj)
	log.Printf("S: %+v", s)
	if s.Int != 123 {
		t.Error("values did not match")
	}
}
func TestComplicatedSchemaDeserialization(t *testing.T) {

	obj := getComplicatedObject(1, 2.0, "hello", 1.0, 2.0, 3.0)
	s2 := complicatedObject{}

	schemaToStruct(&s2, obj)
	if s2.Int != 1 {
		t.Error("Int values did not match")
	}
	if s2.Float != 2.0 {
		t.Error("Float values did not match")
	}
	if s2.Name != "hello" {
		t.Error("String values did not match")
	}
	if s2.Pos[0] != 1.0 && s2.Pos[1] != 2.0 && s2.Pos[2] != 3.0 {
		t.Error("Array values did not match")
	}
}

func TestCompoundSchemaDeserialization(t *testing.T) {
	log.Printf("-----------------------")

	obj := getCompoundObject(10, 5.0, "omg")
	s3 := compoundObject{}

	schemaToStruct(&s3, obj)
	if s3.Int.Val != 10 {
		t.Error("int values did not match")
	}
	if s3.Float.Val != 5.0 {
		t.Error("float values did not match")
	}
	if s3.String.Val != "omg" {
		t.Error("string values did not match")
	}
}
