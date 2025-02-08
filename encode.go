package govdf

import (
	"bytes"
	"fmt"
	"io"

	"github.com/lewisgibson/go-vdf/internal"
)

// Marshaler is the interface implemented by types that can marshal themselves into a VDF description.
type Marshaler interface {
	MarshalVDF() ([]byte, error)
}

// Marshal returns the VDF encoding of v.
func Marshal(in any) ([]byte, error) {
	var encodeBuffer = getEncodeBuffer()
	defer encodeBufferPool.Put(encodeBuffer)
	if err := NewEncoder(encodeBuffer).Encode(in); err != nil {
		return nil, err
	}
	return encodeBuffer.Bytes(), nil
}

// Encoder writes VDF values to an output stream.
type Encoder struct {
	w io.Writer
}

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		w: w,
	}
}

// Encode writes the VDF encoding of v to the stream.
func (e *Encoder) Encode(v any) error {
	return fmt.Errorf("not implemented")
}

// encodeBuffer is a buffer used by the encoder.
type encodeBuffer struct {
	bytes.Buffer // The buffer used to write the VDF encoding.
}

// encodeBufferPool is a pool of encodeBuffer.
//
// This pool is used to reduce the number of allocations of encodeBuffer.
var encodeBufferPool internal.SyncPool[encodeBuffer]

// getEncodeBuffer returns an encodeBuffer from the pool.
// If the pool is empty, a new encodeBuffer is created.
func getEncodeBuffer() *encodeBuffer {
	var existing = encodeBufferPool.Get()
	if existing != nil {
		existing.Reset()
		return existing
	}
	return &encodeBuffer{}
}
