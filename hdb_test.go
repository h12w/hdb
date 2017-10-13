package hdb

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"testing"

	"h12.me/hdb/colfer"
)

var (
	testValue = &TestStruct{
		S1: S{1, 2, 3, 4, 5},
		S2: S{1, 2, 3, 4, 5},
		S3: S{1, 2, 3, 4, 5},
		S4: S{1, 2, 3, 4, 5},
		S5: S{1, 2, 3, 4, 5},
	}
	testColferValue = &colfer.T{
		S1: &colfer.S{1, 2, 3, 4, 5},
		S2: &colfer.S{1, 2, 3, 4, 5},
		S3: &colfer.S{1, 2, 3, 4, 5},
		S4: &colfer.S{1, 2, 3, 4, 5},
		S5: &colfer.S{1, 2, 3, 4, 5},
	}
)

func TestGob(t *testing.T) {
	w := new(bytes.Buffer)
	enc := gob.NewEncoder(w)
	if err := enc.Encode(testValue); err != nil {
		t.Fatal(err)
	}
	l := w.Len()
	if err := enc.Encode(testValue); err != nil {
		t.Fatal(err)
	}
	fmt.Println(w.Len() - l)
}

func TestColfer(t *testing.T) {
	l, err := testColferValue.MarshalLen()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(l)
}

func TestVariantInt(t *testing.T) {
	l := binary.PutUvarint(make([]byte, 10), 5)
	fmt.Println(l)
}
