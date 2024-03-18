package gomk

import (
	"bytes"
	"io"
	"math/rand/v2"
	"os"
	"strings"
	"testing"
)

func Example_prefixWriter() {
	pw := newPrefixWriterString(os.Stdout, "PRE:")
	io.WriteString(pw, "foo")
	io.WriteString(pw, "bar\n")
	io.WriteString(pw, "baz\nquux")
	// Output:
	// PRE:foobar
	// PRE:baz
	// PRE:quux
}

const (
	benchLineMin = 40
	benchLineMax = 180
)

var benchLine = append([]byte("test"), strings.Repeat(" test", (benchLineMax-4)/5+1)...)

func BenchmarkWrite(b *testing.B) {
	var c byte
	var buf bytes.Buffer
	randN := benchLineMax - benchLineMin
	for range b.N {
		buf.Reset()
		lno := 5 + rand.IntN(15)
		for range lno {
			len := benchLineMin + rand.IntN(randN)
			c, benchLine[len-1] = benchLine[len-1], '\n'
			buf.Write(benchLine[:len])
			benchLine[len-1] = c
		}
	}
}

func BenchmarkPrefixWriter(b *testing.B) {
	var c byte
	var buf bytes.Buffer
	pfw := newPrefixWriterString(&buf, "Prefix:")
	randN := benchLineMax - benchLineMin
	for range b.N {
		buf.Reset()
		pfw.Reset()
		lno := 5 + rand.IntN(15)
		for range lno {
			len := benchLineMin + rand.IntN(randN)
			c, benchLine[len-1] = benchLine[len-1], '\n'
			pfw.Write(benchLine[:len])
			benchLine[len-1] = c
		}
	}
}
