package io

import (
	"io"
)

type PredicateReader struct {
	Seperators  map[byte]bool
	Terminators map[byte]bool
	Whitespace  map[byte]bool
}

type ParseResult struct {
	Predicates []string
	Separators []byte
	Terminator byte
}

func NewPredicateReader(seperators []byte, terminators []byte) *PredicateReader {
	pr := &PredicateReader{
		Seperators:  make(map[byte]bool),
		Terminators: make(map[byte]bool),
		Whitespace:  make(map[byte]bool),
	}
	for _, seperator := range seperators {
		pr.Seperators[seperator] = true
	}
	for _, terminator := range terminators {
		pr.Terminators[terminator] = true
	}
	pr.Whitespace[' '] = true
	pr.Whitespace['\t'] = true
	pr.Whitespace['\n'] = true
	pr.Whitespace['\r'] = true
	return pr
}

func (pr *PredicateReader) SplitInput(reader io.ByteReader) <-chan string {
	result := make(chan string)
	go func() {
		line := make([]byte, 0, 10000)
		inComment := false
		for {
			if b, err := reader.ReadByte(); err != nil {
				close(result)
				return
			} else {
				inComment = inComment || (b == '#')
				if pr.Whitespace[b] || inComment {
					// ignore whitespace and comments
					if b == '\r' || b == '\n' {
						inComment = false
					}
					continue
				}
				line = append(line, b)
				if pr.Terminators[b] || pr.Seperators[b] {
					result <- string(line)
					line = line[:0]
				}
			}
		}
	}()
	return result
}

func (pr *PredicateReader) Parse(reader io.ByteReader) <-chan ParseResult {
	result := make(chan ParseResult)
	go func() {
		nextResult := ParseResult{
			Predicates: make([]string, 0, 5),
			Separators: make([]byte, 0, 5),
			Terminator: 0,
		}
		for line := range pr.SplitInput(reader) {
			sep := line[len(line)-1]
			nextResult.Predicates = append(nextResult.Predicates, line[:len(line)-1])
			if pr.Terminators[sep] {
				nextResult.Terminator = sep
				result <- nextResult
				nextResult = ParseResult{
					Predicates: make([]string, 0, 5),
					Separators: make([]byte, 0, 5),
					Terminator: 0,
				}
			} else if pr.Seperators[sep] {
				nextResult.Separators = append(nextResult.Separators, sep)
			}
		}
		close(result)
	}()
	return result
}
