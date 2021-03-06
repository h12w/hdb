package raw

import (
	"bytes"
	"reflect"
	"testing"
	"time"
)

func TestReg(t *testing.T) {
	type s0 struct {
		A struct {
			B struct {
				C string
			}
			D struct {
				E int
			}
			F bool
		}
		G struct {
			H time.Time
		}
	}
	var v s0
	v.A.B.C = "a"
	v.A.D.E = 2
	v.A.F = true
	v.G.H = time.Date(2017, 1, 2, 3, 4, 5, 6, time.UTC)
	data, err := Marshal(&v)
	if err != nil {
		t.Fatal(err)
	}

	// unmarshal
	{
		var res s0
		if err := Unmarshal(data, &res); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(v, res) {
			t.Fatalf("expect %v, got %v", v, res)
		}
	}

	// regression
	{
		expected := []byte{0x1, 0x61, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2, 0x1, 0xf, 0x1, 0x0, 0x0, 0x0, 0xe, 0xcf, 0xfb, 0xba, 0x25, 0x0, 0x0, 0x0, 0x6, 0xff, 0xff}
		if !bytes.Equal(data, expected) {
			t.Fatalf("expect %v, got %v", expected, data)
		}
	}
}

func TestAddRemove(t *testing.T) {
	type s0 struct {
		A int
	}
	type s1 struct {
		A int
		B int
	}
	// test add a field
	{
		data, err := Marshal(s0{A: 1})
		if err != nil {
			t.Fatal(err)
		}
		var res s0
		if err := Unmarshal(data, &res); err != nil {
			t.Fatal(err)
		}
		if res.A != 1 {
			t.Fatalf("expect 1 but got %d", res.A)
		}
	}
	// test remove a field from the bottom
	// remove a field from the middle is be supported
	// if a field is unused, leave it to its default value
	{
		data, err := Marshal(s1{A: 1, B: 2})
		if err != nil {
			t.Fatal(err)
		}
		var res s1
		if err := Unmarshal(data, &res); err != nil {
			t.Fatal(err)
		}
		if res.A != 1 {
			t.Fatalf("expect 1 but got %d", res.A)
		}
	}
}
