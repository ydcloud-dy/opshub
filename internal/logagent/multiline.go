package logagent

import (
	"regexp"
	"strings"
	"time"
)

var multilineContinuationPatterns = map[string]*regexp.Regexp{
	"java":   regexp.MustCompile(`^(\s+at\s|\s*Caused by:|\s*Suppressed:|\s*\.\.\. \d+ more|\s+)`),
	"go":     regexp.MustCompile(`^(\s|goroutine \d+ \[|created by |panic:|runtime\.|[a-zA-Z0-9_./-]+\([^)]*\))`),
	"python": regexp.MustCompile(`^(\s|Traceback \(most recent call last\):|During handling of the above exception|The above exception was the direct cause|(?:[A-Za-z_][A-Za-z0-9_.]*(?:Error|Exception|Warning|Interrupt|Exit)|ExceptionGroup|BaseExceptionGroup|StopIteration)(?::|$))`),
}

type MultilineAssembler struct {
	config       MultilineConfig
	startPattern *regexp.Regexp
	continuation *regexp.Regexp
	lines        []string
	bytes        int
	lastLineAt   time.Time
}

func NewMultilineAssembler(config MultilineConfig) (*MultilineAssembler, error) {
	config.normalize()
	assembler := &MultilineAssembler{config: config}
	if config.StartPattern != "" {
		pattern, err := regexp.Compile(config.StartPattern)
		if err != nil {
			return nil, err
		}
		assembler.startPattern = pattern
	}
	assembler.continuation = multilineContinuationPatterns[config.Preset]
	return assembler, nil
}

func (assembler *MultilineAssembler) Add(line string, now time.Time) []string {
	if !assembler.config.Enabled {
		return []string{line}
	}
	if len(assembler.lines) == 0 {
		assembler.append(line, now)
		return nil
	}
	shouldStart := assembler.isStart(line)
	wouldOverflow := len(assembler.lines) >= assembler.config.MaxLines || assembler.bytes+len(line)+1 > assembler.config.MaxBytes
	if shouldStart || wouldOverflow {
		flushed := assembler.flush()
		assembler.append(line, now)
		return []string{flushed}
	}
	assembler.append(line, now)
	return nil
}

func (assembler *MultilineAssembler) FlushExpired(now time.Time) []string {
	if len(assembler.lines) == 0 || now.Sub(assembler.lastLineAt) < time.Duration(assembler.config.FlushSeconds)*time.Second {
		return nil
	}
	return []string{assembler.flush()}
}

func (assembler *MultilineAssembler) Flush() []string {
	if len(assembler.lines) == 0 {
		return nil
	}
	return []string{assembler.flush()}
}

func (assembler *MultilineAssembler) Reset() {
	assembler.lines = assembler.lines[:0]
	assembler.bytes = 0
	assembler.lastLineAt = time.Time{}
}

func (assembler *MultilineAssembler) isStart(line string) bool {
	if assembler.startPattern != nil {
		return assembler.startPattern.MatchString(line)
	}
	if assembler.continuation != nil {
		return !assembler.continuation.MatchString(line)
	}
	return true
}

func (assembler *MultilineAssembler) append(line string, now time.Time) {
	assembler.lines = append(assembler.lines, line)
	assembler.bytes += len(line)
	if len(assembler.lines) > 1 {
		assembler.bytes++
	}
	assembler.lastLineAt = now
}

func (assembler *MultilineAssembler) flush() string {
	value := strings.Join(assembler.lines, "\n")
	assembler.lines = assembler.lines[:0]
	assembler.bytes = 0
	assembler.lastLineAt = time.Time{}
	return value
}
