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
	err := NewDecoder(bytes.NewReader(b)).Decode(v)
	if err == io.EOF {
		return nil
	}
	return err
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
