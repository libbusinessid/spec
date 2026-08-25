// Package artifact computes the canonical source digest, validates a rule
// bundle defensively, serializes artifacts reproducibly and renders the release
// manifest.
package artifact

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

// Domain separators of the canonical source streams.
const (
	RulesDomain       = "ENTID-SOURCE-V1\n"
	ConformanceDomain = "ENTID-CONFORMANCE-SOURCE-V1\n"
)

// SourceEntry is one file of a canonical source stream. Path is the virtual
// path, independent from the checkout, and Content holds the raw bytes read
// from disk before normalization.
//
// Binary marks an entry whose bytes carry no text encoding: an embedded decoder
// fixture is incorporated verbatim, because normalizing its line endings would
// corrupt it. The framing of the stream is identical for both kinds of entry.
// See docs/normative-decisions.md, decision ND-001.
type SourceEntry struct {
	Path    string
	Content []byte
	Binary  bool
}

// SourceDigest returns the SHA-256 of the canonical stream of the entries.
//
// Each entry contributes the big-endian unsigned 64 bit length of its path, the
// UTF-8 path, the big-endian unsigned 64 bit length of its normalized content
// and the content. No implicit separator is added.
func SourceDigest(domain string, entries []SourceEntry) ([32]byte, error) {
	normalized := make([]SourceEntry, 0, len(entries))
	seen := make(map[string]bool, len(entries))
	for _, e := range entries {
		path, err := virtualPath(e.Path)
		if err != nil {
			return [32]byte{}, err
		}
		if seen[path] {
			return [32]byte{}, fmt.Errorf("%s: duplicate path in the canonical stream", path)
		}
		seen[path] = true
		content := e.Content
		if !e.Binary {
			var err error
			content, err = normalizeContent(path, e.Content)
			if err != nil {
				return [32]byte{}, err
			}
		}
		normalized = append(normalized, SourceEntry{Path: path, Content: content, Binary: e.Binary})
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i].Path < normalized[j].Path })

	h := sha256.New()
	_, _ = h.Write([]byte(domain))
	var length [8]byte
	for _, e := range normalized {
		binary.BigEndian.PutUint64(length[:], uint64(len(e.Path)))
		_, _ = h.Write(length[:])
		_, _ = h.Write([]byte(e.Path))
		binary.BigEndian.PutUint64(length[:], uint64(len(e.Content)))
		_, _ = h.Write(length[:])
		_, _ = h.Write(e.Content)
	}
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out, nil
}

// virtualPath validates and normalizes one virtual path of the stream.
func virtualPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path in the canonical stream")
	}
	if !utf8.ValidString(path) {
		return "", fmt.Errorf("path is not valid UTF-8")
	}
	if strings.ContainsRune(path, '\\') {
		return "", fmt.Errorf("%s: only POSIX separators are accepted", path)
	}
	if strings.HasPrefix(path, "/") {
		return "", fmt.Errorf("%s: absolute paths are refused", path)
	}
	for _, segment := range strings.Split(path, "/") {
		switch segment {
		case "", ".", "..":
			return "", fmt.Errorf("%s: refused path segment %q", path, segment)
		}
	}
	return path, nil
}

// normalizeContent rejects a BOM and non UTF-8 bytes, then maps CRLF and CR to
// LF without removing comments or spaces.
func normalizeContent(path string, content []byte) ([]byte, error) {
	if !utf8.Valid(content) {
		return nil, fmt.Errorf("%s: content is not valid UTF-8", path)
	}
	if len(content) >= 3 && content[0] == 0xEF && content[1] == 0xBB && content[2] == 0xBF {
		return nil, fmt.Errorf("%s: content starts with a UTF-8 byte order mark", path)
	}
	out := make([]byte, 0, len(content))
	for i := 0; i < len(content); i++ {
		switch {
		case content[i] == '\r' && i+1 < len(content) && content[i+1] == '\n':
			out = append(out, '\n')
			i++
		case content[i] == '\r':
			out = append(out, '\n')
		default:
			out = append(out, content[i])
		}
	}
	return out, nil
}
