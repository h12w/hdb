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
		typ    reflect.Type
		enc    *encoder
		mu     sync.Mutex
	}
	Unmarshaler struct {
		typ  reflect.Type
		decs map[uint]*decoder
		mu   sync.Mutex
	}
	Schema struct {
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
func NewMarshaler(v interface{}) (*Marshaler, error) {
	enc, err := newEncoder(v)
	if err != nil {
		return nil, err
	}
	schemaData, err := encodeBytes(v)
	if err != nil {
		return nil, err
	}
	return &Marshaler{
		schema: NewSchema(schemaData, 0),
		enc:    enc,
		typ:    getType(v),
	}, nil
}

func (m *Marshaler) SetVersion(version uint) {
	m.schema.setVersion(version)
}

func (m *Marshaler) Schema() *Schema {
	return m.schema
}

// Marshal marshals v into []byte and returns the result
func (m *Marshaler) Marshal(v interface{}) ([]byte, error) {
	if t := getType(v); t != m.typ {
		return nil, fmt.Errorf("expect type %v but got %v", m.typ, t)
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

func NewUnmarshaler(v interface{}, schemas map[uint]*Schema) (*Unmarshaler, error) {
	decs := make(map[uint]*decoder)
	for ver, schema := range schemas {
		dec, err := newDecoder(v, schema.Bytes())
		if err != nil {
			return nil, err
		}
		decs[ver] = dec
	}
	return &Unmarshaler{
		typ:  getType(v),
		decs: decs,
	}, nil
}

func (u *Unmarshaler) Unmarshal(data []byte, v interface{}) error {
	if t := getType(v); t != u.typ {
		return fmt.Errorf("expect type %v but got %v", u.typ, t)
	}
	r := bytes.NewReader(data)
	ver, err := binary.ReadUvarint(r)
	if err != nil {
		return err
	}
	dec, ok := u.decs[uint(ver)]
	if !ok {
		return errors.New("missing dec for the version")
	}

	u.mu.Lock()
	defer u.mu.Unlock()
	dec.switchTo(r)
	return dec.Decode(v)
}

func NewSchema(data []byte, version uint) *Schema {
	return &Schema{
		data: data,
		ver:  version,
	}
}

// SetVersion of the schema, a valid version should starts from 1
func (s *Schema) setVersion(version uint) {
	s.ver = version
}

func (s *Schema) encodeVersion(w io.Writer) (int, error) {
	if s.ver == 0 {
		return 0, errors.New("schema version is not set")
	}
	buf := make([]byte, binary.MaxVarintLen64)
	return w.Write(buf[:binary.PutUvarint(buf, uint64(s.ver))])
}

func (s *Schema) Bytes() []byte {
	return s.data
}

func newEncoder(v interface{}) (*encoder, error) {
	var schemaBuf bytes.Buffer
	sw := newSwitchWriter(&schemaBuf)
	enc := gob.NewEncoder(sw)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return &encoder{
		Encoder:      enc,
		switchWriter: sw,
	}, nil
}

func newDecoder(v interface{}, schemaData []byte) (*decoder, error) {
	sr := newSwitchReader(bytes.NewReader(schemaData))
	dec := gob.NewDecoder(sr)
	if err := dec.Decode(v); err != nil {
		return nil, err
	}
	return &decoder{
		switchReader: sr,
		Decoder:      dec,
	}, nil
}
func encodeBytes(v interface{}) ([]byte, error) {
	var schemaBuf bytes.Buffer
	if err := gob.NewEncoder(&schemaBuf).Encode(v); err != nil {
		return nil, err
	}
	return schemaBuf.Bytes(), nil
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
