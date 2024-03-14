package gomkore

import (
	"bytes"
	"io"
)

type PrefixWriter struct {
	w      io.Writer
	prefix []byte
	inLine bool // not at start of line (zero…)
}

func NewPrefixWriter(w io.Writer, prefix []byte) *PrefixWriter {
	return &PrefixWriter{w: w, prefix: prefix}
}

func NewPrefixWriterString(w io.Writer, prefix string) *PrefixWriter {
	return &PrefixWriter{w: w, prefix: []byte(prefix)}
}

func (pw *PrefixWriter) Reset() { pw.inLine = false }

func (pw *PrefixWriter) Write(p []byte) (n int, err error) {
	for len(p) > 0 {
		nlIdx := bytes.IndexByte(p, '\n')
		if nlIdx < 0 {
			if !pw.inLine {
				if _, err := pw.w.Write(pw.prefix); err != nil {
					return n, err
				}
			}
			pw.inLine = true
			m, err := pw.w.Write(p)
			return n + m, err
		}
		if !pw.inLine {
			if _, err := pw.w.Write(pw.prefix); err != nil {
				return n, err
			}
		}
		nlIdx++
		if m, err := pw.w.Write(p[:nlIdx]); err != nil {
			return n + m, err
		} else {
			n += m
		}
		pw.inLine = false
		p = p[nlIdx:]
	}
	return n, nil
}
