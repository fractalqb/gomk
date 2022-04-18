package gomk

import (
	"log"
	"path/filepath"
)

type Tracer interface {
	Trace(dir *Dir, a ...interface{})
	Tracef(dir *Dir, format string, a ...interface{})
}

func TraceStart(t Tracer, dir *Dir, a ...interface{}) {
	t.Trace(dir, append([]interface{}{"⏵ start: "}, a...)...)
}

func TracefStart(t Tracer, dir *Dir, format string, a ...interface{}) {
	t.Tracef(dir, "⏵ start: "+format, a...)
}

func TraceDone(t Tracer, dir *Dir, a ...interface{}) {
	t.Trace(dir, append([]interface{}{"▪ done: "}, a...)...)
}

func TracefDone(t Tracer, dir *Dir, format string, a ...interface{}) {
	t.Tracef(dir, "▪ done: "+format, a...)
}

func TraceFail(t Tracer, err error, dir *Dir, a ...interface{}) {
	a = append([]interface{}{"↯ error: "}, a...)
	a = append(a, " ↯[", err, "]")
	t.Trace(dir, a...)
}

func TracefFail(t Tracer, err error, dir *Dir, format string, a ...interface{}) {
	t.Tracef(dir, "↯ error: "+format+" ↯[%s]", a...)
}

const LogTracer = logTracer(0)

type logTracer int

func (logTracer) Trace(dir *Dir, a ...interface{}) {
	d := dir.Prj().Name() + "@" + tarceWDir(dir) + ": "
	log.Print(append([]interface{}{d}, a...)...)
}

func (logTracer) Tracef(dir *Dir, format string, a ...interface{}) {
	d := dir.Prj().Name() + "@" + tarceWDir(dir) + ": "
	log.Printf(d+format, a...)
}

func tarceWDir(dir *Dir) string {
	pdir := dir.Prj().RootDir.Abs()
	cwd := dir.Abs()
	res, _ := filepath.Rel(pdir, cwd)
	return res
}
