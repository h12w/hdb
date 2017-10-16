package vgob

import (
	"reflect"
	"testing"
)

func TestCodec(t *testing.T) {
	type S struct {
		V string
	}
	s := S{V: "a"}
	version := uint(1)
	var data, schemaData []byte
	{
		schema, err := NewSchema(S{})
		if err != nil {
			t.Fatal(err)
		}
		m, err := NewMarshaler(S{}, schema)
		if err != nil {
			t.Fatal(err)
		}
		schemaData = schema.Bytes()
		schema.SetVersion(version)
		data, err = m.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
	}
	{
		schema, err := NewSchema(schemaData)
		if err != nil {
			t.Fatal(err)
		}
		schema.SetVersion(version)
		schemas := map[uint]*Schema{
			version: schema,
		}
		u, err := NewUnmarshaler(&S{}, schemas)
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 3; i++ {
			var res S
			if err := u.Unmarshal(data, &res); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(res, s) {
				t.Fatalf("expect %v, got %v", s, res)
			}
		}
	}
}

func TestTypeMismatchError(t *testing.T) {
	type T1 struct{}
	type T2 struct{}
	var t1Schema *Schema
	{
		var err error
		t1Schema, err = NewSchema(T1{})
		if err != nil {
			t.Fatal(err)
		}
		m, err := NewMarshaler(T1{}, t1Schema)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := m.Marshal(T2{}); err == nil {
			t.Fatal("expect type mismatch error")
		}
	}
	{
		schemas := map[uint]*Schema{
			1: t1Schema,
		}
		u, err := NewUnmarshaler(&T1{}, schemas)
		if err != nil {
			t.Fatal(err)
		}
		if u.Unmarshal(nil, &T2{}) == nil {
			t.Fatal("expect type mismatch error")
		}
	}
}

/*
func TestRemoveField(t *testing.T) {
	buf := new(bytes.Buffer)
	{
		type T struct {
			I1 int
			I2 int
		}
		v := &T{1, 2}
		schema, err := NewSchema(T{})
		if err != nil {
			t.Fatal(err)
		}
		enc := NewEncoder(buf, schema)
		if err := enc.Encode(v); err != nil {
			t.Fatal(err)
		}
	}
	{
		type T struct {
			I1 int
		}
		v := new(T)
		if err := NewDecoder(buf, T{}).Decode(v); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(v, &T{1}) {
			t.Fatal("got", v)
		}
	}
}

*/
