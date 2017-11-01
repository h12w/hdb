// vgob is versioned encoding/gob
package vgob

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"reflect"
)

type (
	SchemaStore struct {
		schemas schemas
		file    string
	}
	Marshaler struct {
		ver uint
		enc *encoder
	}
	Unmarshaler struct {
		decs map[uint]*decoder
	}
)

type (
	schemas map[string]*schema
	schema  struct {
		Versions schemaVersions
		typ      reflect.Type // unmarshaled in gob
	}
	schemaVersions map[string]uint
)

func NewSchemaStore(file string) (*SchemaStore, error) {
	s := &SchemaStore{file: file, schemas: make(schemas)}
	f, err := os.Open(file)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		return s, nil
	}
	defer f.Close()
	if err := gob.NewDecoder(f).Decode(&s.schemas); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *SchemaStore) RegisterName(name string, v interface{}) error {
	typ := getType(v)
	schemaBytes, err := encodeSchema(typ)
	if err != nil {
		return err
	}
	schemaStr := string(schemaBytes)
	if schema, schemaExists := s.schemas[name]; schemaExists {
		schema.typ = typ
		if schema.Versions[schemaStr] == 0 {
			schema.Versions[schemaStr] = uint(len(schema.Versions)) + 1
		}
		return nil
	}
	s.schemas[name] = &schema{
		typ: typ,
		Versions: schemaVersions{
			schemaStr: 1,
		},
	}
	return nil
}

func (s *SchemaStore) Save() error {
	tmpfile := s.file + ".tmp"
	f, err := os.Create(tmpfile)
	if err != nil {
		return nil
	}
	if err := gob.NewEncoder(f).Encode(&s.schemas); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmpfile, s.file)
}

// NewMarshaler creates a new Marshaler for type of v
func (s *SchemaStore) NewMarshaler(name string) (*Marshaler, error) {
	schema, ok := s.schemas[name]
	if !ok {
		return nil, fmt.Errorf("schema for %s is not registered", name)
	}
	if schema.typ == nil {
		return nil, fmt.Errorf("type %s not registered", name)
	}

	enc, err := newEncoder(schema.typ)
	if err != nil {
		return nil, err
	}
	return &Marshaler{
		enc: enc,
		ver: uint(len(schema.Versions)),
	}, nil
}

func (s *SchemaStore) NewUnmarshaler(name string) (*Unmarshaler, error) {
	schema, ok := s.schemas[name]
	if !ok {
		return nil, fmt.Errorf("schema for %s is not registered", name)
	}
	if schema.typ == nil {
		return nil, fmt.Errorf("type %s not registered", name)
	}

	decs := make(map[uint]*decoder)
	for schemaData, version := range schema.Versions {
		dec, err := newDecoder(schema.typ, []byte(schemaData))
		if err != nil {
			return nil, err
		}
		decs[version] = dec
	}
	return &Unmarshaler{
		decs: decs,
	}, nil
}

// Marshal marshals v into []byte and returns the result
func (m *Marshaler) Marshal(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if _, err := encodeVersion(&buf, m.ver); err != nil {
		return nil, err
	}
	if err := m.enc.encode(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (u *Unmarshaler) Unmarshal(data []byte, v interface{}) error {
	r := bytes.NewReader(data)
	ver, err := binary.ReadUvarint(r)
	if err != nil {
		return err
	}
	dec, ok := u.decs[uint(ver)]
	if !ok {
		return errors.New("missing dec for the version")
	}
	return dec.decode(r, v)
}
