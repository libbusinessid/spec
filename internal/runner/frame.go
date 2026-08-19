// Package runner executes the conformance corpus against an external engine.
//
// The runner is the only program that reads expected results. A testee answers
// requests without ever seeing an expectation, so it cannot declare itself
// conformant by comparing too weakly.
package runner

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// defaultMaxFrame bounds a single message. The largest legitimate payload is a
// bundle fixture, itself bounded by the 16 MiB bundle limit of the
// specification; the margin covers the surrounding message.
const defaultMaxFrame = 32 << 20

// errEOF reports a clean end of stream, on a frame boundary.
var errEOF = errors.New("end of stream")

// writeFrame emits one length prefixed message.
func writeFrame(w io.Writer, payload []byte) error {
	var head [4]byte
	binary.LittleEndian.PutUint32(head[:], uint32(len(payload)))
	if _, err := w.Write(head[:]); err != nil {
		return err
	}
	if len(payload) == 0 {
		return nil
	}
	_, err := w.Write(payload)
	return err
}

// frameReader decodes length prefixed messages from an untrusted producer.
type frameReader struct {
	r        io.Reader
	maxFrame uint32
}

func newFrameReader(r io.Reader, maxFrame uint32) *frameReader {
	return &frameReader{r: r, maxFrame: maxFrame}
}

// next returns the next payload, errEOF at a clean end of stream, and a
// diagnostic error otherwise.
//
// The announced length is checked against the limit before any allocation, so
// a hostile testee cannot make the runner reserve memory it will never fill.
func (f *frameReader) next() ([]byte, error) {
	var head [4]byte
	switch _, err := io.ReadFull(f.r, head[:]); {
	case errors.Is(err, io.EOF):
		return nil, errEOF
	case err != nil:
		return nil, fmt.Errorf("cannot read the frame header: %w", err)
	}
	size := binary.LittleEndian.Uint32(head[:])
	if size > f.maxFrame {
		return nil, fmt.Errorf("the testee announces a frame of %d bytes, above the %d byte limit", size, f.maxFrame)
	}
	if size == 0 {
		return []byte{}, nil
	}
	payload := make([]byte, size)
	if _, err := io.ReadFull(f.r, payload); err != nil {
		return nil, fmt.Errorf("cannot read a frame of %d bytes: %w", size, err)
	}
	return payload, nil
}
