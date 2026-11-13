package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// pair is one labelled example: same reports whether the two texts mean the same thing.
type pair struct {
	first  string
	second string
	same   bool
}

// readPairs parses tab-separated rows of id, sentence1, sentence2, label with a header row.
// Label 1 means the two sentences share a meaning and 0 means they do not.
func readPairs(r io.Reader, limit int) ([]pair, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	var pairs []pair
	for line := 1; scanner.Scan(); line++ {
		if line == 1 || strings.TrimSpace(scanner.Text()) == "" {
			continue
		}
		fields := strings.Split(strings.TrimRight(scanner.Text(), "\r"), "\t")
		if len(fields) != 4 {
			return nil, fmt.Errorf("line %d: expected 4 tab-separated fields, got %d", line, len(fields))
		}
		label := strings.TrimSpace(fields[3])
		if label != "0" && label != "1" {
			return nil, fmt.Errorf("line %d: label must be 0 or 1, got %q", line, label)
		}
		pairs = append(pairs, pair{first: fields[1], second: fields[2], same: label == "1"})
		if limit > 0 && len(pairs) == limit {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(pairs) == 0 {
		return nil, fmt.Errorf("no pairs found")
	}
	return pairs, nil
}
