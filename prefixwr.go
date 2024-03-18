package gomk

import (
	"bytes"
	"io"
)

type prefixWriter struct {
	w      io.Writer
	prefix []byte
	inLine bool // not at start of line (zero…)
}

func newPrefixWriter(w io.Writer, prefix []byte) *prefixWriter {
	return &prefixWriter{w: w, prefix: prefix}
}

func newPrefixWriterString(w io.Writer, prefix string) *prefixWriter {
	return &prefixWriter{w: w, prefix: []byte(prefix)}
}

func (pw *prefixWriter) Reset() { pw.inLine = false }

func (pw *prefixWriter) Write(p []byte) (n int, err error) {
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
