package runner

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strings"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	payloads := [][]byte{nil, {}, []byte("x"), bytes.Repeat([]byte("ab"), 5000)}
	var buf bytes.Buffer
	for _, p := range payloads {
		if err := writeFrame(&buf, p); err != nil {
			t.Fatalf("writeFrame: %v", err)
		}
	}
	r := newFrameReader(&buf, defaultMaxFrame)
	for i, want := range payloads {
		got, err := r.next()
		if err != nil {
			t.Fatalf("frame %d: %v", i, err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("frame %d: got %d bytes, want %d", i, len(got), len(want))
		}
	}
	if _, err := r.next(); !errors.Is(err, errEOF) {
		t.Fatalf("expected errEOF after the last frame, got %v", err)
	}
}

func TestRefusesOversizedFrame(t *testing.T) {
	var head [4]byte
	binary.LittleEndian.PutUint32(head[:], 64)
	r := newFrameReader(bytes.NewReader(head[:]), 16)
	_, err := r.next()
	if err == nil || !strings.Contains(err.Error(), "announces") {
		t.Fatalf("an oversized frame must be refused before allocating, got %v", err)
	}
}

func TestRefusesTruncatedHeaderAndBody(t *testing.T) {
	t.Run("header", func(t *testing.T) {
		r := newFrameReader(bytes.NewReader([]byte{1, 2}), defaultMaxFrame)
		if _, err := r.next(); err == nil || errors.Is(err, errEOF) {
			t.Fatalf("a truncated header is not a clean end of stream, got %v", err)
		}
	})
	t.Run("body", func(t *testing.T) {
		var head [4]byte
		binary.LittleEndian.PutUint32(head[:], 8)
		r := newFrameReader(bytes.NewReader(append(head[:], 'a', 'b')), defaultMaxFrame)
		if _, err := r.next(); err == nil || errors.Is(err, errEOF) {
			t.Fatalf("a truncated body is not a clean end of stream, got %v", err)
		}
	})
}
