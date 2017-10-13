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
	var data, sSchema []byte
	{
		m, err := NewMarshaler(S{})
		if err != nil {
			t.Fatal(err)
		}
		m.SetVersion(version)
		sSchema = m.Schema()
		data, err = m.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
	}
	{
		u, err := NewUnmarshaler(&S{}, sSchema, 1)
		if err != nil {
			t.Fatal(err)
		}
		var res S
		if err := u.Unmarshal(data, &res); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(res, s) {
			t.Fatalf("expect %v, got %v", s, res)
		}
	}
}

func TestTypeMismatchError(t *testing.T) {
	type T1 struct{}
	type T2 struct{}
	var t1Schema []byte
	{
		m, err := NewMarshaler(T1{})
		if err != nil {
			t.Fatal(err)
		}
		t1Schema = m.Schema()
		if _, err := m.Marshal(T2{}); err == nil {
			t.Fatal("expect type mismatch error")
		}
	}
	{
		u, err := NewUnmarshaler(&T1{}, t1Schema, 1)
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
