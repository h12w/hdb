// vgob is versioned encoding/gob
package vgob

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"reflect"
)

type (
	Marshaler struct {
		sw     *switchWriter
		schema []byte
		enc    *gob.Encoder
		ver    gobVer
	}
	Unmarshaler struct {
		sr     *switchReader
		schema []byte
		dec    *gob.Decoder
		ver    gobVer
	}
	gobVer struct {
		typ reflect.Type
		ver uint
	}
	switchWriter struct {
		w io.Writer
	}
	switchReader struct {
		r io.Reader
	}
)

// NewMarshaler creates a new Marshaler for type of v
func NewMarshaler(v interface{}) (*Marshaler, error) {
	var schemaBuf bytes.Buffer
	sw := newSwitchWriter(&schemaBuf)
	enc := gob.NewEncoder(sw)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return &Marshaler{
		sw:     sw,
		schema: schemaBuf.Bytes(),
		enc:    enc,
		ver:    gobVer{typ: getType(v)},
	}, nil
}

// Marshal marshals v into []byte and returns the result
func (m *Marshaler) Marshal(v interface{}) ([]byte, error) {
	if err := m.ver.check(v); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	buf.Write(m.ver.bytes())
	m.sw.switchTo(&buf)
	if err := m.enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *Marshaler) Schema() []byte {
	return s.schema
}

func NewUnmarshaler(v interface{}, schema []byte, version uint) (*Unmarshaler, error) {
	sr := newSwitchReader(bytes.NewReader(schema))
	dec := gob.NewDecoder(sr)
	if err := dec.Decode(v); err != nil {
		return nil, err
	}
	return &Unmarshaler{
		sr:  sr,
		dec: dec,
		ver: gobVer{typ: getType(v), ver: version},
	}, nil
}

func (u *Unmarshaler) Unmarshal(data []byte, v interface{}) error {
	if err := u.ver.check(v); err != nil {
		return err
	}
	r := bytes.NewReader(data)
	if _, err := binary.ReadUvarint(r); err != nil {
		return err
	}
	// TODO: check read version
	u.sr.switchTo(r)
	return u.dec.Decode(v)
}

func (s *gobVer) check(v interface{}) error {
	if s.ver == 0 {
		return errors.New("schema version is not set")
	}
	if t := getType(v); t != s.typ {
		return fmt.Errorf("expect type %v but got %v", s.typ, t)
	}
	return nil
}

func (s *gobVer) bytes() []byte {
	buf := make([]byte, binary.MaxVarintLen64)
	return buf[:binary.PutUvarint(buf, uint64(s.ver))]
}

// SetVersion for the schema, a valid version should starts from 1
func (s *Marshaler) SetVersion(version uint) {
	s.ver.ver = version
}

func getType(v interface{}) reflect.Type {
	t := reflect.TypeOf(v)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func newSwitchWriter(w io.Writer) *switchWriter {
	return &switchWriter{w: w}
}

func (s *switchWriter) Write(data []byte) (int, error) {
	return s.w.Write(data)
}

func (s *switchWriter) switchTo(w io.Writer) {
	s.w = w
}

func newSwitchReader(r io.Reader) *switchReader {
	return &switchReader{r: r}
}

func (s *switchReader) switchTo(r io.Reader) {
	s.r = r
}

func (s *switchReader) Read(data []byte) (int, error) {
	return s.r.Read(data)
}
