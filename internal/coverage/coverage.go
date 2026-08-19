// Package coverage parses a Go coverage profile and enforces the line and
// block coverage thresholds of the repository.
//
// Go does not report branch coverage directly. The profile lists basic blocks
// with their statement count, and every conditional branch of a function starts
// a new basic block; the ratio of covered blocks is therefore a conservative
// proxy for branch coverage, and it is reported as such.
package coverage

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// Block is one basic block of a coverage profile.
type Block struct {
	Statements int
	Count      int
}

// Summary is the aggregated result of a coverage profile.
type Summary struct {
	LineCoverage      float64            `json:"lineCoverage"`
	BlockCoverage     float64            `json:"blockCoverage"`
	TotalStatements   int                `json:"totalStatements"`
	CoveredStatements int                `json:"coveredStatements"`
	TotalBlocks       int                `json:"totalBlocks"`
	CoveredBlocks     int                `json:"coveredBlocks"`
	WorstFiles        []FileCoverage     `json:"worstFiles"`
	PerFile           map[string]float64 `json:"perFile"`
}

// FileCoverage is the coverage of one file.
type FileCoverage struct {
	File     string  `json:"file"`
	Coverage float64 `json:"coverage"`
}

// ParseLine parses one profile line `name:startLine.startCol,endLine.endCol numStmt count`
// and returns the file name, the block range and the block.
func ParseLine(line string) (string, string, Block, bool) {
	idx := strings.LastIndex(line, ":")
	if idx < 0 {
		return "", "", Block{}, false
	}
	name := line[:idx]
	fields := strings.Fields(line[idx+1:])
	if len(fields) != 3 {
		return "", "", Block{}, false
	}
	statements, err := strconv.Atoi(fields[1])
	if err != nil || statements < 0 {
		return "", "", Block{}, false
	}
	count, err := strconv.Atoi(fields[2])
	if err != nil || count < 0 {
		return "", "", Block{}, false
	}
	return name, fields[0], Block{Statements: statements, Count: count}, true
}

// Parse reads a whole coverage profile and aggregates it.
//
// With `-coverpkg`, the same basic block is reported once per test binary. The
// blocks are therefore merged by (file, range) and the highest execution count
// wins, exactly like `go tool cover` does for a merged profile.
func Parse(r io.Reader) (Summary, error) {
	type blockKey struct{ file, span string }
	merged := map[blockKey]Block{}
	order := map[string][]blockKey{}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024*1024), 8*1024*1024)
	first := true
	for scanner.Scan() {
		line := scanner.Text()
		if first {
			first = false
			if strings.HasPrefix(line, "mode:") {
				continue
			}
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		name, span, b, ok := ParseLine(line)
		if !ok {
			return Summary{}, fmt.Errorf("malformed profile line: %q", line)
		}
		key := blockKey{name, span}
		previous, seen := merged[key]
		if !seen {
			order[name] = append(order[name], key)
			merged[key] = b
			continue
		}
		if b.Count > previous.Count {
			previous.Count = b.Count
		}
		merged[key] = previous
	}
	if err := scanner.Err(); err != nil {
		return Summary{}, err
	}
	perFile := make(map[string][]Block, len(order))
	for name, keys := range order {
		blocks := make([]Block, 0, len(keys))
		for _, key := range keys {
			blocks = append(blocks, merged[key])
		}
		perFile[name] = blocks
	}
	return aggregate(perFile), nil
}

func aggregate(perFile map[string][]Block) Summary {
	s := Summary{PerFile: map[string]float64{}}
	for name, blocks := range perFile {
		var total, covered int
		for _, b := range blocks {
			total += b.Statements
			s.TotalBlocks++
			if b.Count > 0 {
				covered += b.Statements
				s.CoveredBlocks++
			}
		}
		s.TotalStatements += total
		s.CoveredStatements += covered
		if total > 0 {
			s.PerFile[name] = 100 * float64(covered) / float64(total)
		}
	}
	if s.TotalStatements > 0 {
		s.LineCoverage = 100 * float64(s.CoveredStatements) / float64(s.TotalStatements)
	}
	if s.TotalBlocks > 0 {
		s.BlockCoverage = 100 * float64(s.CoveredBlocks) / float64(s.TotalBlocks)
	}
	names := make([]string, 0, len(s.PerFile))
	for name := range s.PerFile {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		if s.PerFile[names[i]] != s.PerFile[names[j]] {
			return s.PerFile[names[i]] < s.PerFile[names[j]]
		}
		return names[i] < names[j]
	})
	for i, name := range names {
		if i >= 10 {
			break
		}
		s.WorstFiles = append(s.WorstFiles, FileCoverage{File: name, Coverage: s.PerFile[name]})
	}
	return s
}

// Report renders the summary and returns the failures against the thresholds.
func (s Summary) Report(w io.Writer, lineMin, blockMin float64) []string {
	_, _ = fmt.Fprintf(w, "line coverage   %.2f%% (%d/%d statements)\n", s.LineCoverage, s.CoveredStatements, s.TotalStatements)
	_, _ = fmt.Fprintf(w, "block coverage  %.2f%% (%d/%d blocks)\n", s.BlockCoverage, s.CoveredBlocks, s.TotalBlocks)
	_, _ = fmt.Fprintln(w, "lowest covered files:")
	for _, f := range s.WorstFiles {
		_, _ = fmt.Fprintf(w, "  %-70s %6.2f%%\n", f.File, f.Coverage)
	}
	var failures []string
	if s.LineCoverage < lineMin {
		failures = append(failures, fmt.Sprintf("line coverage %.2f%% is below the %.2f%% threshold", s.LineCoverage, lineMin))
	}
	if s.BlockCoverage < blockMin {
		failures = append(failures, fmt.Sprintf("block coverage %.2f%% is below the %.2f%% threshold", s.BlockCoverage, blockMin))
	}
	return failures
}
