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
	"sync"
)

type (
	Marshaler struct {
		schema *Schema
		enc    *encoder
		mu     sync.Mutex
	}
	Unmarshaler struct {
		schema *Schema
		dec    *decoder
	}
	Schema struct {
		v    interface{}
		typ  reflect.Type
		data []byte
		ver  uint
	}

	encoder struct {
		*gob.Encoder
		*switchWriter
	}
	decoder struct {
		*switchReader
		*gob.Decoder
	}
	switchWriter struct {
		w io.Writer
	}
	switchReader struct {
		r io.Reader
	}
)

// NewMarshaler creates a new Marshaler for type of v
func NewMarshaler(schema *Schema) (*Marshaler, error) {
	enc, err := schema.newEncoder()
	if err != nil {
		return nil, err
	}
	return &Marshaler{
		schema: schema,
		enc:    enc,
	}, nil
}

// Marshal marshals v into []byte and returns the result
func (m *Marshaler) Marshal(v interface{}) ([]byte, error) {
	if err := m.schema.check(v); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if _, err := m.schema.encodeVersion(&buf); err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.enc.switchTo(&buf)
	if err := m.enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func NewUnmarshaler(schema *Schema) (*Unmarshaler, error) {
	dec, err := schema.newDecoder()
	if err != nil {
		return nil, err
	}
	return &Unmarshaler{
		schema: schema,
		dec:    dec,
	}, nil
}

func (u *Unmarshaler) Unmarshal(data []byte, v interface{}) error {
	if err := u.schema.check(v); err != nil {
		return err
	}
	r := bytes.NewReader(data)
	ver, err := binary.ReadUvarint(r)
	if err != nil {
		return err
	}
	if uint64(u.schema.ver) != ver {
		return errors.New("schema version mismatch")
	}
	u.dec.switchTo(r)
	return u.dec.Decode(v)
}

func NewSchema(v interface{}, data []byte) (*Schema, error) {
	var schemaBuf bytes.Buffer
	sw := newSwitchWriter(&schemaBuf)
	enc := gob.NewEncoder(sw)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	if data == nil {
		data = schemaBuf.Bytes()
	}
	return &Schema{
		v:    v,
		typ:  getType(v),
		data: data,
	}, nil
}

func (s *Schema) newEncoder() (*encoder, error) {
	var schemaBuf bytes.Buffer
	sw := newSwitchWriter(&schemaBuf)
	enc := gob.NewEncoder(sw)
	if err := enc.Encode(s.v); err != nil {
		return nil, err
	}
	return &encoder{
		Encoder:      enc,
		switchWriter: sw,
	}, nil
}

func (s *Schema) newDecoder() (*decoder, error) {
	sr := newSwitchReader(bytes.NewReader(s.data))
	dec := gob.NewDecoder(sr)
	if err := dec.Decode(reflect.New(s.typ).Interface()); err != nil {
		return nil, err
	}
	return &decoder{
		switchReader: sr,
		Decoder:      dec,
	}, nil
}

func (s *Schema) Bytes() []byte {
	return s.data
}

// SetVersion of the schema, a valid version should starts from 1
func (s *Schema) SetVersion(version uint) {
	s.ver = version
}

func (s *Schema) check(v interface{}) error {
	if s.ver == 0 {
		return errors.New("schema version is not set")
	}
	if t := getType(v); t != s.typ {
		return fmt.Errorf("expect type %v but got %v", s.typ, t)
	}
	return nil
}

func (s *Schema) encodeVersion(w io.Writer) (int, error) {
	buf := make([]byte, binary.MaxVarintLen64)
	return w.Write(buf[:binary.PutUvarint(buf, uint64(s.ver))])
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
