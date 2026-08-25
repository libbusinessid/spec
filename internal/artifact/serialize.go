package artifact

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/protobuf/proto"
)

// GzipLevel is the locked compression level of every published `.gz` artifact.
const GzipLevel = gzip.BestCompression

// Marshal serializes a message with the locked deterministic options.
//
// Determinism here is a property of this pinned compiler, not a universal
// canonical form: engines verify the SHA-256 of the file and never re-serialize
// a decoded message.
func Marshal(m proto.Message) ([]byte, error) {
	return proto.MarshalOptions{Deterministic: true}.Marshal(m)
}

// SHA256Hex returns the lower case hexadecimal SHA-256 of the payload.
func SHA256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Gzip compresses the payload with the locked reproducible settings: fixed
// level, no name, no comment, `mtime` taken from SOURCE_DATE_EPOCH and an OS
// byte fixed to 255 (unknown).
func Gzip(data []byte, sourceDateEpoch int64) ([]byte, error) {
	var buf bytes.Buffer
	// The level is a compile time constant of the package, so the writer can
	// only fail on an invalid level, which cannot happen here.
	w, _ := gzip.NewWriterLevel(&buf, GzipLevel)
	w.Name = ""
	w.Comment = ""
	w.ModTime = epochTime(sourceDateEpoch)
	w.OS = 255
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	out := buf.Bytes()
	if len(out) > 9 {
		// compress/gzip writes the OS byte at offset 9; Go versions differ on
		// whether they honour Writer.OS, so the value is forced here.
		out[9] = 255
	}
	return out, nil
}

// WriteFileAtomic writes the payload through a temporary file in the same
// directory, then renames it. A failure never leaves a partial artifact.
func WriteFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".entidc-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("cannot publish %s: %w", path, err)
	}
	return nil
}

// epochTime converts a SOURCE_DATE_EPOCH value into the time embedded in a
// gzip header.
func epochTime(sourceDateEpoch int64) time.Time {
	return time.Unix(sourceDateEpoch, 0).UTC()
}
