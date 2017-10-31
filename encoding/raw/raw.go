package raw

import (
	"bytes"
	"io"

	"github.com/alecthomas/binary"
)

var defaultEndian = binary.BigEndian

func Marshal(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	err := NewEncoder(&buf).Encode(v)
	return buf.Bytes(), err
}

func Unmarshal(b []byte, v interface{}) error {
	return NewDecoder(bytes.NewReader(b)).Decode(v)
}

func NewEncoder(w io.Writer) *binary.Encoder {
	enc := binary.NewEncoder(w)
	enc.Order = defaultEndian
	return enc
}

func NewDecoder(r io.Reader) *binary.Decoder {
	dec := binary.NewDecoder(r)
	dec.Order = defaultEndian
	return dec
}
